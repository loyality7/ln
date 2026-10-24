// Package stage is the engine layer: build a scene graph out of shapes, script
// a timeline of motion in seconds, and render stills or video from one object.
package stage

import (
	"sync"
	"sync/atomic"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/ink"
	"github.com/loyality7/ln/ln/model"
	"github.com/loyality7/ln/ln/shapes"
	"github.com/loyality7/ln/ln/solid"
)

// Node is a thing in the scene: a group (just a transform) or a mesh. Children
// move, turn and scale with their parent, so an arm hangs off a shoulder that
// hangs off a body.
type Node struct {
	Name     string
	Pos      geom.Vector // position relative to parent
	Rot      geom.Vector // euler angles in radians, applied X then Y then Z
	Scl      geom.Vector // scale, default (1,1,1)
	Parent   *Node
	Children []*Node

	mesh    *model.Mesh
	shape   shapes.Shape // an analytic shape (water, plane) instead of a mesh
	id      int          // object tag the renderers use to find this node's material
	merge   bool         // this node and everything under it is drawn as one object
	gid     int          // that object's tag
	cacheID int
	mat     ink.Material // how the manga look inks this object
	emit    *emitter     // set if the node is a lamp
	crease  float64      // outline crease angle in degrees; <0 = no outline
	hidden  bool
	vis     float64 // animated visibility (>= 0.5 shows)
	cacheW  geom.Matrix
	cacheOK bool
	cacheM  *model.Mesh
}

func NewNode(name string) *Node {
	return &Node{Name: name, Scl: geom.Vector{X: 1, Y: 1, Z: 1}, vis: 1, crease: 35, mat: ink.DefaultMaterial()}
}

// Group adds an empty child, handy as a pivot or a container.
func (n *Node) Group(name string) *Node {
	c := NewNode(name)
	c.Parent = n
	n.Children = append(n.Children, c)
	return c
}

// Add adds a child that draws mesh m (build it with package solid).
func (n *Node) Add(m *model.Mesh) *Node {
	c := n.Group("mesh")
	c.mesh = m
	c.id = int(atomic.AddInt64(&nextID, 1))
	return c
}

// Place adds an analytic shape (such as shapes.Mirror) that moves with this node.
func (n *Node) Place(s shapes.Shape) *Node {
	c := n.Group("shape")
	c.shape = s
	return c
}

// Reparent moves the node under p; its local transform is kept.
func (n *Node) Reparent(p *Node) *Node {
	if n.Parent != nil {
		kids := n.Parent.Children[:0]
		for _, k := range n.Parent.Children {
			if k != n {
				kids = append(kids, k)
			}
		}
		n.Parent.Children = kids
	}
	n.Parent = p
	if p != nil {
		p.Children = append(p.Children, n)
	}
	return n
}

func (n *Node) At(x, y, z float64) *Node     { n.Pos = geom.Vector{X: x, Y: y, Z: z}; return n }
func (n *Node) Turn(x, y, z float64) *Node   { n.Rot = geom.Vector{X: x, Y: y, Z: z}; return n }
func (n *Node) Size(x, y, z float64) *Node   { n.Scl = geom.Vector{X: x, Y: y, Z: z}; return n }
func (n *Node) Uniform(s float64) *Node      { return n.Size(s, s, s) }
func (n *Node) Named(name string) *Node      { n.Name = name; return n }
func (n *Node) NoOutline() *Node             { n.crease = -1; n.mat.Line = false; return n }
func (n *Node) Crease(degrees float64) *Node { n.crease = degrees; return n }

// Merge draws this node and all its descendants as one object: one outline
// around the whole thing, no seams between its parts. Use it for a limb or a body
// whose joints should not show as lines.
func (n *Node) Merge() *Node {
	n.merge = true
	if n.gid == 0 {
		n.gid = int(atomic.AddInt64(&nextID, 1))
	}
	return n
}

// tag is the object id and material this node is drawn with: its own, or that of
// the nearest merged ancestor.
func (n *Node) tag() (int, ink.Material) {
	for p := n; p != nil; p = p.Parent {
		if p.merge {
			return p.gid, p.mat
		}
	}
	return n.id, n.mat
}

// Ink sets how the manga look draws this object (solid black, never-shaded lamps, ...).
func (n *Node) Ink(m ink.Material) *Node { n.mat = m; return n }

// Paint sets the colour this object is drawn in by ColourLook.
func (n *Node) Paint(c ink.Color) *Node { n.mat.Color = c; return n }

// Emit makes the node a lamp: a point light at its position that fades with
// distance (half power at radius). It lights the scene in every look.
func (n *Node) Emit(power, radius float64) *Node {
	n.emit = &emitter{power, radius, sizeOf(n.mesh)}
	return n
}

type emitter struct{ power, radius, size float64 }

var nextID int64

func (n *Node) Hide() *Node { n.hidden = true; return n }

// Local is the node's transform relative to its parent.
func (n *Node) Local() geom.Matrix {
	return geom.Identity().
		Scale(n.Scl).
		Rotate(geom.Vector{X: 1}, -n.Rot.X). // geom.Rotate turns clockwise
		Rotate(geom.Vector{Y: 1}, -n.Rot.Y).
		Rotate(geom.Vector{Z: 1}, -n.Rot.Z).
		Translate(n.Pos)
}

// World is the node's transform in scene space.
func (n *Node) World() geom.Matrix {
	if n.Parent == nil {
		return n.Local()
	}
	return n.Parent.World().Mul(n.Local())
}

func (n *Node) WorldPos() geom.Vector { return n.World().MulPosition(geom.Vector{}) }

func (n *Node) shown() bool {
	for p := n; p != nil; p = p.Parent {
		if p.hidden || p.vis < 0.5 {
			return false
		}
	}
	return true
}

// walk visits n and all descendants.
func (n *Node) walk(f func(*Node)) {
	f(n)
	for _, c := range n.Children {
		c.walk(f)
	}
}

// outline edges are expensive to weld, so share them between nodes using one mesh.
var (
	outlineMu    sync.Mutex
	outlineCache = map[outlineKey]*solid.Outlined{}
)

type outlineKey struct {
	m      *model.Mesh
	crease float64
}

func outlineFor(m *model.Mesh, crease float64) *solid.Outlined {
	outlineMu.Lock()
	defer outlineMu.Unlock()
	k := outlineKey{m, crease}
	if o, ok := outlineCache[k]; ok {
		return o
	}
	o := solid.Outline(m, crease)
	outlineCache[k] = o
	return o
}

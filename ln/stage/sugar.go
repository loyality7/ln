package stage

import (
	"math"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/ink"
	"github.com/loyality7/ln/ln/model"
	"github.com/loyality7/ln/ln/solid"
)

// Shape shortcuts: each adds a child node holding the shape, ready for At / Turn / Size.

func (n *Node) Box(sx, sy, sz float64) *Node       { return n.Add(solid.Box(sx, sy, sz)) }
func (n *Node) Sphere(r float64) *Node             { return n.Add(solid.Sphere(r)) }
func (n *Node) Ellipsoid(rx, ry, rz float64) *Node { return n.Add(solid.Ellipsoid(rx, ry, rz)) }
func (n *Node) Cylinder(r, h float64) *Node        { return n.Add(solid.Cylinder(r, h)) }
func (n *Node) Cone(r, h float64) *Node            { return n.Add(solid.Cone(r, h)) }
func (n *Node) Torus(big, small float64) *Node     { return n.Add(solid.Torus(big, small)) }
func (n *Node) Capsule(r, length float64) *Node    { return n.Add(solid.Capsule(r, length)) }

// Surface adds a large shaded surface (terrain, a floor) that is outlined where
// it meets the sky but has no crease lines across it.
func (n *Node) Surface(m *model.Mesh) *Node {
	return n.Add(m).NoOutline().Ink(ink.DefaultMaterial())
}

// Orbit swings a node (usually the camera eye) around a centre point: radius and
// height from the centre, angle from `from` to `to` in radians, over dur seconds.
func (tl *Timeline) Orbit(eye *Node, center geom.Vector, radius, height, from, to, dur float64, e Ease) {
	tl.Do(dur, func(p float64) {
		if e != nil {
			p = e(p)
		}
		a := from + (to-from)*p
		eye.Pos = center.Add(geom.V(math.Cos(a)*radius, math.Sin(a)*radius, height))
	})
}

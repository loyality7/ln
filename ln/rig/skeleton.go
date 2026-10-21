package rig

import (
	"fmt"
	"math"
	"strings"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/ink"
	"github.com/loyality7/ln/ln/model"
	"github.com/loyality7/ln/ln/solid"
	"github.com/loyality7/ln/ln/stage"
)

// Axis is the local axis a joint turns about. The zero value is Y, which suits a
// character that moves in the x-z plane and is filmed from the side.
type Axis int

const (
	AxisY Axis = iota
	AxisX
	AxisZ
)

// Limb is a capsule drawn from a joint: down (-z) by default, up with Up.
type Limb struct {
	Radius, Length float64
	Up             bool
}

// Part is an extra mesh carried by a joint (a head, a fist, a foot, a scarf tail).
type Part struct {
	Mesh     *model.Mesh
	At, Turn geom.Vector
	Ink      *ink.Material // nil = the skeleton's default ink
}

// Joint is one bone of the skeleton. At is its offset from its parent.
type Joint struct {
	Name, Parent string
	At           geom.Vector
	Axis         Axis
	Sign         float64 // -1 flips the direction a positive angle turns (0 means 1)
	Limb         *Limb
	Parts        []Part
	Track        bool         // effects such as smear arcs follow this joint's tip
	Tip          *geom.Vector // the tracked point in the joint's space (default: end of its limb)
}

// Spec describes a whole skeleton as data.
type Spec struct {
	Height float64      // the height the dimensions were drawn at
	Ink    ink.Material // default ink for limbs and parts (zero value: solid black)
	Joints []Joint
}

// Skeleton is a built rig: one stage node per joint.
type Skeleton struct {
	Root           *stage.Node
	Origin, Facing float64 // where it stands along x and which way forward points (1 or -1)

	joints map[string]*stage.Node
	spec   map[string]Joint
	order  []string
}

// Build creates the skeleton's nodes under parent, scaled to height. It stands at
// x = origin and faces +x (facing 1) or -x (facing -1).
func Build(parent *stage.Node, spec Spec, height, origin, facing float64) *Skeleton {
	s := &Skeleton{Origin: origin, Facing: facing, joints: map[string]*stage.Node{}, spec: map[string]Joint{}}
	s.Root = parent.Group("rig").Uniform(height / spec.Height)
	s.Root.Pos.X = origin
	defaultInk := spec.Ink
	if defaultInk == (ink.Material{}) {
		defaultInk = ink.Material{Class: ink.Black}
	}
	for _, j := range spec.Joints {
		p := s.Root
		if j.Parent != "" {
			var ok bool
			if p, ok = s.joints[j.Parent]; !ok {
				panic(fmt.Sprintf("rig: joint %q has parent %q which is not defined before it", j.Name, j.Parent))
			}
		}
		n := p.Group(j.Name).At(j.At.X, j.At.Y, j.At.Z)
		if l := j.Limb; l != nil {
			at := 0.0
			if !l.Up {
				at = -l.Length
			}
			n.Add(solid.Capsule(l.Radius, l.Length-2*l.Radius)).At(0, 0, at).NoOutline().Ink(defaultInk)
		}
		for _, pt := range j.Parts {
			m := defaultInk
			if pt.Ink != nil {
				m = *pt.Ink
			}
			n.Add(pt.Mesh).At(pt.At.X, pt.At.Y, pt.At.Z).Turn(pt.Turn.X, pt.Turn.Y, pt.Turn.Z).NoOutline().Ink(m)
		}
		s.joints[j.Name], s.spec[j.Name] = n, j
		s.order = append(s.order, j.Name)
	}
	return s
}

// Joint returns the node of a joint.
func (s *Skeleton) Joint(name string) *stage.Node {
	n, ok := s.joints[name]
	if !ok {
		panic(fmt.Sprintf("rig: no joint named %q", name))
	}
	return n
}

// Apply poses the skeleton: every channel not in p is at rest.
func (s *Skeleton) Apply(p Pose) {
	for _, name := range s.order {
		n, j := s.joints[name], s.spec[name]
		n.Rot, n.Pos = geom.Vector{}, j.At
	}
	root := geom.Vector{}
	for ch, v := range p {
		switch ch {
		case "root.x":
			root.X = v
		case "root.z":
			root.Z = v
		case "root.yaw":
			root.Y = v
		default:
			name, axis, move := ch, "", false
			if i := strings.LastIndex(ch, "."); i >= 0 {
				name, axis, move = ch[:i], ch[i+1:], true
			}
			n, ok := s.joints[name]
			if !ok {
				panic(fmt.Sprintf("rig: pose channel %q names no joint", ch))
			}
			if move {
				switch axis {
				case "x":
					n.Pos.X += v
				case "y":
					n.Pos.Y += v
				case "z":
					n.Pos.Z += v
				}
				continue
			}
			sign := s.spec[name].Sign
			if sign == 0 {
				sign = 1
			}
			switch s.spec[name].Axis {
			case AxisX:
				n.Rot.X += sign * v
			case AxisZ:
				n.Rot.Z += sign * v
			default:
				n.Rot.Y += sign * v
			}
		}
	}
	s.Root.Pos.X, s.Root.Pos.Z = s.Origin+s.Facing*root.X, root.Z
	s.Root.Rot.Z = root.Y
	if s.Facing < 0 {
		s.Root.Rot.Z += math.Pi
	}
}

// Tips are the world positions of the tracked joints' tips (hands, feet, head),
// which smear arcs follow.
func (s *Skeleton) Tips() []geom.Vector {
	var out []geom.Vector
	for _, name := range s.order {
		j := s.spec[name]
		if !j.Track {
			continue
		}
		tip := geom.Vector{}
		switch {
		case j.Tip != nil:
			tip = *j.Tip
		case j.Limb != nil && !j.Limb.Up:
			tip = geom.V(0, 0, -j.Limb.Length)
		case j.Limb != nil:
			tip = geom.V(0, 0, j.Limb.Length)
		}
		out = append(out, s.joints[name].World().MulPosition(tip))
	}
	return out
}

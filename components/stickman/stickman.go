// Package stickman is a classic stick figure: a skeleton description and a library
// of moves written as keyframe and oscillator tables for ln/rig. There is no motion
// code here; to add a move, list its poses.
//
// Joints: pelvis, spine, head, and for each of N (near) and F (far): shoulder,
// elbow, hip, knee. Angles are positive forward (the figure faces +x).
package stickman

import (
	"math"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/ink"
	"github.com/loyality7/ln/ln/rig"
	"github.com/loyality7/ln/ln/solid"
	"github.com/loyality7/ln/ln/stage"
)

const (
	thick = 0.034 // limb radius
	headR = 0.115
)

// Look chooses how a figure is dressed so two of them can be told apart.
type Look struct {
	SolidHead bool // a filled head instead of a ring
	Scarf     bool // trailing scarf
}

// New builds a stick man of the given height under parent, standing at x = origin
// and facing +x (facing 1) or -x (facing -1).
func New(parent *stage.Node, height, origin, facing float64, look Look) *rig.Skeleton {
	return rig.Build(parent, spec(look), height, origin, facing)
}

func spec(look Look) rig.Spec {
	line := ink.Material{Class: ink.Black}
	ring := ink.Material{Class: ink.Paper, Line: true, LineW: 1.15, BlackAt: 2}
	headInk := &ring
	if look.SolidHead {
		headInk = &line
	}
	headTip := geom.V(0, 0, 0.03+headR)
	head := []rig.Part{
		{Mesh: solid.Sphere(headR), At: headTip, Ink: headInk},
		{Mesh: solid.Capsule(thick*0.8, 0.01)}, // neck
	}
	if look.Scarf { // two tails streaming behind the neck
		for _, a := range []float64{0.35, 0.78} {
			head = append(head, rig.Part{Mesh: solid.Capsule(thick*0.55, 0.28), At: geom.V(-0.05, 0, 0.04), Turn: geom.V(0, math.Pi+a, 0)})
		}
	}
	const fwd = -1 // a positive angle swings the limb toward +x

	joints := []rig.Joint{
		{Name: "pelvis", At: geom.V(0, 0, 0.95), Sign: fwd},
		{Name: "spine", Parent: "pelvis", Sign: fwd, Limb: &rig.Limb{Radius: thick, Length: 0.5, Up: true}},
		{Name: "head", Parent: "spine", At: geom.V(0, 0, 0.5), Sign: fwd, Parts: head, Track: true, Tip: &headTip},
	}
	for _, s := range []struct {
		side string
		y    float64
	}{{"N", -0.07}, {"F", 0.07}} {
		joints = append(joints,
			rig.Joint{Name: "shoulder" + s.side, Parent: "spine", At: geom.V(0, s.y, 0.46), Sign: fwd, Limb: &rig.Limb{Radius: thick * 0.92, Length: 0.29}},
			rig.Joint{Name: "elbow" + s.side, Parent: "shoulder" + s.side, At: geom.V(0, 0, -0.29), Sign: fwd, Limb: &rig.Limb{Radius: thick * 0.85, Length: 0.27},
				Parts: []rig.Part{{Mesh: solid.Sphere(0.042), At: geom.V(0, 0, -0.27)}}, Track: true},
			rig.Joint{Name: "hip" + s.side, Parent: "pelvis", At: geom.V(0, s.y*0.7, 0), Sign: fwd, Limb: &rig.Limb{Radius: thick, Length: 0.46}},
			rig.Joint{Name: "knee" + s.side, Parent: "hip" + s.side, At: geom.V(0, 0, -0.46), Limb: &rig.Limb{Radius: thick * 0.95, Length: 0.46},
				Parts: []rig.Part{{Mesh: solid.Capsule(thick*0.9, 0.12), At: geom.V(-0.03, 0, -0.455), Turn: geom.V(0, math.Pi/2, 0)}}, Track: true},
		)
	}
	return rig.Spec{Height: 1.8, Joints: joints}
}

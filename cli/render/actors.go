package render

import (
	"fmt"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/model"
	"github.com/loyality7/ln/ln/rig"
	"github.com/loyality7/ln/ln/solid"
	"github.com/loyality7/ln/ln/stage"
)

var axes = map[string]rig.Axis{"": rig.AxisDefault, "y": rig.AxisY, "x": rig.AxisX, "z": rig.AxisZ}

// partMesh makes the solid for a joint part.
func partMesh(who string, p jointPart) (*model.Mesh, error) {
	a := p.Args
	need := func(c int) error {
		if len(a) != c {
			return fmt.Errorf("actor part %q in %s needs %d number(s), got %d", p.Shape, who, c, len(a))
		}
		return nil
	}
	switch p.Shape {
	case "sphere":
		if err := need(1); err != nil {
			return nil, err
		}
		return solid.Sphere(a[0]), nil
	case "box":
		if err := need(3); err != nil {
			return nil, err
		}
		return solid.Box(a[0], a[1], a[2]), nil
	case "ellipsoid":
		if err := need(3); err != nil {
			return nil, err
		}
		return solid.Ellipsoid(a[0], a[1], a[2]), nil
	case "capsule":
		if err := need(2); err != nil {
			return nil, err
		}
		return solid.Capsule(a[0], a[1]), nil
	}
	return nil, fmt.Errorf("actor part: unknown shape %q (sphere, box, ellipsoid, capsule)", p.Shape)
}

// buildActor makes the skeleton and its player.
func buildActor(st *stage.Stage, a actor) (*rig.Actor, error) {
	if len(a.Joints) == 0 {
		return nil, fmt.Errorf("actor %q has no joints", a.Name)
	}
	if a.Height <= 0 {
		a.Height = 1.8
	}
	if a.Drawn <= 0 {
		a.Drawn = a.Height
	}
	if a.Facing == 0 {
		a.Facing = 1
	}
	ax, ok := axes[a.Axis]
	if !ok {
		return nil, fmt.Errorf("actor %q: axis is x, y or z", a.Name)
	}
	spec := rig.Spec{Axis: ax, Height: a.Drawn}
	for _, j := range a.Joints {
		jj := rig.Joint{Name: j.Name, Parent: j.Parent, At: geom.V(j.At[0], j.At[1], j.At[2]), Sign: j.Sign, Track: j.Track}
		if jj.Axis, ok = axes[j.Axis]; !ok {
			return nil, fmt.Errorf("actor %q, joint %q: axis is x, y or z", a.Name, j.Name)
		}
		if j.Limb != nil {
			jj.Limb = &rig.Limb{Radius: j.Limb.Radius, Length: j.Limb.Length, Up: j.Limb.Up}
		}
		for _, p := range j.Parts {
			m, err := partMesh(a.Name+"."+j.Name, p)
			if err != nil {
				return nil, err
			}
			jj.Parts = append(jj.Parts, rig.Part{Mesh: m, At: geom.V(p.At[0], p.At[1], p.At[2])})
		}
		spec.Joints = append(spec.Joints, jj)
	}
	var sk *rig.Skeleton
	var perr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				perr = fmt.Errorf("actor %q: %v", a.Name, r)
			}
		}()
		sk = rig.Build(st.Root, spec, a.Height, a.Origin, a.Facing)
	}()
	if perr != nil {
		return nil, perr
	}
	return rig.NewActor(st.TL, sk), nil
}

// clipOf turns a timeline step into a clip for an actor.
func clipOf(a action, who string) (rig.Clip, error) {
	dur := a.Dur
	if dur <= 0 {
		return rig.Clip{}, fmt.Errorf("actor %q: a move needs \"dur\" above 0", who)
	}
	var c rig.Clip
	switch {
	case a.Hold != "":
		c = rig.Freeze(dur)
	case a.Play != "":
		if len(a.Keys) < 2 {
			return c, fmt.Errorf("actor %q: play needs at least 2 \"keys\"", who)
		}
		keys := make([]rig.Key, len(a.Keys))
		for i, k := range a.Keys {
			ease, ok := eases[k.Ease]
			if !ok {
				return c, fmt.Errorf("actor %q: unknown ease %q", who, k.Ease)
			}
			keys[i] = rig.K(k.T, k.Pose)
			keys[i].Ease = ease
		}
		c = rig.Keys(dur, keys...)
	default:
		cycles := a.Cycles
		if cycles == 0 {
			cycles = 1
		}
		waves := map[string]rig.Wave{"": rig.Sine, "sine": rig.Sine, "half": rig.Half, "abs": rig.Abs}
		var osc []rig.Osc
		for _, s := range a.Swing {
			w, ok := waves[s.Wave]
			if !ok {
				return c, fmt.Errorf("actor %q: wave is sine, half or abs", who)
			}
			osc = append(osc, rig.Osc{Ch: s.Channel, Amp: s.Amp, Phase: s.Phase, Mult: s.Mult, Wave: w})
		}
		c = rig.Cycle(dur, cycles, a.Base, osc...)
	}
	if a.Arc != nil {
		c = c.Arc(a.Arc.Height, a.Arc.From, a.Arc.To)
	}
	if a.Spin != nil {
		c = c.Spin(a.Spin.Channel, a.Spin.Turns, a.Spin.From, a.Spin.To)
	}
	if a.Travel != nil {
		c = c.Travel(a.Travel.Dist, a.Travel.From, a.Travel.To)
	}
	for _, m := range a.Marks {
		c = c.Mark(rig.Mark{Kind: m.Kind, At: m.At, Span: m.Span, Secs: m.Secs, X: m.X, Z: m.Z, AtEnd: m.AtEnd, Size: m.Size})
	}
	return c, nil
}

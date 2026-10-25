package render

import (
	"fmt"
	"github.com/loyality7/ln/ln/anim"
	"github.com/loyality7/ln/ln/fx"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/ink"
	"github.com/loyality7/ln/ln/light"
	"github.com/loyality7/ln/ln/physics"
	"github.com/loyality7/ln/ln/rig"
	"github.com/loyality7/ln/ln/shapes"
	"github.com/loyality7/ln/ln/solid"
	"github.com/loyality7/ln/ln/stage"
	"path/filepath"
)

// build turns a scene description into a stage, or says what is wrong with it.
func build(sc scene, lookOverride, dir string) (*stage.Stage, error) {
	w, h, fps := sc.Width, sc.Height, sc.FPS
	if w == 0 {
		w = 1280
	}
	if h == 0 {
		h = 720
	}
	if fps == 0 {
		fps = 24
	}
	st := stage.New(w, h, fps)
	if g := sc.Gravity; g != nil {
		if g.Mass <= 0 {
			return nil, fmt.Errorf("gravity needs a \"mass\" above 0")
		}
		capture, escape := g.Capture, g.Escape
		if capture == 0 {
			capture = 2 * g.Mass
		}
		if escape == 0 {
			escape = 80 * g.Mass
		}
		f := &physics.Field{Law: physics.Gravity{M: g.Mass}.Light(), Capture: capture, Escape: escape}
		if g.Center != nil {
			f.Center = vec(g.Center)
		}
		st.Bend = f
	}
	if l := sc.Light; l != nil {
		sun := geom.V(-0.5, -0.6, 1)
		if l.Sun != nil {
			sun = vec(l.Sun)
		}
		rg := light.Sun(sun)
		if l.Sun == nil && len(l.Lights) > 0 {
			rg.Lights = nil // lights were listed and no sun: only those light the scene
		}
		if l.Power > 0 {
			rg.Lights = []light.Light{light.Directional{Dir: sun, Power: l.Power}}
		}
		for i, ls := range l.Lights {
			switch ls.Type {
			case "directional":
				if ls.Dir == nil {
					return nil, fmt.Errorf("light %d: a directional light needs \"dir\"", i+1)
				}
				rg.Lights = append(rg.Lights, light.Directional{Dir: vec(ls.Dir), Power: orOne(ls.Power)})
			case "point", "spot":
				if ls.Pos == nil {
					return nil, fmt.Errorf("light %d: a %s light needs \"pos\"", i+1, ls.Type)
				}
				p := light.Point{Pos: vec(ls.Pos), Power: orOne(ls.Power), Radius: orVal(ls.Radius, 6), Size: ls.Size}
				if ls.Type == "point" {
					rg.Lights = append(rg.Lights, p)
					break
				}
				if ls.Axis == nil || ls.Cone <= 0 {
					return nil, fmt.Errorf("light %d: a spot light needs \"axis\" and a \"cone\" half-angle in radians", i+1)
				}
				rg.Lights = append(rg.Lights, light.Spot{Point: p, Axis: vec(ls.Axis), Cone: ls.Cone})
			default:
				return nil, fmt.Errorf("light %d: type is point, spot or directional, not %q", i+1, ls.Type)
			}
		}
		if l.Ambient != nil {
			rg.Ambient = *l.Ambient
		}
		if l.Shadows != nil {
			rg.Shadows = *l.Shadows
		}
		rg.Exposure = l.Exposure
		st.Atmosphere = stage.Studio{Lights: rg}
	}
	if sc.Style != nil && sc.Style.Step != 0 {
		plotterStep = sc.Style.Step
	}
	look := sc.Look
	if lookOverride != "" {
		look = lookOverride
	}
	if err := setLook(st, look, sc.Style, sc.Sky); err != nil {
		return nil, err
	}
	if sc.Camera.Eye != nil {
		st.Cam.Eye.At(sc.Camera.Eye[0], sc.Camera.Eye[1], sc.Camera.Eye[2])
	}
	if sc.Camera.Target != nil {
		st.Cam.Target.At(sc.Camera.Target[0], sc.Camera.Target[1], sc.Camera.Target[2])
	}
	if sc.Camera.Fov > 0 {
		st.Cam.Fovy = sc.Camera.Fov
	}
	byName := map[string]*stage.Node{"camera": st.Cam.Eye}
	var add func(parent *stage.Node, n node) error
	add = func(parent *stage.Node, n node) error {
		var made *stage.Node
		a := n.Args
		need := func(count int) error {
			if len(a) != count {
				return fmt.Errorf("node %q: shape %s needs %d number(s) in \"args\", got %d", n.Name, n.Shape, count, len(a))
			}
			return nil
		}
		switch n.Shape {
		case "group", "":
			made = parent.Group(n.Name)
		case "box":
			if err := need(3); err != nil {
				return err
			}
			made = parent.Box(a[0], a[1], a[2])
		case "sphere":
			if err := need(1); err != nil {
				return err
			}
			made = parent.Sphere(a[0])
		case "ellipsoid":
			if err := need(3); err != nil {
				return err
			}
			made = parent.Ellipsoid(a[0], a[1], a[2])
		case "cylinder":
			if err := need(2); err != nil {
				return err
			}
			made = parent.Cylinder(a[0], a[1])
		case "cone":
			if err := need(2); err != nil {
				return err
			}
			made = parent.Cone(a[0], a[1])
		case "torus":
			if err := need(2); err != nil {
				return err
			}
			made = parent.Torus(a[0], a[1])
		case "capsule":
			if err := need(2); err != nil {
				return err
			}
			made = parent.Capsule(a[0], a[1])
		case "mesh":
			file := n.File
			if file != "" && !filepath.IsAbs(file) {
				file = filepath.Join(dir, file)
			}
			m, err := loadMesh(file)
			if err != nil {
				return fmt.Errorf("node %q: %v", n.Name, err)
			}
			made = parent.Add(m)
			if len(a) == 1 {
				made.Uniform(a[0])
			}
		case "lathe":
			if len(n.Profile) < 2 {
				return fmt.Errorf("node %q: lathe needs a \"profile\" of at least 2 [distance, height] points", n.Name)
			}
			pts := make([]solid.Pt, len(n.Profile))
			for i, q := range n.Profile {
				pts[i] = solid.Pt{X: q[0], Y: q[1]}
			}
			if n.Smooth > 0 {
				pts = solid.Smooth(pts, n.Smooth)
			}
			sides := n.Sides
			if sides == 0 {
				sides = 32
			}
			made = parent.Add(solid.Lathe(pts, sides))
		case "extrude":
			if len(n.Polygon) < 3 || len(a) != 1 {
				return fmt.Errorf("node %q: extrude needs a \"polygon\" of at least 3 [x, y] points and \"args\": [height]", n.Name)
			}
			pts := make([]solid.Pt, len(n.Polygon))
			for i, q := range n.Polygon {
				pts[i] = solid.Pt{X: q[0], Y: q[1]}
			}
			made = parent.Add(solid.Extrude(pts, a[0]))
		case "tube":
			if len(n.Path) < 2 || len(a) != 1 {
				return fmt.Errorf("node %q: tube needs a \"path\" of at least 2 [x, y, z] points and \"args\": [radius]", n.Name)
			}
			path := make([]geom.Vector, len(n.Path))
			for i, q := range n.Path {
				path[i] = geom.V(q[0], q[1], q[2])
			}
			sides := n.Sides
			if sides == 0 {
				sides = 16
			}
			made = parent.Add(solid.Tube(path, solid.Constant(a[0]), sides, n.Closed))
		case "annulus":
			if err := need(2); err != nil {
				return err
			}
			made = parent.Place(shapes.NewAnnulus(geom.Vector{}, a[0], a[1]))
		case "heightfield":
			if err := need(3); err != nil {
				return err
			}
			file := n.File
			if file != "" && !filepath.IsAbs(file) {
				file = filepath.Join(dir, file)
			}
			m, err := loadHeightfield(file, a[0], a[1], a[2])
			if err != nil {
				return fmt.Errorf("node %q: %v", n.Name, err)
			}
			made = parent.Add(m)
		case "mirror":
			if err := need(2); err != nil {
				return err
			}
			made = parent.Place(shapes.NewMirror(a[0], a[1]))
		case "csg":
			if len(n.Parts) < 2 {
				return fmt.Errorf("node %q: csg needs at least 2 \"parts\"", n.Name)
			}
			var ps []shapes.Shape
			for _, p := range n.Parts {
				sh, err := csgPiece(n.Name, p)
				if err != nil {
					return err
				}
				ps = append(ps, sh)
			}
			switch n.Op {
			case "difference", "":
				made = parent.Place(shapes.NewDifference(ps...))
			case "intersection":
				made = parent.Place(shapes.NewIntersection(ps...))
			default:
				return fmt.Errorf("node %q: csg \"op\" is difference or intersection, not %q", n.Name, n.Op)
			}
		default:
			return fmt.Errorf("node %q: unknown shape %q (box, sphere, ellipsoid, cylinder, cone, torus, capsule, group, mesh, lathe, extrude, tube, mirror, csg)", n.Name, n.Shape)
		}
		if n.Name != "" {
			if _, dup := byName[n.Name]; dup {
				return fmt.Errorf("two nodes are named %q", n.Name)
			}
			byName[n.Name] = made.Named(n.Name)
		}
		if n.At != nil {
			made.At(n.At[0], n.At[1], n.At[2])
		}
		if n.Turn != nil {
			made.Turn(n.Turn[0], n.Turn[1], n.Turn[2])
		}
		if n.Scale != nil {
			made.Size(n.Scale[0], n.Scale[1], n.Scale[2])
		}
		if n.Paint != nil {
			made.Paint(ink.RGB(n.Paint[0], n.Paint[1], n.Paint[2]))
		}
		if n.NoLine {
			made.NoOutline()
		}
		if n.Hidden {
			made.Hide()
		}
		if n.Lamp != nil {
			power, radius := n.Lamp.Power, n.Lamp.Radius
			if power == 0 {
				power = 1
			}
			if radius == 0 {
				radius = 6
			}
			made.Emit(power, radius)
		}
		for _, c := range n.Children {
			if err := add(made, c); err != nil {
				return err
			}
		}
		return nil
	}
	for _, n := range sc.Nodes {
		if err := add(st.Root, n); err != nil {
			return nil, err
		}
	}
	actors := map[string]*rig.Actor{}
	var actorList []*rig.Actor
	for _, a := range sc.Actors {
		if _, dup := actors[a.Name]; dup || a.Name == "" {
			return nil, fmt.Errorf("every actor needs its own name (%q)", a.Name)
		}
		ac, err := buildActor(st, a)
		if err != nil {
			return nil, err
		}
		actors[a.Name] = ac
		actorList = append(actorList, ac)
	}
	var run func(acts []action) error
	run = func(acts []action) error {
		for i, a := range acts {
			ease, ok := eases[a.Ease]
			if !ok {
				return fmt.Errorf("timeline step %d: unknown ease %q", i+1, a.Ease)
			}
			target := func(name string) (*stage.Node, error) {
				n, ok := byName[name]
				if !ok {
					return nil, fmt.Errorf("timeline step %d: no node named %q", i+1, name)
				}
				return n, nil
			}
			need := func(what string) error {
				if a.To == nil {
					return fmt.Errorf("timeline step %d: %s needs \"to\": [x, y, z]", i+1, what)
				}
				return nil
			}
			who := firstOf(a.Play, a.Cycle, a.Hold)
			switch {
			case who != "":
				ac, ok := actors[who]
				if !ok {
					return fmt.Errorf("timeline step %d: no actor named %q", i+1, who)
				}
				c, err := clipOf(a, who)
				if err != nil {
					return fmt.Errorf("timeline step %d: %v", i+1, err)
				}
				ac.Play(c)
			case a.Move != "":
				n, err := target(a.Move)
				if err != nil {
					return err
				}
				if err := need("move"); err != nil {
					return err
				}
				st.TL.Move(n, vec(a.To), a.Dur, ease)
			case a.Turn != "":
				n, err := target(a.Turn)
				if err != nil {
					return err
				}
				if err := need("turn"); err != nil {
					return err
				}
				st.TL.Turn(n, vec(a.To), a.Dur, ease)
			case a.Scale != "":
				n, err := target(a.Scale)
				if err != nil {
					return err
				}
				if err := need("scale"); err != nil {
					return err
				}
				st.TL.Scale(n, vec(a.To), a.Dur, ease)
			case a.Follow != "":
				n, err := target(a.Follow)
				if err != nil {
					return err
				}
				if len(a.Path) < 2 {
					return fmt.Errorf("timeline step %d: follow needs a \"path\" of at least 2 [x, y, z] points", i+1)
				}
				pts := make([]geom.Vector, len(a.Path))
				for k, q := range a.Path {
					pts[k] = geom.V(q[0], q[1], q[2])
				}
				st.TL.Follow(n, anim.Spline(pts), a.Dur, ease)
			case a.Orbit != nil:
				o := a.Orbit
				st.TL.Orbit(st.Cam.Eye, vec(&o.Center), o.Radius, o.Height, o.From, o.To, a.Dur, ease)
			case a.Wait != nil:
				st.TL.Wait(*a.Wait)
			case a.Fov != nil:
				st.TL.Fov(*a.Fov, a.Dur, ease)
			case a.Show != "":
				n, err := target(a.Show)
				if err != nil {
					return err
				}
				st.TL.Show(n)
			case a.Hide != "":
				n, err := target(a.Hide)
				if err != nil {
					return err
				}
				st.TL.Hide(n)
			case len(a.Par) > 0:
				var perr error
				st.TL.Par(func() { perr = run(a.Par) })
				if perr != nil {
					return perr
				}
			case len(a.Seq) > 0:
				var perr error
				st.TL.Seq(func() { perr = run(a.Seq) })
				if perr != nil {
					return perr
				}
			default:
				return fmt.Errorf("timeline step %d does nothing: use one of move, turn, scale, follow, orbit, wait, fov, show, hide, par, seq, play, cycle, hold", i+1)
			}
		}
		return nil
	}
	if err := run(sc.Timeline); err != nil {
		return nil, err
	}
	for _, c := range sc.Captions {
		fx.Caption(st, c.At, c.Dur, c.Text)
	}
	if sc.Follow != nil {
		if len(actorList) == 0 {
			return nil, fmt.Errorf("\"follow\" needs at least one actor to follow")
		}
		f := fx.DefaultFollow()
		if o := sc.Follow.Offset; o != nil {
			f.Offset = vec(o)
		}
		if sc.Follow.Lead != 0 {
			f.Lead = sc.Follow.Lead
		}
		if sc.Follow.Height != 0 {
			f.Height = sc.Follow.Height
		}
		if sc.Follow.Zoom != 0 {
			f.Zoom = sc.Follow.Zoom
		}
		f.Shake = sc.Follow.Shake
		fx.Install(st, f, nil, actorList...)
	}
	return st, nil
}

func orOne(v float64) float64 { return orVal(v, 1) }

func orVal(v, def float64) float64 {
	if v == 0 {
		return def
	}
	return v
}

func firstOf(s ...string) string {
	for _, v := range s {
		if v != "" {
			return v
		}
	}
	return ""
}

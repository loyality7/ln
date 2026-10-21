// Package fx draws the comic-book effects that sell a move: dust puffs, impact
// stars, speed lines, smear arcs and screen shake, plus a camera that follows the
// action. It reacts to the events actors record (rig.Event), keyed by name:
//
//	"dust"    a puff of dust at a landing or push-off
//	"impact"  a jagged star and a camera shake where a blow lands
//	"speed"   horizontal speed lines behind a runner
//	"trail"   tapering smear arcs behind the head, hands and feet
package fx

import (
	"math"
	"math/rand"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/ink"
	"github.com/loyality7/ln/ln/rig"
	"github.com/loyality7/ln/ln/stage"
)

// Follow says how the camera tracks the actors.
type Follow struct {
	Offset geom.Vector // eye position relative to the point being followed
	Lead   float64     // look this far ahead of a lone actor
	Height float64     // height of the look-at point
	Zoom   float64     // pull back by this much per unit the actors are apart beyond 3 (0 = fixed)
}

// DefaultFollow is a three-quarter side view that follows the action.
func DefaultFollow() Follow {
	return Follow{Offset: geom.V(1.7, -5.3, 0.4), Lead: 1.0, Height: 1.0, Zoom: 0.7}
}

// Install adds every effect for the actors' events to the stage, and makes the
// camera follow them. Call it once, after all moves are scheduled.
func Install(st *stage.Stage, follow Follow, actors ...*rig.Actor) {
	var events []rig.Event
	for _, a := range actors {
		events = append(events, a.Events...)
	}
	dust(st, events)
	st.Overlays = append(st.Overlays,
		ink.Overlay{Slot: ink.OverBlack, Draw: speed(st, events)},
		ink.Overlay{Slot: ink.OverBlack, Draw: trails(st, events)},
		ink.Overlay{Slot: ink.OverBlack, Draw: impacts(st, events)},
	)
	camera(st, follow, events, actors)
}

func dust(st *stage.Stage, events []rig.Event) {
	puff := ink.Material{Class: ink.Paper, Line: true, LineW: 1, BlackAt: 2}
	const n = 8
	for _, e := range events {
		if e.Kind != "dust" {
			continue
		}
		e := e
		puffs := make([]*stage.Node, n)
		for i := range puffs {
			puffs[i] = st.Root.Sphere(0.1).Ink(puff).Uniform(0.0001)
		}
		st.TL.At(e.T, func() {
			st.TL.Do(e.Dur, func(p float64) {
				out := 1 - math.Pow(1-p, 2.2)
				for i, nd := range puffs {
					a := 2 * math.Pi * float64(i) / n
					r := (0.12 + 0.75*out) * (0.7 + 0.3*float64(i%3)/2)
					s := math.Max(0.0001, (1-math.Pow(p, 1.6))*(1+0.5*float64(i%3)))
					nd.At(e.X+math.Cos(a)*r, math.Sin(a)*r*0.35, 0.06+0.16*out*(1+float64(i%2)))
					nd.Uniform(s)
				}
			})
		})
	}
}

func active(events []rig.Event, now float64, kind string) (rig.Event, bool) {
	for _, e := range events {
		if e.Kind == kind && now >= e.T && now <= e.T+e.Dur {
			return e, true
		}
	}
	return rig.Event{}, false
}

func speed(st *stage.Stage, events []rig.Event) func(*ink.Ctx) ink.Layer {
	return func(c *ink.Ctx) ink.Layer {
		now := st.TL.Now
		var l ink.Layer
		e, ok := active(events, now, "speed")
		if !ok {
			return l
		}
		fade := math.Min(1, math.Min((now-e.T)/0.25, (e.T+e.Dur-now)/0.25))
		body := c.Project(e.Who.Skel.Root.WorldPos().Add(geom.V(0, 0, 1)))
		back := -e.Who.Skel.Facing // lines stream opposite to the run
		r := rand.New(rand.NewSource(int64(now * 24)))
		for i := 0; i < 16; i++ {
			y := body.Y + (r.Float64()*2-1)*c.H*0.30
			x0 := body.X + back*(70+r.Float64()*40)
			length := (90 + r.Float64()*240) * fade
			w := 1.2 + r.Float64()*2.4
			l.Lines = append(l.Lines, ink.Line{Pts: geom.Path{{X: x0, Y: y}, {X: x0 + back*length, Y: y}}, W: []float64{w, w * 0.4}})
		}
		return l
	}
}

func trails(st *stage.Stage, events []rig.Event) func(*ink.Ctx) ink.Layer {
	return func(c *ink.Ctx) ink.Layer {
		now := st.TL.Now
		var l ink.Layer
		seen := map[*rig.Actor]bool{}
		for _, e := range events {
			if e.Kind != "trail" || now < e.T || now > e.T+e.Dur || seen[e.Who] {
				continue
			}
			seen[e.Who] = true
			const steps, span = 12, 0.30
			var paths []geom.Path
			for i := steps; i >= 0; i-- { // oldest first
				t := math.Max(0, now-span*float64(i)/steps)
				st.Probe(t, func() {
					for k, p := range e.Who.Skel.Tips() {
						if k >= len(paths) {
							paths = append(paths, nil)
						}
						paths[k] = append(paths[k], c.Project(p))
					}
				})
			}
			for _, p := range paths {
				w := make([]float64, len(p))
				for i := range w {
					w[i] = 0.5 + 4.2*float64(i)/float64(len(p)-1)
				}
				l.Lines = append(l.Lines, ink.Line{Pts: p, W: w})
			}
		}
		return l
	}
}

func impacts(st *stage.Stage, events []rig.Event) func(*ink.Ctx) ink.Layer {
	return func(c *ink.Ctx) ink.Layer {
		now := st.TL.Now
		var l ink.Layer
		for n, e := range events {
			if e.Kind != "impact" || now < e.T || now > e.T+e.Dur {
				continue
			}
			p := (now - e.T) / e.Dur
			centre := c.Project(geom.V(e.X, 0, e.Z))
			size := 1.0
			if e.Size > 0 {
				size = e.Size
			}
			radius := (30 + 95*(1-math.Pow(1-math.Min(1, p*2.5), 3))) * (1 - 0.35*p) * size
			star := func(scale float64) geom.Path {
				r := rand.New(rand.NewSource(int64(n + 5)))
				const spikes = 11
				var pts geom.Path
				for i := 0; i < spikes*2; i++ {
					a := math.Pi * float64(i) / spikes
					rr := radius * scale
					if i%2 == 0 {
						rr *= 0.95 + 0.35*r.Float64()
					} else {
						rr *= 0.42
					}
					pts = append(pts, geom.Vector{X: centre.X + rr*math.Cos(a), Y: centre.Y + rr*math.Sin(a)})
				}
				return pts
			}
			l.Fills = append(l.Fills, ink.Fill{Loops: []geom.Path{star(1.0)}}, ink.Fill{Loops: []geom.Path{star(0.62)}, White: true})
		}
		return l
	}
}

// camera follows the midpoint of the actors, pulls back as they separate, rises
// with a leap, and shakes on impacts and landings.
func camera(st *stage.Stage, f Follow, events []rig.Event, actors []*rig.Actor) {
	st.TL.At(0, func() {
		st.TL.Do(st.TL.Duration(), func(float64) {
			now := st.TL.Now
			var mx, hiz float64
			hi, lo := math.Inf(-1), math.Inf(1)
			for _, a := range actors {
				x := a.Skel.Root.WorldPos().X
				mx += x / float64(len(actors))
				hi, lo, hiz = math.Max(hi, x), math.Min(lo, x), math.Max(hiz, a.Skel.Root.Pos.Z)
			}
			lead := f.Lead
			if len(actors) > 1 {
				lead = 0
			}
			target := geom.V(mx+lead, 0, f.Height+0.5*hiz)
			zoom := 1 + f.Zoom*math.Max(0, hi-lo-3)/3
			eye := target.Add(geom.V(f.Offset.X*zoom, f.Offset.Y*zoom, f.Offset.Z))
			var sx, sy float64
			for _, e := range events {
				amp := 0.16
				switch e.Kind {
				case "impact":
				case "dust":
					amp = 0.05
				default:
					continue
				}
				if age := now - e.T; age >= 0 && age < 0.5 {
					decay := math.Exp(-7 * age)
					sx += amp * decay * math.Sin(85*now)
					sy += amp * decay * math.Cos(71*now)
				}
			}
			st.Cam.Target.At(target.X, target.Y, target.Z)
			st.Cam.Eye.At(eye.X+sx, eye.Y, eye.Z+sy)
		})
	})
}

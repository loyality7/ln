// Package comicfx draws comic-book effects for rig events: dust puffs, impact stars,
// speed lines and smear trails. They are ordinary fx.Effect values; ln/fx itself knows
// none of them, so other kinds of effects can be written the same way.
//
//	"dust"    a puff of dust at a landing or push-off
//	"impact"  a jagged star where a blow lands
//	"speed"   horizontal speed lines behind a runner
//	"trail"   tapering smear arcs behind the head, hands and feet
package comicfx

import (
	"math"
	"math/rand"

	"github.com/loyality7/ln/ln/fx"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/ink"
	"github.com/loyality7/ln/ln/rig"
	"github.com/loyality7/ln/ln/stage"
)

// All is every comic effect.
var All = []fx.Effect{Dust, Speed, Trail, Impact}

// Follow is fx.DefaultFollow plus the screen shake that goes with these effects:
// strong on impacts, light on dust.
func Follow() fx.Follow {
	f := fx.DefaultFollow()
	f.Shake = map[string]float64{"impact": 0.16, "dust": 0.05}
	return f
}

// Install makes the camera follow the actors and adds every comic effect.
func Install(st *stage.Stage, actors ...*rig.Actor) { fx.Install(st, Follow(), All, actors...) }

// Dust puffs at "dust" events.
func Dust(st *stage.Stage, events []rig.Event) {
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

// Speed draws speed lines at "speed" events.
func Speed(st *stage.Stage, events []rig.Event) {
	st.Overlays = append(st.Overlays, ink.Overlay{Slot: ink.OverBlack, Draw: speedLines(st, events)})
}

func speedLines(st *stage.Stage, events []rig.Event) func(*ink.Ctx) ink.Layer {
	return func(c *ink.Ctx) ink.Layer {
		now := st.TL.Now
		var l ink.Layer
		e, ok := fx.Active(events, now, "speed")
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

// Trail draws smear arcs at "trail" events.
func Trail(st *stage.Stage, events []rig.Event) {
	st.Overlays = append(st.Overlays, ink.Overlay{Slot: ink.OverBlack, Draw: trailLines(st, events)})
}

func trailLines(st *stage.Stage, events []rig.Event) func(*ink.Ctx) ink.Layer {
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

// Impact draws a jagged star at "impact" events.
func Impact(st *stage.Stage, events []rig.Event) {
	st.Overlays = append(st.Overlays, ink.Overlay{Slot: ink.OverBlack, Draw: impactStars(st, events)})
}

func impactStars(st *stage.Stage, events []rig.Event) func(*ink.Ctx) ink.Layer {
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

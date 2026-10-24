// Package fx adds camera work and on-screen effects to a stage that has actors in
// it: a camera that follows the action and screen shake, and a registry so any kind
// of effect can react to the events actors record (rig.Event). The effects themselves
// are not here: an Effect is any function that reads the events and adds to the stage.
// components/comicfx has comic-book ones (dust, impact stars, speed lines, smear trails).
package fx

import (
	"math"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/rig"
	"github.com/loyality7/ln/ln/stage"
)

// Effect reacts to events by adding nodes, timeline effects or overlays to the stage.
type Effect func(st *stage.Stage, events []rig.Event)

// Follow says how the camera tracks the actors.
type Follow struct {
	Offset geom.Vector        // eye position relative to the point being followed
	Lead   float64            // look this far ahead of a lone actor
	Height float64            // height of the look-at point
	Zoom   float64            // pull back by this much per unit the actors are apart beyond 3 (0 = fixed)
	Shake  map[string]float64 // screen-shake amplitude for each event kind (kinds not listed do not shake)
}

// DefaultFollow is a three-quarter side view that follows the action, with no shake.
func DefaultFollow() Follow {
	return Follow{Offset: geom.V(1.7, -5.3, 0.4), Lead: 1.0, Height: 1.0, Zoom: 0.7}
}

// Install runs each effect on the actors' events and makes the camera follow them.
// Call it once, after all moves are scheduled.
func Install(st *stage.Stage, follow Follow, effects []Effect, actors ...*rig.Actor) {
	var events []rig.Event
	for _, a := range actors {
		events = append(events, a.Events...)
	}
	for _, e := range effects {
		e(st, events)
	}
	camera(st, follow, events, actors)
}

// Active returns an event of the given kind that is in progress at time now.
func Active(events []rig.Event, now float64, kind string) (rig.Event, bool) {
	for _, e := range events {
		if e.Kind == kind && now >= e.T && now <= e.T+e.Dur {
			return e, true
		}
	}
	return rig.Event{}, false
}

// camera follows the midpoint of the actors, pulls back as they separate, rises
// with a leap, and shakes on the event kinds listed in f.Shake.
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
				amp, ok := f.Shake[e.Kind]
				if !ok {
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

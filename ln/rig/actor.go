package rig

import "github.com/loyality7/ln/ln/stage"

// Event is something an effect should react to: a landing, a hit, a run.
type Event struct {
	Kind string
	T    float64 // seconds into the film
	Dur  float64
	X, Z float64 // world position
	Size float64
	Who  *Actor
}

// Actor plays clips on the timeline one after another. Each move blends in from
// where the last one ended, and the actor remembers where it stands.
type Actor struct {
	Skel   *Skeleton
	TL     *stage.Timeline
	Blend  float64 // seconds to ease from the previous move (default 0.18)
	Events []Event

	x    float64
	last Pose
}

func NewActor(tl *stage.Timeline, skel *Skeleton) *Actor {
	return &Actor{Skel: skel, TL: tl, Blend: 0.18, last: Pose{}}
}

// X is how far forward the actor will have travelled after the moves so far.
func (a *Actor) X() float64 { return a.x }

// Play schedules a clip at the timeline's cursor and returns when it starts.
func (a *Actor) Play(c Clip) float64 {
	if c.freeze {
		held := a.last
		c.base = func(float64) Pose { return held }
	}
	start, x0, prev := a.TL.Cursor(), a.x, a.last
	blend := a.Blend
	if blend <= 0 {
		blend = 0.18
	}
	a.TL.Do(c.Dur, func(p float64) {
		a.Skel.Apply(prev.Mix(c.pose(p, x0), Smooth(p*c.Dur/blend)))
	})
	a.last, a.x = c.pose(1, x0), x0+c.Dist
	for _, m := range c.Marks {
		span := m.Span * c.Dur
		if m.Span == 0 {
			span = m.Secs
		}
		x := x0 + m.X
		if m.AtEnd {
			x = a.x + m.X
		}
		a.Events = append(a.Events, Event{Kind: m.Kind, T: start + m.At*c.Dur, Dur: span,
			X: a.Skel.Origin + a.Skel.Facing*x, Z: m.Z, Size: m.Size, Who: a})
	}
	return start
}

// Hold keeps the pose the last move ended in, for dur seconds.
func (a *Actor) Hold(dur float64) float64 { return a.Play(Freeze(dur)) }

// A Lane is a run of moves for one actor.
type Lane struct {
	Actor *Actor
	Clips []Clip
}

// Moves makes a lane: the actor plays these clips in order.
func Moves(a *Actor, clips ...Clip) Lane { return Lane{a, clips} }

// Beat plays several lanes at the same time; the next beat starts when the
// slowest lane finishes. Every actor sharing a timeline is choreographed with Beats.
func Beat(tl *stage.Timeline, lanes ...Lane) {
	tl.Par(func() {
		for _, l := range lanes {
			l := l
			tl.Seq(func() {
				for _, c := range l.Clips {
					l.Actor.Play(c)
				}
			})
		}
	})
}

// Gap is the distance between two actors facing each other.
func Gap(a, b *Actor) float64 { return (b.Skel.Origin - a.Skel.Origin) - a.x - b.x }

// Close brings two actors together until they are gap apart, each travelling
// half the distance with the clip dash(dist) makes.
func Close(tl *stage.Timeline, a, b *Actor, gap float64, dash func(dist float64) Clip) {
	each := (Gap(a, b) - gap) / 2
	Beat(tl, Moves(a, dash(each)), Moves(b, dash(each)))
}

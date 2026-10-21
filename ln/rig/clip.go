package rig

import (
	"math"

	"github.com/loyality7/ln/ln/anim"
)

// Key is a pose at a moment of a clip (T runs 0..1). Ease shapes the move into
// this key from the previous one (default: smooth).
type Key struct {
	T    float64
	Pose Pose
	Ease anim.Ease
}

// K is a key whose pose is written in degrees.
func K(t float64, degrees Pose) Key { return Key{T: t, Pose: Deg(degrees)} }

// Wave is the shape an oscillator follows.
type Wave int

const (
	Sine Wave = iota // swings both ways
	Half             // only the positive half (a knee that bends but never hyperextends)
	Abs              // always positive, twice per cycle (a bounce)
)

// Osc makes a channel swing: value = Amp * wave(2*pi*(Mult*cycles*progress + Phase)).
// Amp is in degrees for rotation channels. Phase is in cycles (0.25 turns sine into cosine).
type Osc struct {
	Ch         string
	Amp, Phase float64
	Mult       float64 // harmonic: 2 swings twice as fast (0 means 1)
	Wave       Wave
}

// Mark asks for an effect at a moment of a clip.
type Mark struct {
	Kind  string  // "dust", "impact", "trail", "speed"
	At    float64 // progress when it starts, 0..1
	Span  float64 // how long it lasts as a fraction of the clip...
	Secs  float64 // ...or in seconds (used if Span is 0)
	X, Z  float64 // where: forward distance from where the clip starts, and height
	AtEnd bool    // X is measured from where the clip ends instead
	Size  float64 // scale for impact stars (0 = normal)
}

// Clip is one move: a pose over time, plus where it travels and what it marks.
// Build one with Keys or Cycle, then shape it with Arc, Spin, Travel and Mark.
type Clip struct {
	Dur, Dist float64 // Dist is how far the move travels forward
	from, to  float64 // the stretch of the clip over which Travel happens (0,0 = all)
	base      func(p float64) Pose
	mods      []func(p float64, po Pose)
	Marks     []Mark
	freeze    bool // keep whatever pose the actor is in when it starts
}

// Freeze keeps the actor in the pose its last move ended in, for dur seconds:
// a fighter who stays down.
func Freeze(dur float64) Clip { return Clip{Dur: dur, freeze: true} }

// Keys is a move defined by poses at moments (progress 0..1, in order).
func Keys(dur float64, keys ...Key) Clip {
	return Clip{Dur: dur, base: func(p float64) Pose {
		if p <= keys[0].T {
			return keys[0].Pose
		}
		for i := 1; i < len(keys); i++ {
			if p <= keys[i].T {
				a, b := keys[i-1], keys[i]
				ease := b.Ease
				if ease == nil {
					ease = Smooth
				}
				return a.Pose.Mix(b.Pose, ease((p-a.T)/(b.T-a.T)))
			}
		}
		return keys[len(keys)-1].Pose
	}}
}

// Cycle is a repeating move: a base pose (degrees) with oscillators swinging
// channels, over `cycles` repetitions during the clip. A walk is a Cycle.
func Cycle(dur, cycles float64, base Pose, osc ...Osc) Clip {
	b := Deg(base)
	return Clip{Dur: dur, base: func(p float64) Pose {
		po := make(Pose, len(b)+len(osc))
		for k, v := range b {
			po[k] = v
		}
		for _, o := range osc {
			mult := o.Mult
			if mult == 0 {
				mult = 1
			}
			s := math.Sin(2 * math.Pi * (mult*cycles*p + o.Phase))
			switch o.Wave {
			case Half:
				s = math.Max(0, s)
			case Abs:
				s = math.Abs(s)
			}
			amp := o.Amp
			if isAngle(o.Ch) {
				amp *= math.Pi / 180
			}
			po[o.Ch] += amp * s
		}
		return po
	}}
}

func (c Clip) with(mod func(p float64, po Pose)) Clip {
	c.mods = append(append([]func(float64, Pose){}, c.mods...), mod)
	return c
}

func window(p, from, to float64) (float64, bool) {
	if p < from || p > to {
		return 0, false
	}
	if to == from {
		return 1, true
	}
	return (p - from) / (to - from), true
}

// Arc lifts the whole figure along a parabola of the given height between two
// moments of the clip: a jump.
func (c Clip) Arc(height, from, to float64) Clip {
	return c.with(func(p float64, po Pose) {
		if q, ok := window(p, from, to); ok {
			po["root.z"] += height * 4 * q * (1 - q)
		}
	})
}

// Spin turns a channel through `turns` full rotations between two moments of the
// clip: a flip is a Spin of the pelvis, a pirouette a Spin of "root.yaw".
func (c Clip) Spin(ch string, turns, from, to float64) Clip {
	return c.with(func(p float64, po Pose) {
		switch {
		case p < from:
		case p > to: // a whole number of turns looks the same as none
			po[ch] += 2 * math.Pi * (turns - math.Trunc(turns))
		default:
			q, _ := window(p, from, to)
			po[ch] += 2 * math.Pi * turns * Smooth(q)
		}
	})
}

// Travel moves the figure forward by dist (negative is back) between two
// moments of the clip. With no Travel it stays where it is.
func (c Clip) Travel(dist, from, to float64) Clip {
	c.Dist, c.from, c.to = dist, from, to
	return c
}

// Mark adds an effect event.
func (c Clip) Mark(m Mark) Clip {
	c.Marks = append(append([]Mark{}, c.Marks...), m)
	return c
}

// pose is the clip's pose at progress p for a figure that started it at x0.
func (c Clip) pose(p, x0 float64) Pose {
	src := c.base(p)
	po := make(Pose, len(src)+1)
	for k, v := range src {
		po[k] = v
	}
	for _, m := range c.mods {
		m(p, po)
	}
	from, to := c.from, c.to
	if from == 0 && to == 0 {
		to = 1
	}
	q := 0.0
	switch {
	case p >= to:
		q = 1
	case p > from:
		q = (p - from) / (to - from)
	}
	po["root.x"] = x0 + c.Dist*q
	return po
}

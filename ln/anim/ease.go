package anim

import (
	"math"

	"github.com/loyality7/ln/ln/geom"
)

// Ease maps progress in [0,1] to eased progress (may overshoot for back/elastic).
type Ease func(t float64) float64

func Linear(t float64) float64   { return t }
func InQuad(t float64) float64   { return t * t }
func OutQuad(t float64) float64  { return t * (2 - t) }
func InCubic(t float64) float64  { return t * t * t }
func OutCubic(t float64) float64 { t--; return t*t*t + 1 }
func InOutSine(t float64) float64 {
	return -(math.Cos(math.Pi*t) - 1) / 2
}
func InOutQuad(t float64) float64 {
	if t < 0.5 {
		return 2 * t * t
	}
	return -1 + (4-2*t)*t
}
func InOutCubic(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	t = 2*t - 2
	return 0.5*t*t*t + 1
}
func OutBack(t float64) float64 {
	const c1, c3 = 1.70158, 2.70158
	t--
	return 1 + c3*t*t*t + c1*t*t
}
func OutElastic(t float64) float64 {
	if t <= 0 || t >= 1 {
		return t
	}
	return math.Pow(2, -10*t)*math.Sin((t*10-0.75)*2*math.Pi/3) + 1
}
func OutBounce(t float64) float64 {
	const n, d = 7.5625, 2.75
	switch {
	case t < 1/d:
		return n * t * t
	case t < 2/d:
		t -= 1.5 / d
		return n*t*t + 0.75
	case t < 2.5/d:
		t -= 2.25 / d
		return n*t*t + 0.9375
	}
	t -= 2.625 / d
	return n*t*t + 0.984375
}

func Clamp(t float64) float64 { return math.Max(0, math.Min(1, t)) }

func Lerp(a, b, t float64) float64 { return a + (b-a)*t }

func LerpVec(a, b geom.Vector, t float64) geom.Vector { return a.Add(b.Sub(a).MulScalar(t)) }

// Segment is progress of t through the window [start,end], clamped to [0,1]
// and eased. Use it to schedule parts of an animation on one timeline:
// grow := Segment(t, 0.1, 0.4, OutBack).
func Segment(t, start, end float64, e Ease) float64 {
	if end <= start { // instant step
		if t >= end {
			t, start, end = 1, 0, 1
		} else {
			t, start, end = 0, 0, 1
		}
	}
	p := Clamp((t - start) / (end - start))
	if e == nil {
		return p
	}
	return e(p)
}

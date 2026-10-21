package anim

import (
	"sort"

	"github.com/loyality7/ln/ln/geom"
)

// Key is a value at time T. Ease shapes the move from the previous key to this one.
type Key struct {
	T    float64
	V    float64
	Ease Ease
}

// Track is a keyframed scalar; keys must be added in any order, At sorts them.
type Track []Key

// At holds the first value before the first key and the last value after the last.
func (tr Track) At(t float64) float64 {
	if len(tr) == 0 {
		return 0
	}
	keys := append(Track(nil), tr...)
	sort.Slice(keys, func(i, j int) bool { return keys[i].T < keys[j].T })
	if t <= keys[0].T {
		return keys[0].V
	}
	for i := 1; i < len(keys); i++ {
		if t <= keys[i].T {
			a, b := keys[i-1], keys[i]
			return Lerp(a.V, b.V, Segment(t, a.T, b.T, b.Ease))
		}
	}
	return keys[len(keys)-1].V
}

// VecTrack keyframes a point or direction, one Track per axis.
type VecTrack struct{ X, Y, Z Track }

func (v VecTrack) At(t float64) geom.Vector {
	return geom.Vector{X: v.X.At(t), Y: v.Y.At(t), Z: v.Z.At(t)}
}

// Spline is a Catmull-Rom curve through pts, parameterized by u in [0,1].
// Use it for camera flight paths.
func Spline(pts []geom.Vector) func(u float64) geom.Vector {
	n := len(pts)
	return func(u float64) geom.Vector {
		if n == 1 {
			return pts[0]
		}
		f := Clamp(u) * float64(n-1)
		i := int(f)
		if i >= n-1 {
			return pts[n-1]
		}
		at := func(k int) geom.Vector { return pts[max(0, min(n-1, k))] }
		p0, p1, p2, p3 := at(i-1), at(i), at(i+1), at(i+2)
		t := f - float64(i)
		t2, t3 := t*t, t*t*t
		term := func(a, b, c, d float64) float64 {
			return 0.5 * (2*b + (-a+c)*t + (2*a-5*b+4*c-d)*t2 + (-a+3*b-3*c+d)*t3)
		}
		return geom.Vector{
			X: term(p0.X, p1.X, p2.X, p3.X),
			Y: term(p0.Y, p1.Y, p2.Y, p3.Y),
			Z: term(p0.Z, p1.Z, p2.Z, p3.Z),
		}
	}
}

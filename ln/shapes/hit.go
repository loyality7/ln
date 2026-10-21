package shapes

import (
	"github.com/loyality7/ln/ln/geom"
)

// Hit is the first surface a ray meets. Normal is unit length and, for closed
// shapes, points outward; a zero Normal means "unknown" and renderers fall back
// to estimating it from neighbouring pixels.
type Hit struct {
	Shape  Shape
	T      float64
	Normal geom.Vector
}

var NoHit = Hit{T: geom.INF}

func (hit Hit) Ok() bool {
	return hit.T < geom.INF
}

func (a Hit) Min(b Hit) Hit {
	if a.T <= b.T {
		return a
	}
	return b
}

func (a Hit) Max(b Hit) Hit {
	if a.T > b.T {
		return a
	}
	return b
}

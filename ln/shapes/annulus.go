package shapes

import (
	"math"

	"github.com/loyality7/ln/ln/geom"
)

// Annulus is a flat ring: the part of a plane between two radii around a centre.
// With Inner 0 it is a disc. Normal gives the plane's orientation (default +Z).
type Annulus struct {
	Center       geom.Vector
	Normal       geom.Vector
	Inner, Outer float64
}

// NewAnnulus is a ring in the XY plane around center.
func NewAnnulus(center geom.Vector, inner, outer float64) *Annulus {
	return &Annulus{Center: center, Normal: geom.Vector{Z: 1}, Inner: inner, Outer: outer}
}

func (a *Annulus) normal() geom.Vector {
	if a.Normal == (geom.Vector{}) {
		return geom.Vector{Z: 1}
	}
	return a.Normal.Normalize()
}

// frame returns two unit vectors spanning the ring's plane.
func (a *Annulus) frame() (u, v geom.Vector) {
	n := a.normal()
	u = n.Cross(n.MinAxis()).Normalize()
	return u, n.Cross(u)
}

func (a *Annulus) Compile() {}

func (a *Annulus) BoundingBox() geom.Box {
	n := a.normal()
	// the ring's extent along each axis is Outer * sqrt(1 - n_axis^2)
	e := geom.Vector{X: a.Outer * math.Sqrt(1-n.X*n.X), Y: a.Outer * math.Sqrt(1-n.Y*n.Y), Z: a.Outer * math.Sqrt(1-n.Z*n.Z)}
	pad := geom.Vector{X: 1e-3, Y: 1e-3, Z: 1e-3}
	return geom.Box{Min: a.Center.Sub(e).Sub(pad), Max: a.Center.Add(e).Add(pad)}
}

func (a *Annulus) Contains(geom.Vector, float64) bool { return false }

func (a *Annulus) Intersect(r geom.Ray) Hit {
	n := a.normal()
	d := r.Direction.Dot(n)
	if math.Abs(d) < 1e-12 {
		return NoHit
	}
	t := a.Center.Sub(r.Origin).Dot(n) / d
	if t < 1e-3 {
		return NoHit
	}
	rr := r.Position(t).Sub(a.Center).Length()
	if rr < a.Inner || rr > a.Outer {
		return NoHit
	}
	if d > 0 {
		n = n.MulScalar(-1) // face the ray
	}
	return Hit{a, t, n}
}

// Paths draws the inner and outer edges.
func (a *Annulus) Paths() geom.Paths {
	u, v := a.frame()
	var out geom.Paths
	for _, rad := range []float64{a.Inner, a.Outer} {
		if rad <= 0 {
			continue
		}
		var p geom.Path
		for i := 0; i <= 120; i++ {
			t := 2 * math.Pi * float64(i) / 120
			p = append(p, a.Center.Add(u.MulScalar(rad*math.Cos(t))).Add(v.MulScalar(rad*math.Sin(t))))
		}
		out = append(out, p)
	}
	return out
}

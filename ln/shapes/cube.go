package shapes

import (
	"math"

	"github.com/loyality7/ln/ln/geom"
)

type Cube struct {
	Min geom.Vector
	Max geom.Vector
	Box geom.Box
}

func NewCube(min, max geom.Vector) *Cube {
	box := geom.Box{min, max}
	return &Cube{min, max, box}
}

func (c *Cube) Compile() {
}

func (c *Cube) BoundingBox() geom.Box {
	return c.Box
}

func (c *Cube) Contains(v geom.Vector, f float64) bool {
	if v.X < c.Min.X-f || v.X > c.Max.X+f {
		return false
	}
	if v.Y < c.Min.Y-f || v.Y > c.Max.Y+f {
		return false
	}
	if v.Z < c.Min.Z-f || v.Z > c.Max.Z+f {
		return false
	}
	return true
}

func (c *Cube) Intersect(r geom.Ray) Hit {
	n := c.Min.Sub(r.Origin).Div(r.Direction)
	f := c.Max.Sub(r.Origin).Div(r.Direction)
	n, f = n.Min(f), n.Max(f)
	t0 := math.Max(math.Max(n.X, n.Y), n.Z)
	t1 := math.Min(math.Min(f.X, f.Y), f.Z)
	if t0 < 1e-3 && t1 > 1e-3 {
		return Hit{c, t1, c.normalAt(r.Position(t1))}
	}
	if t0 >= 1e-3 && t0 < t1 {
		return Hit{c, t0, c.normalAt(r.Position(t0))}
	}
	return NoHit
}

// normalAt is the outward normal of the face nearest to p.
func (c *Cube) normalAt(p geom.Vector) geom.Vector {
	best, n := math.Inf(1), geom.Vector{}
	try := func(d float64, v geom.Vector) {
		if d = math.Abs(d); d < best {
			best, n = d, v
		}
	}
	try(p.X-c.Min.X, geom.Vector{X: -1})
	try(p.X-c.Max.X, geom.Vector{X: 1})
	try(p.Y-c.Min.Y, geom.Vector{Y: -1})
	try(p.Y-c.Max.Y, geom.Vector{Y: 1})
	try(p.Z-c.Min.Z, geom.Vector{Z: -1})
	try(p.Z-c.Max.Z, geom.Vector{Z: 1})
	return n
}

func (c *Cube) Paths() geom.Paths {
	x1, y1, z1 := c.Min.X, c.Min.Y, c.Min.Z
	x2, y2, z2 := c.Max.X, c.Max.Y, c.Max.Z
	paths := geom.Paths{
		{{x1, y1, z1}, {x1, y1, z2}},
		{{x1, y1, z1}, {x1, y2, z1}},
		{{x1, y1, z1}, {x2, y1, z1}},
		{{x1, y1, z2}, {x1, y2, z2}},
		{{x1, y1, z2}, {x2, y1, z2}},
		{{x1, y2, z1}, {x1, y2, z2}},
		{{x1, y2, z1}, {x2, y2, z1}},
		{{x1, y2, z2}, {x2, y2, z2}},
		{{x2, y1, z1}, {x2, y1, z2}},
		{{x2, y1, z1}, {x2, y2, z1}},
		{{x2, y1, z2}, {x2, y2, z2}},
		{{x2, y2, z1}, {x2, y2, z2}},
	}
	return paths
}

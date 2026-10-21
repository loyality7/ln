package shapes

import (
	"math"

	"github.com/loyality7/ln/ln/geom"
)

type Cone struct {
	Radius float64
	Height float64
}

func NewCone(radius, height float64) *Cone {
	return &Cone{radius, height}
}

func (c *Cone) Compile() {
}

func (c *Cone) BoundingBox() geom.Box {
	r := c.Radius
	return geom.Box{geom.Vector{-r, -r, 0}, geom.Vector{r, r, c.Height}}
}

func (c *Cone) Contains(v geom.Vector, f float64) bool {
	return false
}

func (shape *Cone) Intersect(ray geom.Ray) Hit {
	o := ray.Origin
	d := ray.Direction
	r := shape.Radius
	h := shape.Height

	k := r / h
	k = k * k

	a := d.X*d.X + d.Y*d.Y - k*d.Z*d.Z
	b := 2 * (d.X*o.X + d.Y*o.Y - k*d.Z*(o.Z-h))
	c := o.X*o.X + o.Y*o.Y - k*(o.Z-h)*(o.Z-h)
	q := b*b - 4*a*c
	if q <= 0 {
		return NoHit
	}
	s := math.Sqrt(q)
	t0 := (-b + s) / (2 * a)
	t1 := (-b - s) / (2 * a)
	if t0 > t1 {
		t0, t1 = t1, t0
	}
	if t0 > 1e-6 {
		p := ray.Position(t0)
		if p.Z > 0 && p.Z < h {
			return Hit{shape, t0, shape.normalAt(p)}
		}
	}
	if t1 > 1e-6 {
		p := ray.Position(t1)
		if p.Z > 0 && p.Z < h {
			return Hit{shape, t1, shape.normalAt(p)}
		}
	}
	return NoHit

}

// normalAt is the outward normal of the slanted side at p.
func (c *Cone) normalAt(p geom.Vector) geom.Vector {
	rho := math.Hypot(p.X, p.Y)
	if rho < 1e-9 {
		return geom.Vector{Z: 1}
	}
	return geom.Vector{X: p.X / rho, Y: p.Y / rho, Z: c.Radius / c.Height}.Normalize()
}

func (c *Cone) Paths() geom.Paths {
	var result geom.Paths
	for a := 0; a < 360; a += 30 {
		x := c.Radius * math.Cos(geom.Radians(float64(a)))
		y := c.Radius * math.Sin(geom.Radians(float64(a)))
		result = append(result, geom.Path{{x, y, 0}, {0, 0, c.Height}})
	}
	return result
}

type OutlineCone struct {
	Cone
	Eye geom.Vector
	Up  geom.Vector
}

func NewOutlineCone(eye, up geom.Vector, radius, height float64) *OutlineCone {
	cone := NewCone(radius, height)
	return &OutlineCone{*cone, eye, up}
}

func (c *OutlineCone) Paths() geom.Paths {
	center := geom.Vector{0, 0, 0}
	hyp := center.Sub(c.Eye).Length()
	opp := c.Radius
	theta := math.Asin(opp / hyp)
	adj := opp / math.Tan(theta)
	d := math.Cos(theta) * adj
	// r := math.Sin(theta) * adj
	w := center.Sub(c.Eye).Normalize()
	u := w.Cross(c.Up).Normalize()
	c0 := c.Eye.Add(w.MulScalar(d))
	a0 := c0.Add(u.MulScalar(c.Radius * 1.01))
	b0 := c0.Add(u.MulScalar(-c.Radius * 1.01))

	var p0 geom.Path
	for a := 0; a < 360; a++ {
		x := c.Radius * math.Cos(geom.Radians(float64(a)))
		y := c.Radius * math.Sin(geom.Radians(float64(a)))
		p0 = append(p0, geom.Vector{x, y, 0})
	}
	return geom.Paths{
		p0,
		{{a0.X, a0.Y, 0}, {0, 0, c.Height}},
		{{b0.X, b0.Y, 0}, {0, 0, c.Height}},
	}
}

func NewTransformedOutlineCone(eye, up, v0, v1 geom.Vector, radius float64) Shape {
	d := v1.Sub(v0)
	z := d.Length()
	a := math.Acos(d.Normalize().Dot(up))
	m := geom.Translate(v0)
	if a != 0 {
		u := d.Cross(up).Normalize()
		m = geom.Rotate(u, a).Translate(v0)
	}
	c := NewOutlineCone(m.Inverse().MulPosition(eye), up, radius, z)
	return NewTransformedShape(c, m)
}

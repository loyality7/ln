package shapes

import (
	"math"

	"github.com/loyality7/ln/ln/geom"
)

type Cylinder struct {
	Radius float64
	Z0, Z1 float64
}

func NewCylinder(radius, z0, z1 float64) *Cylinder {
	return &Cylinder{radius, z0, z1}
}

func (c *Cylinder) Compile() {
}

func (c *Cylinder) BoundingBox() geom.Box {
	r := c.Radius
	return geom.Box{geom.Vector{-r, -r, c.Z0}, geom.Vector{r, r, c.Z1}}
}

func (c *Cylinder) Contains(v geom.Vector, f float64) bool {
	xy := geom.Vector{v.X, v.Y, 0}
	if xy.Length() > c.Radius+f {
		return false
	}
	return v.Z >= c.Z0-f && v.Z <= c.Z1+f
}

func (shape *Cylinder) Intersect(ray geom.Ray) Hit {
	r := shape.Radius
	o := ray.Origin
	d := ray.Direction
	a := d.X*d.X + d.Y*d.Y
	b := 2*o.X*d.X + 2*o.Y*d.Y
	c := o.X*o.X + o.Y*o.Y - r*r
	q := b*b - 4*a*c
	if q < 0 {
		return NoHit
	}
	s := math.Sqrt(q)
	t0 := (-b + s) / (2 * a)
	t1 := (-b - s) / (2 * a)
	if t0 > t1 {
		t0, t1 = t1, t0
	}
	z0 := o.Z + t0*d.Z
	z1 := o.Z + t1*d.Z
	if t0 > 1e-6 && shape.Z0 < z0 && z0 < shape.Z1 {
		return Hit{shape, t0, sideNormal(ray.Position(t0))}
	}
	if t1 > 1e-6 && shape.Z0 < z1 && z1 < shape.Z1 {
		return Hit{shape, t1, sideNormal(ray.Position(t1))}
	}
	return NoHit

}

func sideNormal(p geom.Vector) geom.Vector { return geom.Vector{X: p.X, Y: p.Y}.Normalize() }

func (c *Cylinder) Paths() geom.Paths {
	var result geom.Paths
	for a := 0; a < 360; a += 10 {
		x := c.Radius * math.Cos(geom.Radians(float64(a)))
		y := c.Radius * math.Sin(geom.Radians(float64(a)))
		result = append(result, geom.Path{{x, y, c.Z0}, {x, y, c.Z1}})
	}
	return result
}

type OutlineCylinder struct {
	Cylinder
	Eye geom.Vector
	Up  geom.Vector
}

func NewOutlineCylinder(eye, up geom.Vector, radius, z0, z1 float64) *OutlineCylinder {
	cylinder := NewCylinder(radius, z0, z1)
	return &OutlineCylinder{*cylinder, eye, up}
}

func (c *OutlineCylinder) Paths() geom.Paths {
	center := geom.Vector{0, 0, c.Z0}
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

	center = geom.Vector{0, 0, c.Z1}
	hyp = center.Sub(c.Eye).Length()
	opp = c.Radius
	theta = math.Asin(opp / hyp)
	adj = opp / math.Tan(theta)
	d = math.Cos(theta) * adj
	// r = math.Sin(theta) * adj
	w = center.Sub(c.Eye).Normalize()
	u = w.Cross(c.Up).Normalize()
	c1 := c.Eye.Add(w.MulScalar(d))
	a1 := c1.Add(u.MulScalar(c.Radius * 1.01))
	b1 := c1.Add(u.MulScalar(-c.Radius * 1.01))

	var p0, p1 geom.Path
	for a := 0; a < 360; a++ {
		x := c.Radius * math.Cos(geom.Radians(float64(a)))
		y := c.Radius * math.Sin(geom.Radians(float64(a)))
		p0 = append(p0, geom.Vector{x, y, c.Z0})
		p1 = append(p1, geom.Vector{x, y, c.Z1})
	}
	return geom.Paths{
		p0,
		p1,
		{{a0.X, a0.Y, c.Z0}, {a1.X, a1.Y, c.Z1}},
		{{b0.X, b0.Y, c.Z0}, {b1.X, b1.Y, c.Z1}},
	}
}

func NewTransformedOutlineCylinder(eye, up, v0, v1 geom.Vector, radius float64) Shape {
	d := v1.Sub(v0)
	z := d.Length()
	a := math.Acos(d.Normalize().Dot(up))
	m := geom.Translate(v0)
	if a != 0 {
		u := d.Cross(up).Normalize()
		m = geom.Rotate(u, a).Translate(v0)
	}
	c := NewOutlineCylinder(m.Inverse().MulPosition(eye), up, radius, 0, z)
	return NewTransformedShape(c, m)
}

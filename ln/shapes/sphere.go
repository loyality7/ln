package shapes

import (
	"github.com/loyality7/ln/ln/geom"
	"math"
)

type Sphere struct {
	Center geom.Vector
	Radius float64
	Box    geom.Box
}

func NewSphere(center geom.Vector, radius float64) *Sphere {
	min := geom.Vector{center.X - radius, center.Y - radius, center.Z - radius}
	max := geom.Vector{center.X + radius, center.Y + radius, center.Z + radius}
	box := geom.Box{min, max}
	return &Sphere{center, radius, box}
}

func (s *Sphere) Compile() {
}

func (s *Sphere) BoundingBox() geom.Box {
	return s.Box
}

func (s *Sphere) Contains(v geom.Vector, f float64) bool {
	return v.Sub(s.Center).Length() <= s.Radius+f
}

func (s *Sphere) Intersect(r geom.Ray) Hit {
	radius := s.Radius
	to := r.Origin.Sub(s.Center)
	b := to.Dot(r.Direction)
	c := to.Dot(to) - radius*radius
	d := b*b - c
	if d > 0 {
		d = math.Sqrt(d)
		t1 := -b - d
		if t1 > 1e-2 {
			return Hit{s, t1, r.Position(t1).Sub(s.Center).Normalize()}
		}
		t2 := -b + d
		if t2 > 1e-2 {
			return Hit{s, t2, r.Position(t2).Sub(s.Center).Normalize()}
		}
	}
	return NoHit
}

func (s *Sphere) Paths() geom.Paths {
	var paths geom.Paths
	n := 10
	o := 10
	for lat := -90 + o; lat <= 90-o; lat += n {
		var path geom.Path
		for lng := 0; lng <= 360; lng++ {
			v := LatLngToXYZ(float64(lat), float64(lng), s.Radius).Add(s.Center)
			path = append(path, v)
		}
		paths = append(paths, path)
	}
	for lng := 0; lng <= 360; lng += n {
		var path geom.Path
		for lat := -90 + o; lat <= 90-o; lat++ {
			v := LatLngToXYZ(float64(lat), float64(lng), s.Radius).Add(s.Center)
			path = append(path, v)
		}
		paths = append(paths, path)
	}
	return paths
}

func LatLngToXYZ(lat, lng, radius float64) geom.Vector {
	lat, lng = geom.Radians(lat), geom.Radians(lng)
	x := radius * math.Cos(lat) * math.Cos(lng)
	y := radius * math.Cos(lat) * math.Sin(lng)
	z := radius * math.Sin(lat)
	return geom.Vector{x, y, z}
}

type OutlineSphere struct {
	Sphere
	Eye geom.Vector
	Up  geom.Vector
}

func NewOutlineSphere(eye, up, center geom.Vector, radius float64) *OutlineSphere {
	sphere := NewSphere(center, radius)
	return &OutlineSphere{*sphere, eye, up}
}

func (s *OutlineSphere) Paths() geom.Paths {
	var path geom.Path
	center := s.Sphere.Center
	radius := s.Sphere.Radius

	hyp := center.Sub(s.Eye).Length()
	opp := radius
	theta := math.Asin(opp / hyp)
	adj := opp / math.Tan(theta)
	d := math.Cos(theta) * adj
	r := math.Sin(theta) * adj

	w := center.Sub(s.Eye).Normalize()
	u := w.Cross(s.Up).Normalize()
	v := w.Cross(u).Normalize()
	c := s.Eye.Add(w.MulScalar(d))
	for i := 0; i <= 360; i++ {
		a := geom.Radians(float64(i))
		p := c
		p = p.Add(u.MulScalar(math.Cos(a) * r))
		p = p.Add(v.MulScalar(math.Sin(a) * r))
		path = append(path, p)
	}
	return geom.Paths{path}
}

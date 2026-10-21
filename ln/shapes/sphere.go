package shapes

import (
	"github.com/loyality7/ln/ln/geom"
	"math"
	"math/rand"
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

func (s *Sphere) Paths4() geom.Paths {
	var paths geom.Paths
	var seen []geom.Vector
	var radii []float64
	for i := 0; i < 140; i++ {
		var v geom.Vector
		var m float64
		for {
			v = geom.RandomUnitVector()
			m = rand.Float64()*0.25 + 0.05
			ok := true
			for i, other := range seen {
				threshold := m + radii[i] + 0.02
				if other.Sub(v).Length() < threshold {
					ok = false
					break
				}
			}
			if ok {
				seen = append(seen, v)
				radii = append(radii, m)
				break
			}
		}
		p := v.Cross(geom.RandomUnitVector()).Normalize()
		q := p.Cross(v).Normalize()
		n := rand.Intn(4) + 1
		for k := 0; k < n; k++ {
			var path geom.Path
			for j := 0; j <= 360; j += 5 {
				a := geom.Radians(float64(j))
				x := v
				x = x.Add(p.MulScalar(math.Cos(a) * m))
				x = x.Add(q.MulScalar(math.Sin(a) * m))
				x = x.Normalize()
				x = x.MulScalar(s.Radius).Add(s.Center)
				path = append(path, x)
			}
			paths = append(paths, path)
			m *= 0.75
		}
	}
	return paths
}

func (s *Sphere) Paths3() geom.Paths {
	var paths geom.Paths
	for i := 0; i < 20000; i++ {
		v := geom.RandomUnitVector()
		v = v.MulScalar(s.Radius).Add(s.Center)
		paths = append(paths, geom.Path{v, v})
	}
	return paths
}

func (s *Sphere) Paths2() geom.Paths {
	var equator geom.Path
	for lng := 0; lng <= 360; lng++ {
		v := LatLngToXYZ(0, float64(lng), s.Radius)
		equator = append(equator, v)
	}
	var paths geom.Paths
	for i := 0; i < 100; i++ {
		m := geom.Identity()
		for j := 0; j < 3; j++ {
			v := geom.RandomUnitVector()
			m = m.Rotate(v, rand.Float64()*2*math.Pi)
		}
		m = m.Translate(s.Center)
		paths = append(paths, equator.Transform(m))
	}
	return paths
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

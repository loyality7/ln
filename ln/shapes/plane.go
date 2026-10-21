package shapes

import (
	"github.com/loyality7/ln/ln/geom"
)

type Plane struct {
	Point  geom.Vector
	Normal geom.Vector
}

func (p *Plane) IntersectSegment(v0, v1 geom.Vector) (geom.Vector, bool) {
	u := v1.Sub(v0)
	w := v0.Sub(p.Point)
	d := p.Normal.Dot(u)
	n := -p.Normal.Dot(w)
	if d > -geom.EPS && d < geom.EPS {
		return geom.Vector{}, false
	}
	t := n / d
	if t < 0 || t > 1 {
		return geom.Vector{}, false
	}
	v := v0.Add(u.MulScalar(t))
	return v, true
}

func (p *Plane) IntersectTriangle(t *Triangle) (geom.Vector, geom.Vector, bool) {
	v1, ok1 := p.IntersectSegment(t.V1, t.V2)
	v2, ok2 := p.IntersectSegment(t.V2, t.V3)
	v3, ok3 := p.IntersectSegment(t.V3, t.V1)
	if ok1 && ok2 {
		return v1, v2, true
	}
	if ok1 && ok3 {
		return v1, v3, true
	}
	if ok2 && ok3 {
		return v2, v3, true
	}
	return geom.Vector{}, geom.Vector{}, false
}

package solid

import (
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/model"
)

// Extrude lifts a polygon (any simple outline, convex or not) from z=0 to z=h.
func Extrude(poly []Pt, h float64) *model.Mesh {
	if area(poly) < 0 { // want counter-clockwise
		for i, j := 0, len(poly)-1; i < j; i, j = i+1, j-1 {
			poly[i], poly[j] = poly[j], poly[i]
		}
	}
	var b builder
	n := len(poly)
	for i := 0; i < n; i++ {
		p, q := poly[i], poly[(i+1)%n]
		b.add(geom.V(p.X, p.Y, 0), geom.V(q.X, q.Y, 0), geom.V(q.X, q.Y, h))
		b.add(geom.V(p.X, p.Y, 0), geom.V(q.X, q.Y, h), geom.V(p.X, p.Y, h))
	}
	for _, t := range earClip(poly) {
		a, c, d := poly[t[0]], poly[t[1]], poly[t[2]]
		b.add(geom.V(a.X, a.Y, h), geom.V(c.X, c.Y, h), geom.V(d.X, d.Y, h))
		b.add(geom.V(a.X, a.Y, 0), geom.V(d.X, d.Y, 0), geom.V(c.X, c.Y, 0))
	}
	return Orient(b.mesh())
}

func area(p []Pt) float64 {
	var a float64
	for i := range p {
		q := p[(i+1)%len(p)]
		a += p[i].X*q.Y - q.X*p[i].Y
	}
	return a / 2
}

func cross(a, b, c Pt) float64 { return (b.X-a.X)*(c.Y-a.Y) - (b.Y-a.Y)*(c.X-a.X) }

func inside(p, a, b, c Pt) bool {
	return cross(a, b, p) >= 0 && cross(b, c, p) >= 0 && cross(c, a, p) >= 0
}

// earClip triangulates a counter-clockwise simple polygon.
func earClip(p []Pt) [][3]int {
	idx := make([]int, len(p))
	for i := range idx {
		idx[i] = i
	}
	var out [][3]int
	for len(idx) > 3 {
		cut := false
		for k := range idx {
			i0, i1, i2 := idx[(k+len(idx)-1)%len(idx)], idx[k], idx[(k+1)%len(idx)]
			if cross(p[i0], p[i1], p[i2]) <= 0 {
				continue // reflex corner
			}
			ear := true
			for _, o := range idx {
				if o != i0 && o != i1 && o != i2 && inside(p[o], p[i0], p[i1], p[i2]) {
					ear = false
					break
				}
			}
			if ear {
				out = append(out, [3]int{i0, i1, i2})
				idx = append(idx[:k], idx[k+1:]...)
				cut = true
				break
			}
		}
		if !cut {
			break // degenerate polygon
		}
	}
	if len(idx) == 3 {
		out = append(out, [3]int{idx[0], idx[1], idx[2]})
	}
	return out
}

package solid

import (
	"math"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/model"
)

// Lathe spins a profile (X = distance from the z axis, Y = height) around the
// z axis. Profile points are used as given, so corners stay sharp; use Smooth
// for curved profiles. Start/end the profile on the axis (X=0) for a closed solid.
func Lathe(profile []Pt, segments int) *model.Mesh {
	var b builder
	ring := func(i, j int) geom.Vector {
		th := 2 * math.Pi * float64(j) / float64(segments)
		return geom.V(profile[i].X*math.Cos(th), profile[i].X*math.Sin(th), profile[i].Y)
	}
	for i := 0; i+1 < len(profile); i++ {
		for j := 0; j < segments; j++ {
			b.add(ring(i, j), ring(i+1, j), ring(i+1, j+1))
			b.add(ring(i, j), ring(i+1, j+1), ring(i, j+1))
		}
	}
	m := b.mesh()
	if profile[0].X == 0 && profile[len(profile)-1].X == 0 {
		Orient(m)
	}
	return m
}

// Smooth fits a Catmull-Rom curve through pts with n points per span, for
// vases, bottles and other curved profiles.
func Smooth(pts []Pt, perSpan int) []Pt {
	at := func(i int) Pt { return pts[max(0, min(len(pts)-1, i))] }
	var out []Pt
	for i := 0; i+1 < len(pts); i++ {
		for k := 0; k < perSpan; k++ {
			t := float64(k) / float64(perSpan)
			p0, p1, p2, p3 := at(i-1), at(i), at(i+1), at(i+2)
			c := func(a, b, c, d float64) float64 {
				return 0.5 * (2*b + (-a+c)*t + (2*a-5*b+4*c-d)*t*t + (-a+3*b-3*c+d)*t*t*t)
			}
			out = append(out, Pt{c(p0.X, p1.X, p2.X, p3.X), c(p0.Y, p1.Y, p2.Y, p3.Y)})
		}
	}
	return append(out, pts[len(pts)-1])
}

// Tube sweeps a circle of the given radius along path. radius(t) lets it vary
// along the path (t in [0,1]). closed joins the ends into a loop (knots, rings);
// otherwise the ends are capped.
func Tube(path []geom.Vector, radius func(t float64) float64, sides int, closed bool) *model.Mesh {
	n := len(path)
	if closed {
		path = append(append([]geom.Vector{}, path...), path[0])
		n++
	}
	tan := make([]geom.Vector, n)
	for i := range path {
		a, b := path[max(0, i-1)], path[min(n-1, i+1)]
		if closed {
			a, b = path[(i-1+n-1)%(n-1)], path[(i+1)%(n-1)]
		}
		tan[i] = b.Sub(a).Normalize()
	}
	// rotation-minimizing frames: carry the normal along without spinning it
	nor := make([]geom.Vector, n)
	nor[0] = tan[0].Cross(geom.Vector{X: 0.31, Y: 0.57, Z: 0.76}).Normalize()
	for i := 1; i < n; i++ {
		nor[i] = nor[i-1].Sub(tan[i].MulScalar(nor[i-1].Dot(tan[i]))).Normalize()
	}
	if closed { // spread the leftover twist over the loop so the seam matches
		end := nor[n-1].Sub(tan[0].MulScalar(nor[n-1].Dot(tan[0]))).Normalize()
		phi := math.Atan2(nor[0].Cross(end).Dot(tan[0]), nor[0].Dot(end))
		for i := range nor {
			nor[i] = geom.Rotate(tan[i], phi*float64(i)/float64(n-1)).MulDirection(nor[i])
		}
	}
	var b builder
	ring := func(i, j int) geom.Vector {
		th := 2 * math.Pi * float64(j) / float64(sides)
		bin := tan[i].Cross(nor[i])
		r := radius(float64(i) / float64(n-1))
		return path[i].Add(nor[i].MulScalar(r * math.Cos(th))).Add(bin.MulScalar(r * math.Sin(th)))
	}
	for i := 0; i+1 < n; i++ {
		for j := 0; j < sides; j++ {
			b.add(ring(i, j), ring(i+1, j), ring(i+1, j+1))
			b.add(ring(i, j), ring(i+1, j+1), ring(i, j+1))
		}
	}
	if !closed {
		for j := 0; j < sides; j++ {
			b.add(path[0], ring(0, j+1), ring(0, j))
			b.add(path[n-1], ring(n-1, j), ring(n-1, j+1))
		}
	}
	m := b.mesh()
	return Orient(m)
}

// Constant is a radius function for Tube.
func Constant(r float64) func(float64) float64 { return func(float64) float64 { return r } }

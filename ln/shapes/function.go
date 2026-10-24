package shapes

import (
	"math"

	"github.com/loyality7/ln/ln/geom"
)

type Direction int

const (
	Above Direction = iota
	Below
)

type Function struct {
	Function  func(x, y float64) float64
	Box       geom.Box
	Direction Direction
}

func NewFunction(function func(x, y float64) float64, box geom.Box, direction Direction) Shape {
	return &Function{function, box, direction}
}

func (f *Function) Compile() {
}

func (f *Function) BoundingBox() geom.Box {
	return f.Box
}

func (f *Function) Contains(v geom.Vector, eps float64) bool {
	if f.Direction == Below {
		return v.Z < f.Function(v.X, v.Y)
	} else {
		return v.Z > f.Function(v.X, v.Y)
	}
}

func (f *Function) Intersect(ray geom.Ray) Hit {
	step := 1.0 / 64
	sign := f.Contains(ray.Position(step), 0)
	for t := step; t < 10; t += step {
		v := ray.Position(t)
		if f.Contains(v, 0) != sign && f.Box.Contains(v) {
			return Hit{f, t, f.normalAt(v)}
		}
	}
	return NoHit
}

// normalAt is the upward normal of the surface z = f(x,y) from finite differences.
func (f *Function) normalAt(v geom.Vector) geom.Vector {
	const e = 1e-3
	fx := (f.Function(v.X+e, v.Y) - f.Function(v.X-e, v.Y)) / (2 * e)
	fy := (f.Function(v.X, v.Y+e) - f.Function(v.X, v.Y-e)) / (2 * e)
	return geom.Vector{X: -fx, Y: -fy, Z: 1}.Normalize()
}

func (f *Function) Paths() geom.Paths {
	var paths geom.Paths
	fine := 1.0 / 256
	for a := 0; a < 360; a += 5 {
		var path geom.Path
		for r := 0.0; r <= 8.0; r += fine {
			x := math.Cos(geom.Radians(float64(a))) * r
			y := math.Sin(geom.Radians(float64(a))) * r
			z := f.Function(x, y)
			o := -math.Pow(-z, 1.4)
			x = math.Cos(geom.Radians(float64(a))-o) * r
			y = math.Sin(geom.Radians(float64(a))-o) * r
			z = math.Min(z, f.Box.Max.Z)
			z = math.Max(z, f.Box.Min.Z)
			path = append(path, geom.Vector{x, y, z})
		}
		paths = append(paths, path)
	}
	return paths
}

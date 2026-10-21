// Package world builds recognizable real-world objects (trees, houses,
// fences, clouds, terrain) out of ln shapes. Objects with silhouette outlines
// need the camera, so rebuild them per frame from the current eye.
package world

import (
	"math"
	"math/rand"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/model"
	"github.com/loyality7/ln/ln/shapes"
)

// Tree is a branching trunk with leaf clusters at the tips. Same seed = same
// tree; sway (radians, animate it) bends the branches.
func Tree(eye, up, base geom.Vector, height float64, seed int64, sway float64) []shapes.Shape {
	r := rand.New(rand.NewSource(seed))
	var out []shapes.Shape
	leaf := height * 0.13
	var grow func(p, dir geom.Vector, length, radius float64, depth int)
	grow = func(p, dir geom.Vector, length, radius float64, depth int) {
		tip := p.Add(dir.MulScalar(length))
		out = append(out, shapes.NewTransformedOutlineCylinder(eye, up, p, tip, radius))
		if depth == 0 {
			out = append(out, shapes.NewOutlineSphere(eye, up, tip, leaf*(0.8+0.5*r.Float64())))
			return
		}
		n := 2 + r.Intn(2)
		for i := 0; i < n; i++ {
			axis := dir.Cross(geom.RandomUnitVectorFrom(r)).Normalize()
			ang := 0.45 + 0.35*r.Float64() + sway*0.12*float64(4-depth)
			grow(tip, geom.Rotate(axis, ang).MulDirection(dir), length*0.72, radius*0.62, depth-1)
		}
	}
	grow(base, geom.V(0, 0, 1), height*0.42, height*0.045, 3)
	return out
}

// Cloud is a cluster of puffs around center.
func Cloud(eye, up, center geom.Vector, size float64, seed int64) []shapes.Shape {
	r := rand.New(rand.NewSource(seed))
	var out []shapes.Shape
	for i := 0; i < 6; i++ {
		o := geom.V((r.Float64()-0.5)*size*2, (r.Float64()-0.5)*size*0.6, (r.Float64()-0.3)*size*0.3)
		out = append(out, shapes.NewOutlineSphere(eye, up, center.Add(o), size*(0.35+0.3*r.Float64())))
	}
	return out
}

// Terrain is a hatch-only heightfield over [-half,half]^2.
func Terrain(height func(x, y float64) float64, half float64, n int) shapes.Shape {
	var tris []*shapes.Triangle
	step := 2 * half / float64(n)
	p := func(i, j int) geom.Vector {
		x, y := -half+float64(i)*step, -half+float64(j)*step
		return geom.V(x, y, height(x, y))
	}
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			a, b, c, d := p(i, j), p(i+1, j), p(i+1, j+1), p(i, j+1)
			tris = append(tris, shapes.NewTriangle(a, b, c), shapes.NewTriangle(a, c, d))
		}
	}
	return shapes.NoOutline{Shape: model.NewMesh(tris)}
}

// Smoothstep eases 0..1 between edge0 and edge1; handy for blending terrain.
func Smoothstep(e0, e1, x float64) float64 {
	t := math.Max(0, math.Min(1, (x-e0)/(e1-e0)))
	return t * t * (3 - 2*t)
}

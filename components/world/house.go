package world

import (
	"math"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/shapes"
)

func box(x0, y0, z0, x1, y1, z1 float64) shapes.Shape {
	return shapes.NewCube(geom.V(x0, y0, z0), geom.V(x1, y1, z1))
}

// House is a gabled cottage w wide (x), d deep (y), h to the eaves, with door,
// windows, chimney and steps. Its door faces -y before yaw (radians).
func House(origin geom.Vector, yaw, w, d, h float64) []shapes.Shape {
	var parts []shapes.Shape
	theta := geom.Radians(27)
	run := d/2 + 0.35 // eave overhang
	slope := run / math.Cos(theta)
	hw := slope / 2
	zr := h + slope*math.Sin(theta)

	parts = append(parts, box(-w/2, -d/2, 0, w/2, d/2, h)) // walls
	// door, steps, windows
	dw := w * 0.16
	parts = append(parts, box(-dw/2, -d/2-0.1, 0, dw/2, -d/2, h*0.72))
	parts = append(parts, box(-dw*0.8, -d/2-0.45, 0, dw*0.8, -d/2, 0.12))
	for _, sx := range []float64{-1, 1} {
		cx := sx * w * 0.3
		parts = append(parts, box(cx-w*0.09, -d/2-0.07, h*0.4, cx+w*0.09, -d/2, h*0.82))
		parts = append(parts, box(cx-w*0.09, d/2, h*0.4, cx+w*0.09, d/2+0.07, h*0.82))
	}
	parts = append(parts, box(w*0.18, d*0.05, h, w*0.18+0.4, d*0.05+0.4, zr+0.25)) // chimney

	// roof: two tilted slabs meeting at the ridge, plus gable triangles
	slab := func(side float64) shapes.Shape {
		s := shapes.NewCube(geom.V(-w/2-0.35, -hw, -0.07), geom.V(w/2+0.35, hw, 0.07))
		a := -theta // geom.Rotate turns clockwise about its axis
		if side > 0 {
			a = math.Pi + theta
		}
		return shapes.NewTransformedShape(s, geom.Rotate(geom.V(1, 0, 0), a).Translate(geom.V(0, side*hw*math.Cos(theta), zr-hw*math.Sin(theta))))
	}
	parts = append(parts, slab(-1), slab(1))
	for _, sx := range []float64{-1, 1} {
		x := sx * w / 2
		parts = append(parts, shapes.NewTriangle(geom.V(x, -d/2, h), geom.V(x, d/2, h), geom.V(x, 0, zr-0.08)))
	}

	m := geom.Rotate(geom.V(0, 0, 1), yaw).Translate(origin)
	for i, p := range parts {
		parts[i] = shapes.NewTransformedShape(p, m)
	}
	return parts
}

// Fence runs posts and two rails from a to b.
func Fence(a, b geom.Vector, posts int, height float64) []shapes.Shape {
	var out []shapes.Shape
	dir := b.Sub(a)
	for i := 0; i < posts; i++ {
		p := a.Add(dir.MulScalar(float64(i) / float64(posts-1)))
		out = append(out, box(p.X-0.07, p.Y-0.07, p.Z, p.X+0.07, p.Y+0.07, p.Z+height))
	}
	yaw := math.Atan2(dir.Y, dir.X)
	for _, z := range []float64{0.35, 0.7} {
		rail := box(0, -0.03, z*height-0.04, dir.Length(), 0.03, z*height+0.04)
		out = append(out, shapes.NewTransformedShape(rail, geom.Rotate(geom.V(0, 0, 1), -yaw).Translate(a)))
	}
	return out
}

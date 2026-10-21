package main

import (
	"math"
	"os"

	"github.com/fogleman/gg"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/ink"
	"github.com/loyality7/ln/ln/scene"
	"github.com/loyality7/ln/ln/shapes"
)

func box(x0, y0, z0, x1, y1, z1 float64) shapes.Shape {
	return shapes.NewCube(geom.V(x0, y0, z0), geom.V(x1, y1, z1))
}

// roofSlab is a thin slab tilted by theta whose upper edge meets the ridge.
func roofSlab(side, hw, theta, zr float64) shapes.Shape {
	slab := shapes.NewCube(geom.V(-2.5, -hw, -0.1), geom.V(2.5, hw, 0.1))
	a := -theta // geom.Rotate is clockwise about its axis
	if side > 0 {
		a = math.Pi + theta
	}
	m := geom.Rotate(geom.V(1, 0, 0), a).Translate(geom.V(0, side*hw*math.Cos(theta), zr-hw*math.Sin(theta)))
	return shapes.NewTransformedShape(slab, m)
}

func main() {
	eye, center, up := geom.V(9, -10, 5), geom.V(0, 0, 1.6), geom.V(0, 0, 1)
	s := &scene.Scene{}

	// ground, house body, door, windows
	s.Add(box(-9, -7, -0.3, 9, 7, 0))
	s.Add(box(-2, -1.5, 0, 2, 1.5, 2))
	s.Add(box(-0.5, -1.6, 0, 0.5, -1.5, 1.5))
	s.Add(box(0.8, -1.56, 0.9, 1.7, -1.5, 1.6))
	s.Add(box(-1.7, -1.56, 0.9, -0.8, -1.5, 1.6))
	s.Add(box(1.0, 0.45, 2.2, 1.45, 0.9, 3.6)) // chimney (overlaps roof)

	// roof
	theta := geom.Radians(25)
	s.Add(roofSlab(-1, 1.0, theta, 2.85))
	s.Add(roofSlab(1, 1.0, theta, 2.85))

	// tree
	trunkA, trunkB := geom.V(-3.8, -1.8, 0), geom.V(-3.8, -1.8, 2.2)
	s.Add(shapes.NewTransformedOutlineCylinder(eye, up, trunkA, trunkB, 0.28))
	for _, c := range []struct{ x, y, z, r float64 }{
		{-3.8, -1.8, 3.0, 1.1}, {-4.4, -1.5, 2.5, 0.8}, {-3.2, -2.2, 2.6, 0.8}, {-3.7, -1.9, 4.0, 0.8}, {-4.3, -2.3, 3.5, 0.7},
	} {
		s.Add(shapes.NewOutlineSphere(eye, up, geom.V(c.x, c.y, c.z), c.r))
	}

	w, h := 1400.0, 1000.0
	st := ink.DefaultStyle()
	dr := ink.Render(s, eye, center, up, w, h, 32, st)
	os.MkdirAll("out", 0755)
	gg.SavePNG("out/house_ink.png", dr.Image(w, h))
	os.WriteFile("out/house_ink.svg", []byte(dr.SVG()), 0644)
}

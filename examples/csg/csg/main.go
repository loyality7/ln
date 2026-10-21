package main

import (
	"fmt"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/scene"
	"github.com/loyality7/ln/ln/shapes"
)

func main() {
	shape := shapes.NewDifference(
		shapes.NewIntersection(
			shapes.NewSphere(geom.Vector{}, 1),
			shapes.NewCube(geom.Vector{-0.8, -0.8, -0.8}, geom.Vector{0.8, 0.8, 0.8}),
		),
		shapes.NewCylinder(0.4, -2, 2),
		shapes.NewTransformedShape(shapes.NewCylinder(0.4, -2, 2), geom.Rotate(geom.Vector{1, 0, 0}, geom.Radians(90))),
		shapes.NewTransformedShape(shapes.NewCylinder(0.4, -2, 2), geom.Rotate(geom.Vector{0, 1, 0}, geom.Radians(90))),
	)
	for i := 0; i < 90; i += 2 {
		fmt.Println(i)
		scene := scene.Scene{}
		m := geom.Rotate(geom.Vector{0, 0, 1}, geom.Radians(float64(i)))
		scene.Add(shapes.NewTransformedShape(shape, m))
		eye := geom.Vector{0, 6, 2}
		center := geom.Vector{0, 0, 0}
		up := geom.Vector{0, 0, 1}
		width := 750.0
		height := 750.0
		paths := scene.Render(eye, center, up, width, height, 20, 0.1, 100, 0.01)
		paths.WriteToPNG(fmt.Sprintf("out%03d.png", i), width, height)
	}
}

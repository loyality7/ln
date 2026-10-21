package main

import (
	"math/rand"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/scene"
	"github.com/loyality7/ln/ln/shapes"
)

func cube(x, y, z float64) shapes.Shape {
	size := 0.5
	v := geom.Vector{x, y, z}
	return shapes.NewCube(v.SubScalar(size), v.AddScalar(size))
}

func main() {
	scene := scene.Scene{}
	for x := -2; x <= 2; x++ {
		for y := -2; y <= 2; y++ {
			z := rand.Float64()
			scene.Add(cube(float64(x), float64(y), z))
		}
	}
	eye := geom.Vector{6, 5, 3}
	center := geom.Vector{0, 0, 0}
	up := geom.Vector{0, 0, 1}
	width := 1920.0
	height := 1200.0
	fovy := 30.0
	paths := scene.Render(eye, center, up, width, height, fovy, 0.1, 100, 0.01)
	paths.WriteToPNG("out.png", width, height)
	paths.WriteToSVG("out.svg", width, height)
}

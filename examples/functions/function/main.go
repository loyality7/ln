package main

import (
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/scene"
	"github.com/loyality7/ln/ln/shapes"
)

func function(x, y float64) float64 {
	return -1 / (x*x + y*y)
	// return math.Cos(x*y) * (x*x - y*y)
}

func main() {
	scene := scene.Scene{}
	box := geom.Box{geom.Vector{-2, -2, -4}, geom.Vector{2, 2, 2}}
	scene.Add(shapes.NewFunction(function, box, shapes.Below))
	eye := geom.Vector{3, 0, 3}
	center := geom.Vector{1.1, 0, 0}
	up := geom.Vector{0, 0, 1}
	width := 1024.0
	height := 1024.0
	paths := scene.Render(eye, center, up, width, height, 50, 0.1, 100, 0.01)
	paths.WriteToPNG("out.png", width, height)
	paths.WriteToSVG("out.svg", width, height)
	// paths.Print()
}

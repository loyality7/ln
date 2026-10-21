package main

import (
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/model"
	"github.com/loyality7/ln/ln/scene"
)

func main() {
	scene := scene.Scene{}
	mesh, err := model.LoadBinarySTL("bowser.stl")
	// mesh, err := model.LoadOBJ("../pt/examples/bunny.obj")
	if err != nil {
		panic(err)
	}
	mesh.FitInside(geom.Box{geom.Vector{-1, -1, -1}, geom.Vector{1, 1, 1}}, geom.Vector{0.5, 0.5, 0.5})
	cubes := mesh.Voxelize(1.0 / 64)
	for _, cube := range cubes {
		scene.Add(cube)
	}
	eye := geom.Vector{-1, -2, 0}
	center := geom.Vector{0, 0, 0}
	up := geom.Vector{0, 0, 1}
	width := 1024.0 * 2
	height := 1024.0 * 2
	paths := scene.Render(eye, center, up, width, height, 60, 0.1, 100, 0.01)
	paths.WriteToPNG("out.png", width, height)
	// paths.WriteToSVG("out.svg", width, height)
	// paths.Print()
}

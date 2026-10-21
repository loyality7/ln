package main

import (
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/model"
	"github.com/loyality7/ln/ln/scene"
	"github.com/loyality7/ln/ln/shapes"
)

func main() {
	scene := scene.Scene{}
	mesh, err := model.LoadOBJ("assets/models/suzanne.obj")
	if err != nil {
		panic(err)
	}
	mesh.UnitCube()
	scene.Add(shapes.NewTransformedShape(mesh, geom.Rotate(geom.Vector{0, 1, 0}, 0.5)))
	// scene.Add(mesh)
	eye := geom.Vector{-0.5, 0.5, 2}
	center := geom.Vector{}
	up := geom.Vector{0, 1, 0}
	width := 1024.0
	height := 1024.0
	paths := scene.Render(eye, center, up, width, height, 35, 0.1, 100, 0.01)
	paths.WriteToPNG("out.png", width, height)
}

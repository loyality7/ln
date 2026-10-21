package main

import (
	"fmt"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/model"
	"github.com/loyality7/ln/ln/shapes"
)

const Slices = 32
const Size = 1024

func main() {
	// mesh, err := model.LoadBinarySTL("bowser.stl")
	mesh, err := model.LoadOBJ("assets/models/suzanne.obj")
	if err != nil {
		panic(err)
	}
	mesh.FitInside(geom.Box{geom.Vector{-1, -1, -1}, geom.Vector{1, 1, 1}}, geom.Vector{0.5, 0.5, 0.5})
	for i := 0; i < Slices; i++ {
		fmt.Printf("slice%04d\n", i)
		p := (float64(i)/(Slices-1))*2 - 1
		point := geom.Vector{0, 0, p}
		plane := shapes.Plane{point, geom.Vector{0, 0, 1}}
		paths := model.IntersectPlane(&plane, mesh)
		paths = paths.Transform(geom.Scale(geom.Vector{Size / 2, Size / 2, 1}).Translate(geom.Vector{Size / 2, Size / 2, 0}))
		paths.WriteToPNG(fmt.Sprintf("slice%04d.png", i), Size, Size)
		// paths.WriteToSVG(fmt.Sprintf("slice%04d.svg", i), Size, Size)
	}
}

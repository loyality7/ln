package main

import (
	"fmt"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/model"
	"github.com/loyality7/ln/ln/scene"
	"github.com/loyality7/ln/ln/shapes"
)

type Shape struct {
	model.Mesh
}

func (s *Shape) Paths() geom.Paths {
	var result geom.Paths
	// for i := 0; i < 360; i++ {
	// 	fmt.Println(i)
	// 	a := geom.Radians(float64(i))
	// 	x := math.Cos(a)
	// 	y := math.Sin(a)
	// 	plane := shapes.Plane{geom.Vector{}, geom.Vector{x, y, 0}}
	// 	paths := model.IntersectPlane(&plane, &s.Mesh)
	// 	result = append(result, paths...)
	// }
	for i := 0; i <= 100; i++ {
		fmt.Println(i)
		p := float64(i) / 100
		plane := shapes.Plane{geom.Vector{0, 0, p*2 - 1}, geom.Vector{0, 0, 1}}
		result = append(result, model.IntersectPlane(&plane, &s.Mesh)...)
		plane = shapes.Plane{geom.Vector{p*2 - 1, 0, 0}, geom.Vector{1, 0, 0}}
		result = append(result, model.IntersectPlane(&plane, &s.Mesh)...)
		plane = shapes.Plane{geom.Vector{0, p*2 - 1, 0}, geom.Vector{0, 1, 0}}
		result = append(result, model.IntersectPlane(&plane, &s.Mesh)...)
	}
	return result
}

func main() {
	scene := scene.Scene{}
	mesh, err := model.LoadBinarySTL("bowser.stl")
	// mesh, err := model.LoadOBJ("../pt/examples/bunny.obj")
	if err != nil {
		panic(err)
	}
	mesh.FitInside(geom.Box{geom.Vector{-1, -1, -1}, geom.Vector{1, 1, 1}}, geom.Vector{0.5, 0.5, 0.5})
	scene.Add(&Shape{*mesh})
	// scene.Add(mesh)
	eye := geom.Vector{-2, 2, 1}
	center := geom.Vector{0, 0, 0}
	up := geom.Vector{0, 0, 1}
	width := 1024.0 * 2
	height := 1024.0 * 2
	paths := scene.Render(eye, center, up, width, height, 50, 0.1, 100, 0.01)
	paths.WriteToPNG("out.png", width, height)
	// paths.WriteToSVG("out.svg", width, height)
	// paths.Print()
}

package main

import (
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/scene"
	"github.com/loyality7/ln/ln/shapes"
)

func main() {
	// create a scene and add a single cube
	scene := scene.Scene{}
	scene.Add(shapes.NewCube(geom.Vector{-1, -1, -1}, geom.Vector{1, 1, 1}))

	// define camera parameters
	eye := geom.Vector{4, 3, 2}    // camera position
	center := geom.Vector{0, 0, 0} // camera looks at
	up := geom.Vector{0, 0, 1}     // up direction

	// define rendering parameters
	width := 1024.0  // rendered width
	height := 1024.0 // rendered height
	fovy := 50.0     // vertical field of view, degrees
	znear := 0.1     // near z plane
	zfar := 10.0     // far z plane
	step := 0.01     // how finely to chop the paths for visibility testing

	// compute 2D paths that depict the 3D scene
	paths := scene.Render(eye, center, up, width, height, fovy, znear, zfar, step)

	// save the result as a png
	paths.WriteToPNG("out.png", width, height)

	// save the result as an svg
	paths.WriteToSVG("out.svg", width, height)
}

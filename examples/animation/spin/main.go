package main

import (
	"math"

	"github.com/loyality7/ln/ln/anim"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/scene"
	"github.com/loyality7/ln/ln/shapes"
)

// Toolkit demo: the pen draws the scene in, the cube pops with OutBack while
// spinning, a sphere orbits it, and the camera glides along a spline.
func main() {
	w, h := 800.0, 800.0
	flight := anim.Spline([]geom.Vector{{X: 9, Y: 0, Z: 6}, {X: 0, Y: 9, Z: 2}, {X: -9, Y: 0, Z: 5}, {X: 0, Y: -9, Z: 3}, {X: 9, Y: 0, Z: 6}})
	a := anim.Animation(func(t float64) geom.Paths {
		s := &scene.Scene{}
		pop := anim.Segment(t, 0, 0.25, anim.OutBack)
		spin := geom.Rotate(geom.Vector{Z: 1}, t*2*math.Pi*2).Rotate(geom.Vector{X: 1}, t*2*math.Pi)
		cube := shapes.NewCube(geom.Vector{X: -1, Y: -1, Z: -1}, geom.Vector{X: 1, Y: 1, Z: 1})
		s.Add(shapes.NewTransformedShape(cube, spin.Scale(geom.Vector{X: pop, Y: pop, Z: pop})))
		a := t * 2 * math.Pi * 3
		s.Add(shapes.NewSphere(geom.Vector{X: math.Cos(a) * 3, Y: math.Sin(a) * 3, Z: math.Sin(a*2) * 0.8}, 0.6*pop))
		cam := anim.Camera{Eye: flight(t), Up: geom.Vector{Z: 1}, Fovy: 35}
		paths := cam.Render(s, w, h, 0.02)
		return anim.Reveal(paths, anim.Segment(t, 0, 0.2, anim.InOutCubic))
	})
	if err := a.WriteVideo("out/spin.mp4", 120, 30, w, h); err != nil {
		panic(err)
	}
}

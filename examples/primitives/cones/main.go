package main

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/fogleman/pt/pt"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/scene"
	"github.com/loyality7/ln/ln/shapes"
)

type Tree struct {
	shapes.Shape
	V0, V1 geom.Vector
}

func (t *Tree) Paths() geom.Paths {
	paths := t.Shape.Paths()
	for i := 0; i < 128; i++ {
		p := math.Pow(rand.Float64(), 1.5)*0.5 + 0.5
		c := t.V0.Add(t.V1.Sub(t.V0).MulScalar(p))
		a := rand.Float64() * 2 * math.Pi
		l := (1 - p) * 8
		d := geom.Vector{math.Cos(a), math.Sin(a), -2.75}.Normalize()
		e := c.Add(d.MulScalar(l))
		paths = append(paths, geom.Path{c, e})
	}
	return paths
}

func run(seed int) {
	// fmt.Println(seed)
	rand.Seed(int64(seed))
	eye := geom.Vector{}
	center := geom.Vector{0.5, 0, 8}
	up := geom.Vector{0, 0, 1}
	scene := scene.Scene{}
	n := 9.0
	points := pt.PoissonDisc(-n, -n, n, n, 2, 32)
	for _, p := range points {
		z := rand.Float64()*5 + 20
		v0 := geom.Vector{p.X, p.Y, 0}
		v1 := geom.Vector{p.X, p.Y, z}
		if v0.Distance(eye) < 1 {
			continue
		}
		c := shapes.NewTransformedOutlineCone(eye, up, v0, v1, z/64)
		tree := Tree{c, v0, v1}
		scene.Add(&tree)
	}
	width := 2048.0
	height := 2048.0
	fovy := 90.0
	paths := scene.Render(eye, center, up, width, height, fovy, 0.1, 100, 0.1)
	path := fmt.Sprintf("out%d.png", seed)
	paths.WriteToPNG(path, width, height)
	paths.Print()
}

func main() {
	run(10)
	// for i := 0; i < 100; i++ {
	// 	run(i)
	// }
}

package main

import (
	"math"
	"os"

	"github.com/fogleman/gg"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/ink"
	"github.com/loyality7/ln/ln/model"
	"github.com/loyality7/ln/ln/scene"
	"github.com/loyality7/ln/ln/shapes"
	"github.com/loyality7/ln/ln/solid"
)

// Every object below is made from code: no mesh files, no special shape types.
func main() {
	eye, center, up := geom.V(0, -21, 11), geom.V(0, 0, 1.2), geom.V(0, 0, 1)
	s := &scene.Scene{}
	put := func(m *model.Mesh, x, y, z float64) {
		solid.Place(m, geom.Translate(geom.V(x, y, z)))
		o := solid.Outline(m, 35)
		o.Eye = eye
		s.Add(o)
	}

	s.Add(shapes.NoOutline{Shape: solid.Place(solid.Box(24, 12, 0.4), geom.Translate(geom.V(0, 0, -0.2)))})

	// 1. vase: a smooth curve spun around an axis (lathe)
	vase := solid.Lathe(solid.Smooth([]solid.Pt{{0, 0}, {0.9, 0}, {1.1, 0.5}, {0.7, 1.5}, {0.45, 2.1}, {0.62, 2.6}, {0.85, 2.95}}, 8), 48)
	put(vase, -9, 2.5, 0)

	// 2. torus, stood on edge
	torus := solid.Place(solid.Torus(1, 0.38), geom.Rotate(geom.V(1, 0, 0), geom.Radians(75)))
	put(torus, -5, 2.5, 1.4)

	// 3. five-point star: a polygon extruded
	var star []solid.Pt
	for i := 0; i < 10; i++ {
		r := 1.3
		if i%2 == 1 {
			r = 0.55
		}
		a := math.Pi/2 + float64(i)*math.Pi/5
		star = append(star, solid.Pt{X: r * math.Cos(a), Y: r * math.Sin(a)})
	}
	put(solid.Extrude(star, 0.7), -1, 2.5, 0)

	// 4. trefoil knot: a tube swept along a 3D curve
	var knot []geom.Vector
	for i := 0; i < 200; i++ {
		t := 2 * math.Pi * float64(i) / 200
		knot = append(knot, geom.V((math.Sin(t)+2*math.Sin(2*t))*0.42, (math.Cos(t)-2*math.Cos(2*t))*0.42, -math.Sin(3*t)*0.42))
	}
	put(solid.Tube(knot, solid.Constant(0.2), 20, true), 3.5, 2.5, 1.6)

	// 5. organic blob: any formula f(u,v) becomes a surface
	blob := solid.Orient(solid.Surface(96, 48, func(u, w float64) geom.Vector {
		th, ph := 2*math.Pi*u, math.Pi*w
		r := 1.1 + 0.28*math.Sin(5*th)*math.Sin(4*ph)
		return geom.V(r*math.Sin(ph)*math.Cos(th), r*math.Sin(ph)*math.Sin(th), r*math.Cos(ph))
	}))
	put(blob, 8, 2.5, 1.3)

	// 6. totem built from primitives
	put(solid.Cylinder(0.8, 1.0), -9, -2.5, 0)
	put(solid.Sphere(0.75), -9, -2.5, 1.55)
	put(solid.Cone(0.6, 1.2), -9, -2.5, 2.2)

	// 7. capsule and tilted box
	put(solid.Place(solid.Capsule(0.5, 1.4), geom.Rotate(geom.V(0, 1, 0), geom.Radians(35))), -5, -2.5, 1.0)
	put(solid.Place(solid.Box(1.6, 1.6, 1.6), geom.Rotate(geom.V(0, 0, 1), geom.Radians(30))), -1, -2.5, 0.8)

	// 8. rippled height field
	put(solid.Heightfield(3.6, 3.6, 60, 60, func(x, y float64) float64 {
		return 0.35*math.Cos(math.Hypot(x, y)*3.2) + 0.5
	}), 3.5, -2.5, 0.3)

	// 9. a ring of spheres on a spiral staircase-like path
	for i := 0; i < 9; i++ {
		a := float64(i) * 0.7
		put(solid.Sphere(0.32), 8+math.Cos(a)*1.3, -2.5+math.Sin(a)*1.3, 0.35+0.22*float64(i))
	}

	w, h := 1600.0, 900.0
	st := ink.DefaultStyle()
	st.Light = geom.V(-0.5, -0.6, 1)
	dr := ink.Render(s, eye, center, up, w, h, 36, st)
	os.MkdirAll("out", 0755)
	gg.SavePNG("out/forge_shapes.png", dr.Image(w, h))
}

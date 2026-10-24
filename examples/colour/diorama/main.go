// Command diorama paints a small 3D scene in flat colour: a stepped temple on a hill at dusk, pines
// and rocks. It is drawn as vector shapes and ink lines, so it reads as 3D but stays a drawing.
//
//	go run ./examples/colour/diorama -still 0
//	go run ./examples/colour/diorama -w 540 -h 960
package main

import (
	"math"

	"github.com/loyality7/ln/components/nature"
	"github.com/loyality7/ln/components/studio"
	"github.com/loyality7/ln/ln/ink"
	"github.com/loyality7/ln/ln/stage"
)

// paint colours a node and everything under it.
func paint(n *stage.Node, c ink.Color) *stage.Node {
	n.Paint(c)
	for _, k := range n.Children {
		paint(k, c)
	}
	return n
}

func main() {
	studio.Run(func(s *studio.Scene) {
		s.Look("colour")
		s.TimeOf("17.5")
		hill := func(x, y float64) float64 {
			peak := 3.2 * math.Exp(-(x*x+y*y)/30)
			return peak + 1.4*(nature.FBM(x*0.18+4, y*0.18, 5)-0.45)
		}
		land := s.Terrain(hill, studio.Rect{X0: -22, X1: 22, Y0: -22, Y1: 22})
		paint(land, ink.RGB(0.32, 0.5, 0.24))

		top := hill(0, 0)
		stone := ink.RGB(0.86, 0.66, 0.42)
		warm := ink.RGB(0.78, 0.3, 0.2)
		for i := 0; i < 7; i++ { // the stepped tower
			w := 4.2 - 0.52*float64(i)
			h := 0.9 - 0.05*float64(i)
			c := stone
			if i%2 == 1 {
				c = warm
			}
			paint(s.Root.Box(w, w, h).At(0, 0, top+0.45+float64(i)*0.82), c)
		}
		paint(s.Root.Cone(0.5, 1.1).At(0, 0, top+5.66), ink.RGB(0.95, 0.78, 0.25))

		s.Scatter(30, studio.Rect{X0: -16, X1: 16, Y0: -14, Y1: 14}, func(p studio.Vec, r *studio.Rand) {
			if math.Hypot(p.X, p.Y) < 5 {
				return
			}
			pine := nature.Pine(s.Root, studio.V(p.X, p.Y, hill(p.X, p.Y)), 3.2+r.Float64()*2.2, r.Int63())
			paint(pine, ink.RGB(0.14, 0.34, 0.26))
		})
		s.Camera(studio.V(9, -16, top+2.6), studio.V(0, 0, top+2.8), 38)
	})
}

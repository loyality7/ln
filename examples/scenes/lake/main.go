package main

import (
	"math"

	"github.com/loyality7/ln/components/mannequin"
	"github.com/loyality7/ln/components/nature"
	. "github.com/loyality7/ln/components/studio"
)

// ground: a lake basin in front, a beach, forested hills, and a far mountain range.
func ground(x, y float64) float64 {
	shore := nature.Smoothstep(4, 6.5, y)
	hills := nature.FBM(x*0.045+3, y*0.045, 4)
	z := -2.2 + 3.1*shore
	z += shore * hills * 2.4 * nature.Smoothstep(6, 18, y)
	z += nature.Smoothstep(14, 75, y) * nature.Ridged(x*0.012+1, y*0.012, 5) * 36
	z += nature.Smoothstep(15, 40, math.Abs(x)) * (4 + 7*hills) * nature.Smoothstep(-8, 10, y)
	return z
}

func main() {
	Run(func(s *Scene) {
		s.Look("manga")
		s.TimeOf("golden")
		s.Camera(V(0, -14, 1.4), V(0, 6, 0.6), 40)

		s.Terrain(ground, Rect{-130, 130, -65, 155})
		s.Water(0, 0.28)

		s.Scatter(130, Rect{-55, 55, 6.1, 61}, func(p Vec, r *Rand) {
			if math.Abs(p.X) < 2.6 && p.Y < 10 {
				return // keep a clearing on the beach
			}
			nature.Pine(s.Root, p, 3+r.Float64()*4+p.Y*0.06, r.Int63())
		})
		s.Scatter(7, Rect{-14, 14, 4.6, 6.4}, func(p Vec, r *Rand) {
			s.Root.Add(nature.Rock(0.35+r.Float64()*0.5, r.Int63())).At(p.X, p.Y, p.Z).Turn(0, 0, r.Float64()*6)
		})

		person := mannequin.New(s.Root, 1.75)
		person.Root.At(0.4, 5.9, s.GroundAt(0.4, 5.9))
		person.Shoulder[1].Rot.X = -0.15
		person.Elbow[1].Rot.X = -0.5
	})
}

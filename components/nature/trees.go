package nature

import (
	"math"
	"math/rand"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/model"
	"github.com/loyality7/ln/ln/solid"
	"github.com/loyality7/ln/ln/stage"
)

// Pine adds a conifer standing at base: a trunk and tiered, ragged foliage,
// one lathe so it stays light. The seed varies its proportions.
func Pine(parent *stage.Node, base geom.Vector, height float64, seed int64) *stage.Node {
	r := rand.New(rand.NewSource(seed))
	tree := parent.Group("pine").At(base.X, base.Y, base.Z).Turn(0, 0, r.Float64()*6.28)
	tree.Add(solid.Cylinder(0.035*height, 0.28*height))

	tiers := 5 + r.Intn(2)
	spread := (0.2 + 0.05*r.Float64()) * height
	profile := []solid.Pt{{X: 0, Y: height}}
	top, drop := height, 0.26*height
	for i := 0; i < tiers; i++ {
		f := float64(i+1) / float64(tiers)
		rad := spread * (0.3 + 0.7*f) * (0.9 + 0.2*r.Float64())
		bottom := top - drop*(1.1+0.3*r.Float64())
		profile = append(profile, solid.Pt{X: rad, Y: bottom}, solid.Pt{X: rad * 0.52, Y: bottom + 0.03*height})
		top = bottom + 0.09*height
	}
	profile = append(profile, solid.Pt{X: 0, Y: profile[len(profile)-1].Y})
	tree.Add(solid.Lathe(profile, 14)).At(0, 0, 0.12*height)
	return tree
}

// Rock is a lumpy boulder: a sphere pushed in and out by noise.
func Rock(radius float64, seed int64) *model.Mesh {
	off := float64(seed) * 13.7
	return solid.Orient(solid.Surface(40, 24, func(u, v float64) geom.Vector {
		th, ph := 2*math.Pi*u, math.Pi*v
		n := 0.75 + 0.5*Noise(math.Cos(th)*math.Sin(ph)*1.6+off, math.Sin(th)*math.Sin(ph)*1.6+math.Cos(ph)*1.1+off)
		k := radius * n
		return geom.Vector{X: k * math.Sin(ph) * math.Cos(th), Y: k * math.Sin(ph) * math.Sin(th), Z: k * math.Cos(ph) * 0.7}
	}))
}

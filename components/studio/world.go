package studio

import (
	"math"

	"github.com/loyality7/ln/ln/shapes"
	"github.com/loyality7/ln/ln/solid"
)

// Terrain adds land over the rectangle, shaped by height(x, y). The same function
// is remembered so Scatter and GroundAt can put things on it.
func (s *Scene) Terrain(height func(x, y float64) float64, r Rect) *Node {
	s.ground = height
	w, d := r.X1-r.X0, r.Y1-r.Y0
	nx, ny := clamp(int(w/1.3), 40, 220), clamp(int(d/1.3), 40, 220)
	cx, cy := (r.X0+r.X1)/2, (r.Y0+r.Y1)/2
	mesh := solid.Heightfield(w, d, nx, ny, func(x, y float64) float64 { return height(x+cx, y+cy) })
	return s.Root.Surface(mesh).At(cx, cy, 0)
}

// GroundAt is the terrain height at (x, y), or 0 if there is no terrain.
func (s *Scene) GroundAt(x, y float64) float64 {
	if s.ground == nil {
		return 0
	}
	return s.ground(x, y)
}

// Water adds a mirror lake at the given height. ripple is how strongly it
// distorts reflections (0 = still glass, about 0.3 = gentle, 1 = rough).
func (s *Scene) Water(level, ripple float64) *shapes.Water {
	w := shapes.NewWater(level, 400)
	w.Amp = ripple
	w.Height = func(x, y float64) float64 {
		return 0.05*math.Sin(1.3*x+0.7*y) + 0.04*math.Sin(0.8*x-1.1*y+1.7) + 0.02*math.Sin(3.1*x+2.3*y)
	}
	s.Root.Place(w)
	return w
}

// Scatter calls place n times with random points in the rectangle, standing on the
// terrain. place builds whatever belongs there (a tree, a rock) using the scene's
// seeded random generator, so the result is repeatable.
func (s *Scene) Scatter(n int, r Rect, place func(p Vec, rnd *Rand)) {
	for i := 0; i < n; i++ {
		x := r.X0 + s.Rand.Float64()*(r.X1-r.X0)
		y := r.Y0 + s.Rand.Float64()*(r.Y1-r.Y0)
		place(V(x, y, s.GroundAt(x, y)), s.Rand)
	}
}

func clamp(v, lo, hi int) int { return max(lo, min(hi, v)) }

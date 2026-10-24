package shapes

import (
	"math"

	"github.com/loyality7/ln/ln/geom"
)

// Reflector is implemented by shapes that act as mirrors. Renderers follow the
// ray on after the bounce, so the surface shows whatever it reflects.
type Reflector interface{ Reflectivity() float64 }

// Mirror is a horizontal mirror at height Level, optionally rippled. Height is
// the wave elevation at (x, y); only its slope matters, scaled by Amp, so
// reflections wobble without the surface actually moving.
type Mirror struct {
	Level, Half float64 // surface height; the sheet covers [-Half, Half] in x and y
	Height      func(x, y float64) float64
	Amp         float64
	Gloss       float64 // reflectivity, 0..1
}

func NewMirror(level, half float64) *Mirror {
	return &Mirror{Level: level, Half: half, Amp: 1, Gloss: 1}
}

func (w *Mirror) Compile() {}
func (w *Mirror) BoundingBox() geom.Box {
	return geom.Box{Min: geom.Vector{X: -w.Half, Y: -w.Half, Z: w.Level - 0.05}, Max: geom.Vector{X: w.Half, Y: w.Half, Z: w.Level + 0.05}}
}
func (w *Mirror) Contains(geom.Vector, float64) bool { return false }
func (w *Mirror) Paths() geom.Paths                  { return nil }
func (w *Mirror) Reflectivity() float64              { return w.Gloss }

func (w *Mirror) Intersect(r geom.Ray) Hit {
	if math.Abs(r.Direction.Z) < 1e-9 {
		return NoHit
	}
	t := (w.Level - r.Origin.Z) / r.Direction.Z
	if t < 1e-3 {
		return NoHit
	}
	p := r.Position(t)
	if math.Abs(p.X) > w.Half || math.Abs(p.Y) > w.Half {
		return NoHit
	}
	n := geom.Vector{Z: 1}
	if w.Height != nil {
		const e = 0.02
		hx := (w.Height(p.X+e, p.Y) - w.Height(p.X-e, p.Y)) / (2 * e)
		hy := (w.Height(p.X, p.Y+e) - w.Height(p.X, p.Y-e)) / (2 * e)
		n = geom.Vector{X: -w.Amp * hx, Y: -w.Amp * hy, Z: 1}.Normalize()
	}
	return Hit{w, t, n}
}

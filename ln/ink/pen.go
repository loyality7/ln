package ink

import (
	"math"

	"github.com/loyality7/ln/ln/geom"
)

// wobblePath displaces every point by a smooth field of position, so strokes
// stay continuous and shared endpoints still meet, but nothing is ruler-straight.
// Changing seed per frame gives the "boiling line" look of hand-drawn animation.
func wobblePath(p geom.Path, amp, seed, scale float64) {
	for i, q := range p {
		x, y := q.X*scale, q.Y*scale
		dx := math.Sin(x*1.7+seed)*math.Cos(y*1.3-seed*0.7)*0.6 + math.Sin((x+y)*2.3+seed*1.3)*0.4
		dy := math.Cos(x*1.4-seed*0.9)*math.Sin(y*1.9+seed)*0.6 + math.Cos((x-y)*2.1-seed*1.1)*0.4
		p[i].X += dx * amp
		p[i].Y += dy * amp
	}
}

func (dr *Drawing) wobbleAll(amp, seed, scale float64) {
	if amp == 0 {
		return
	}
	for _, layer := range dr.Layers {
		for _, l := range layer.Lines {
			wobblePath(l.Pts, amp, seed, scale)
		}
		for _, f := range layer.Fills {
			for _, loop := range f.Loops {
				wobblePath(loop, amp, seed, scale)
			}
		}
	}
}

// taper is the pen-pressure factor for segment i of an n-point stroke:
// light at both ends, full in the middle.
func taper(i, n int, min float64) float64 {
	if n <= 2 {
		return 1
	}
	u := (float64(i) - 0.5) / float64(n-1)
	return min + (1-min)*math.Sin(math.Pi*u)
}

// Reveal returns the drawing as if the pen had covered the first fraction p
// of the total stroke length: outlines first, then hatching. Filled regions
// appear once the pen is 60% through.
func (dr *Drawing) Reveal(p float64) *Drawing {
	var total float64
	for _, layer := range dr.Layers {
		for _, l := range layer.Lines {
			total += length(l)
		}
	}
	budget := math.Max(0, math.Min(1, p)) * total
	out := &Drawing{W: dr.W, H: dr.H, mul: dr.mul, taperMin: dr.taperMin}
	var lines []Line
	for _, pass := range []bool{false, true} {
		for _, layer := range dr.Layers {
			for _, l := range layer.Lines {
				if l.Hatch != pass || budget <= 0 {
					continue
				}
				if n := length(l); n <= budget {
					lines = append(lines, l)
					budget -= n
					continue
				}
				cut := Line{Width: l.Width, Hatch: l.Hatch, Pts: l.Pts[:1:1]}
				for i := 1; i < len(l.Pts) && budget > 0; i++ {
					d := l.Pts[i].Distance(l.Pts[i-1])
					if d > budget {
						t := budget / d
						a, b := l.Pts[i-1], l.Pts[i]
						cut.Pts = append(cut.Pts, a.Add(b.Sub(a).MulScalar(t)))
					} else {
						cut.Pts = append(cut.Pts, l.Pts[i])
					}
					budget -= d
				}
				if len(cut.Pts) > 1 {
					if l.W != nil { // keep per-vertex widths for the visible part
						cut.W = l.W[: len(cut.Pts)-1 : len(cut.Pts)-1]
						cut.W = append(cut.W, l.W[len(cut.Pts)-1])
					}
					lines = append(lines, cut)
				}
				budget = 0
			}
		}
	}
	layer := Layer{Lines: lines}
	if p >= 0.6 {
		for _, l := range dr.Layers {
			layer.Fills = append(layer.Fills, l.Fills...)
		}
	}
	out.addLayer(layer)
	return out
}

func length(l Line) float64 {
	var n float64
	for i := 1; i < len(l.Pts); i++ {
		n += l.Pts[i].Distance(l.Pts[i-1])
	}
	return n
}

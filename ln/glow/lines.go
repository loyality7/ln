package glow

import (
	"image"
	"image/color"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/ink"
)

// InkLines draws ink lines (y up, as ink drawings have them) as white light with a soft bloom, on black.
// weight scales the thickness of the light (1 is normal).
func InkLines(lines []ink.Line, w, h, weight float64) Frame {
	cv := New(int(w), int(h))
	for _, l := range lines {
		var p geom.Path
		for _, q := range l.Pts {
			p = append(p, geom.Vector{X: q.X, Y: h - q.Y})
		}
		cv.Path(p, 0.35+0.45*weight, 1, 0.97, 0.94)
	}
	return Frame{Canvas: cv, Look: Look{Exposure: 1.15, Bloom: []float64{0.004, 0.014, 0.045}, Strength: 0.8}}
}

// Imager is anything that can draw itself as a picture, such as an *ink.Drawing.
type Imager interface {
	Image(w, h float64) image.Image
}

// Over is a picture with light on top of it: the light adds to what is underneath (screen blend)
// instead of covering it.
type Over struct {
	Base  Imager
	Light Imager
}

// Image draws Base, then adds Light.
func (o Over) Image(w, h float64) image.Image {
	a, b := o.Base.Image(w, h), o.Light.Image(w, h)
	out := image.NewRGBA(a.Bounds())
	scr := func(p, q uint32) uint8 {
		pf, qf := float64(p)/65535, float64(q)/65535
		return uint8(255*(1-(1-pf)*(1-qf)) + 0.5)
	}
	for y := a.Bounds().Min.Y; y < a.Bounds().Max.Y; y++ {
		for x := a.Bounds().Min.X; x < a.Bounds().Max.X; x++ {
			ar, ag, ab, _ := a.At(x, y).RGBA()
			br, bg, bb, _ := b.At(x, y).RGBA()
			out.SetRGBA(x, y, color.RGBA{scr(ar, br), scr(ag, bg), scr(ab, bb), 255})
		}
	}
	return out
}

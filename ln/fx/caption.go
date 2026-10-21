package fx

import (
	"image"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/ink"
	"github.com/loyality7/ln/ln/stage"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// Caption shows text in a blocky pixel font at the bottom left of the picture
// from t to t+dur seconds: a title card for a move or a scene.
func Caption(st *stage.Stage, t, dur float64, text string) {
	st.Overlays = append(st.Overlays, ink.Overlay{Slot: ink.OverBlack, Draw: func(c *ink.Ctx) ink.Layer {
		var l ink.Layer
		if now := st.TL.Now; now < t || now > t+dur {
			return l
		}
		l.Fills = append(l.Fills, ink.Fill{Loops: PixelText(text, c.W*0.04, c.H*0.07, c.H/105)})
		return l
	}})
}

// PixelText builds the glyphs of text as filled squares, one per lit pixel of the
// font, with its baseline at (x, y) in page pixels (y up) and each font pixel
// drawn as a scale-sized square.
func PixelText(text string, x, y, scale float64) []geom.Path {
	var loops []geom.Path
	face := basicfont.Face7x13
	dot := fixed.P(0, 0)
	for _, r := range text {
		dr, mask, maskp, adv, ok := face.Glyph(dot, r)
		if ok {
			for gy := dr.Min.Y; gy < dr.Max.Y; gy++ {
				for gx := dr.Min.X; gx < dr.Max.X; gx++ {
					if _, _, _, a := mask.At(maskp.X+gx-dr.Min.X, maskp.Y+gy-dr.Min.Y).RGBA(); a > 0x8000 {
						px, py := x+float64(gx)*scale, y-float64(gy)*scale
						loops = append(loops, geom.Path{{X: px, Y: py - scale}, {X: px + scale, Y: py - scale}, {X: px + scale, Y: py}, {X: px, Y: py}})
					}
				}
			}
		}
		dot.X += adv
	}
	return loops
}

var _ = image.Rect

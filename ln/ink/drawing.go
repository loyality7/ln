package ink

import (
	"fmt"
	"image"
	"math"
	"strings"

	"github.com/fogleman/gg"
	"github.com/loyality7/ln/ln/geom"
)

// Color is an RGBA colour with channels 0..1. The zero value (A == 0) means "not
// set": the line or fill then falls back to ink-black, or paper-white if White is set.
type Color struct{ R, G, B, A float64 }

// RGB is an opaque colour.
func RGB(r, g, b float64) Color { return Color{r, g, b, 1} }

// RGBA is a colour with opacity a.
func RGBA(r, g, b, a float64) Color { return Color{r, g, b, a} }

// Mix blends two colours; t = 0 gives c, t = 1 gives o. Unset colours count as opaque black.
func (c Color) Mix(o Color, t float64) Color {
	if c.A == 0 {
		c = Color{A: 1}
	}
	if o.A == 0 {
		o = Color{A: 1}
	}
	l := func(a, b float64) float64 { return a + (b-a)*t }
	return Color{l(c.R, o.R), l(c.G, o.G), l(c.B, o.B), l(c.A, o.A)}
}

// Scale multiplies the colour channels by k (brightness), keeping opacity.
func (c Color) Scale(k float64) Color {
	if c.A == 0 {
		c = Color{A: 1}
	}
	cl := func(x float64) float64 { return math.Max(0, math.Min(1, x*k)) }
	return Color{cl(c.R), cl(c.G), cl(c.B), c.A}
}

// resolve is the colour to paint with: c if set, else white or the default ink.
func (c Color) resolve(white bool) Color {
	switch {
	case c.A > 0:
		return c
	case white:
		return Color{1, 1, 1, 1}
	}
	return Color{0.04, 0.04, 0.06, 1}
}

func (c Color) css() string {
	return fmt.Sprintf("rgb(%d,%d,%d)", int(math.Round(c.R*255)), int(math.Round(c.G*255)), int(math.Round(c.B*255)))
}

// Line is a stroked polyline. Pts carry eye distance in Z. If W is set it gives
// the width in px at every vertex (brush pressure); otherwise Width is scaled by distance.
type Line struct {
	Pts   geom.Path
	Width float64
	W     []float64
	Hatch bool  // shading stroke: tapered, drawn before outlines in the legacy renderer
	White bool  // paper-white ink, for line art on a black ground
	Color Color // any colour; overrides White when set (A > 0)
}

// Fill is a filled region. All loops are filled together with the even-odd
// rule, so a loop inside another cuts a hole.
type Fill struct {
	Loops []geom.Path
	White bool  // paint paper-white instead of ink-black (highlights, cut-outs)
	Color Color // any colour; overrides White when set (A > 0)
}

// Layer is drawn in order: fills first, then lines.
type Layer struct {
	Fills []Fill
	Lines []Line
}

type Drawing struct {
	W, H     float64
	Paper    Color // the page; unset is white
	Layers   []Layer
	mul      func(dist float64) float64
	taperMin float64 // thinnest end of a tapered stroke, fraction of full width
}

// NewDrawing makes a drawing from layers you built yourself (a logo, a diagram).
// Lines should carry per-vertex widths in W.
func NewDrawing(w, h float64, layers ...Layer) *Drawing {
	return &Drawing{W: w, H: h, Layers: layers, mul: func(float64) float64 { return 1 }, taperMin: 0.3}
}

func (dr *Drawing) addLayer(l Layer) { dr.Layers = append(dr.Layers, l) }

// Image rasterizes the layers in order. The arguments are ignored (the drawing
// knows its size) so a Drawing can be used as an anim frame.
func (dr *Drawing) Image(_, _ float64) image.Image {
	dc := gg.NewContext(int(dr.W), int(dr.H))
	dc.InvertY()
	paper := dr.Paper.resolve(true)
	dc.SetRGBA(paper.R, paper.G, paper.B, paper.A)
	dc.Clear()
	dc.SetLineCapRound()
	dc.SetLineJoinRound()
	paint := func(c Color) { dc.SetRGBA(c.R, c.G, c.B, c.A) }
	for _, layer := range dr.Layers {
		for _, f := range layer.Fills {
			paint(f.Color.resolve(f.White))
			for _, loop := range f.Loops {
				if len(loop) < 3 {
					continue
				}
				dc.MoveTo(loop[0].X, loop[0].Y)
				for _, p := range loop[1:] {
					dc.LineTo(p.X, p.Y)
				}
				dc.ClosePath()
			}
			dc.SetFillRuleEvenOdd()
			dc.Fill()
		}
		// one Stroke per quantized width and colour keeps rasterization fast
		type bucket struct {
			k   int
			col Color
		}
		buckets := map[bucket][][2]geom.Vector{}
		for _, l := range layer.Lines {
			for i := 1; i < len(l.Pts); i++ {
				a, b := l.Pts[i-1], l.Pts[i]
				var w float64
				if l.W != nil {
					w = (l.W[i-1] + l.W[i]) / 2
				} else {
					w = l.Width * dr.mul((a.Z+b.Z)/2)
					if l.Hatch {
						w *= taper(i, len(l.Pts), dr.taperMin)
					}
				}
				k := int(math.Round(w * 4))
				if k > 0 {
					bk := bucket{k, l.Color.resolve(l.White)}
					buckets[bk] = append(buckets[bk], [2]geom.Vector{a, b})
				}
			}
		}
		for bk, segs := range buckets {
			paint(bk.col)
			dc.SetLineWidth(float64(bk.k) / 4)
			for _, s := range segs {
				dc.MoveTo(s[0].X, s[0].Y)
				dc.LineTo(s[1].X, s[1].Y)
			}
			dc.Stroke()
		}
	}
	return dc.Image()
}

func (dr *Drawing) SVG() string {
	var b strings.Builder
	fmt.Fprintf(&b, "<svg width=\"%g\" height=\"%g\" xmlns=\"http://www.w3.org/2000/svg\" stroke=\"#0a0a0f\" fill=\"none\" stroke-linecap=\"round\" stroke-linejoin=\"round\">\n", dr.W, dr.H)
	fmt.Fprintf(&b, "<rect width=\"100%%\" height=\"100%%\" fill=\"%s\" stroke=\"none\"/><g transform=\"translate(0,%g) scale(1,-1)\">\n", dr.Paper.resolve(true).css(), dr.H)
	for _, layer := range dr.Layers {
		for _, f := range layer.Fills {
			var d strings.Builder
			for _, loop := range f.Loops {
				for i, p := range loop {
					c := "L"
					if i == 0 {
						c = "M"
					}
					fmt.Fprintf(&d, "%s%.1f,%.1f", c, p.X, p.Y)
				}
				d.WriteString("Z")
			}
			col := f.Color.resolve(f.White).css()
			if f.Color.resolve(f.White).A == 1 && f.Color.A == 0 && !f.White {
				col = "#0a0a0f"
			}
			fmt.Fprintf(&b, "<path fill=\"%s\" fill-rule=\"evenodd\" stroke=\"none\" d=\"%s\"/>\n", col, d.String())
		}
		for _, l := range layer.Lines {
			var z, w float64
			pts := make([]string, len(l.Pts))
			for i, v := range l.Pts {
				z += v.Z
				pts[i] = fmt.Sprintf("%.1f,%.1f", v.X, v.Y)
			}
			if l.W != nil {
				for _, x := range l.W {
					w += x
				}
				w /= float64(len(l.W))
			} else {
				w = l.Width * dr.mul(z/float64(len(l.Pts)))
			}
			col := ""
			if l.Color.A > 0 {
				col = fmt.Sprintf(" stroke=\"%s\"", l.Color.css())
			} else if l.White {
				col = " stroke=\"white\""
			}
			fmt.Fprintf(&b, "<polyline%s stroke-width=\"%.2f\" points=\"%s\"/>\n", col, w, strings.Join(pts, " "))
		}
	}
	b.WriteString("</g></svg>\n")
	return b.String()
}

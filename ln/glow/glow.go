// Package glow draws with light instead of lines: a canvas that adds up the colour of everything
// shining at each pixel, spreads the bright parts into a soft bloom, and tone-maps the result into an
// ordinary image. Anything that emits (gas, flames, sparks, stars, lasers) can be drawn smooth with it, and
// the result plays in video like any other frame.
package glow

import (
	"image"
	"image/color"
	"math"

	"github.com/loyality7/ln/ln/geom"
)

// Canvas holds linear light, three floats per pixel, row 0 at the top. Values can go far above 1:
// that is what makes bright things bloom.
type Canvas struct {
	W, H int
	Pix  []float64
}

// New is a black canvas.
func New(w, h int) *Canvas { return &Canvas{W: w, H: h, Pix: make([]float64, w*h*3)} }

// Add shines light of colour (r, g, b) onto pixel (x, y). Off-canvas positions are ignored.
func (c *Canvas) Add(x, y int, r, g, b float64) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	i := (y*c.W + x) * 3
	c.Pix[i], c.Pix[i+1], c.Pix[i+2] = c.Pix[i]+r, c.Pix[i+1]+g, c.Pix[i+2]+b
}

// Luma is the brightness at pixel (x, y).
func (c *Canvas) Luma(x, y int) float64 {
	i := (y*c.W + x) * 3
	return 0.2126*c.Pix[i] + 0.7152*c.Pix[i+1] + 0.0722*c.Pix[i+2]
}

// Spot shines a soft round spot (a Gaussian of the given radius in pixels) of colour (r, g, b) at a
// position that need not be a whole pixel: a star, a spark, a lamp.
func (c *Canvas) Spot(x, y, radius, r, g, b float64) {
	reach := int(math.Ceil(radius * 3))
	for j := int(y) - reach; j <= int(y)+reach; j++ {
		for i := int(x) - reach; i <= int(x)+reach; i++ {
			d2 := (float64(i)+0.5-x)*(float64(i)+0.5-x) + (float64(j)+0.5-y)*(float64(j)+0.5-y)
			w := math.Exp(-d2 / (2 * radius * radius))
			c.Add(i, j, r*w, g*w, b*w)
		}
	}
}

// Streak shines a thin soft line from (x0, y0) to (x1, y1), fading toward its tail at (x0, y0): a
// shooting star, a speed line, a spark trail.
func (c *Canvas) Streak(x0, y0, x1, y1, width, r, g, b float64) {
	n := int(math.Max(2, math.Hypot(x1-x0, y1-y0)))
	for k := 0; k < n; k++ {
		t := float64(k) / float64(n-1)
		f := t * t // bright at the head
		c.Spot(x0+(x1-x0)*t, y0+(y1-y0)*t, width, r*f, g*f, b*f)
	}
}

// Path shines a glowing line along a polyline (points in canvas pixels, row 0 at the top): soft spots of the
// given radius every fraction of a pixel, so the line is smooth and its brightness does not depend on how
// many points the path has. Use it to draw line art as light.
func (c *Canvas) Path(p geom.Path, radius, r, g, b float64) {
	const step = 0.7
	for i := 0; i+1 < len(p); i++ {
		a, d := p[i], p[i+1]
		length := math.Hypot(d.X-a.X, d.Y-a.Y)
		n := int(math.Max(1, math.Ceil(length/step)))
		w := math.Max(length, 1e-9) / float64(n) * 0.55 // each spot carries the length it stands for
		for k := 0; k < n; k++ {
			t := (float64(k) + 0.5) / float64(n)
			c.Spot(a.X+(d.X-a.X)*t, a.Y+(d.Y-a.Y)*t, radius, r*w, g*w, b*w)
		}
	}
}

// blur is a separable box blur of radius r, applied twice (a close match for a Gaussian).
func blur(src []float64, w, h, r int) []float64 {
	if r < 1 {
		return src
	}
	out, tmp := append([]float64(nil), src...), make([]float64, len(src))
	for pass := 0; pass < 2; pass++ {
		for y := 0; y < h; y++ {
			for ch := 0; ch < 3; ch++ {
				var sum float64
				for x := -r; x <= r; x++ {
					sum += out[(y*w+min(w-1, max(0, x)))*3+ch]
				}
				for x := 0; x < w; x++ {
					tmp[(y*w+x)*3+ch] = sum / float64(2*r+1)
					sum += out[(y*w+min(w-1, x+r+1))*3+ch] - out[(y*w+max(0, x-r))*3+ch]
				}
			}
		}
		for x := 0; x < w; x++ {
			for ch := 0; ch < 3; ch++ {
				var sum float64
				for y := -r; y <= r; y++ {
					sum += tmp[(min(h-1, max(0, y))*w+x)*3+ch]
				}
				for y := 0; y < h; y++ {
					out[(y*w+x)*3+ch] = sum / float64(2*r+1)
					sum += tmp[(min(h-1, y+r+1)*w+x)*3+ch] - tmp[(max(0, y-r)*w+x)*3+ch]
				}
			}
		}
	}
	return out
}

// Look is how light becomes a picture.
type Look struct {
	Exposure float64   // brightness gain before tone mapping (0 = 1)
	Bloom    []float64 // each entry is a blur radius as a fraction of the picture height; the glows are added back
	Strength float64   // how strongly each bloom is added back (0 = 0.6)
	Vignette float64   // 0..1 darkening toward the corners
}

// Soft is a film-like default: two glows, a wide faint one and a tight bright one.
func Soft() Look {
	return Look{Exposure: 1.3, Bloom: []float64{0.012, 0.045}, Strength: 0.7, Vignette: 0.25}
}

// Image blooms and tone-maps the canvas into an image. Bright light rolls off toward white instead of
// clipping, so a very bright core stays colourful at its edges.
func (c *Canvas) Image(l Look) image.Image {
	exp := l.Exposure
	if exp == 0 {
		exp = 1
	}
	str := l.Strength
	if str == 0 {
		str = 0.6
	}
	lit := append([]float64(nil), c.Pix...)
	for _, f := range l.Bloom {
		glow := blur(c.Pix, c.W, c.H, int(f*float64(c.H)))
		for i := range lit {
			lit[i] += str * glow[i]
		}
	}
	img := image.NewRGBA(image.Rect(0, 0, c.W, c.H))
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			i := (y*c.W + x) * 3
			dx, dy := (float64(x)/float64(c.W)-0.5)*2, (float64(y)/float64(c.H)-0.5)*2
			v := 1 - l.Vignette*math.Min(1, (dx*dx+dy*dy)/2)
			var rgb [3]uint8
			for ch := 0; ch < 3; ch++ {
				t := 1 - math.Exp(-exp*lit[i+ch]*v)
				rgb[ch] = uint8(255*math.Pow(t, 0.92) + 0.5)
			}
			img.SetRGBA(x, y, color.RGBA{rgb[0], rgb[1], rgb[2], 255})
		}
	}
	return img
}

// Frame is a canvas with its look, which plays in anim.FrameAnimation like any other frame.
type Frame struct {
	*Canvas
	Look Look
}

// Image renders the frame; its arguments are ignored (the canvas knows its size).
func (f Frame) Image(_, _ float64) image.Image { return f.Canvas.Image(f.Look) }

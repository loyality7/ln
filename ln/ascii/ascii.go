// Package ascii is a rendering look that draws the 3D scene as text: tone becomes
// a ramp of characters and silhouettes, creases and depth edges become | - / \
// lines. It reads the same buffers as the ink looks, so any scene works unchanged.
package ascii

import (
	"image"
	"math"
	"strings"

	"github.com/fogleman/gg"
	"github.com/loyality7/ln/ln/buffer"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/light"
	"github.com/loyality7/ln/ln/scene"
	"golang.org/x/image/draw"
	"golang.org/x/image/font/basicfont"
)

const (
	cellW, cellH = 7, 13 // pixel size of one character (basicfont.Face7x13)
	probe        = 4     // buffer pixels sampled per character across
)

// Sky is the background tone; env.Environment implements it.
type Sky interface {
	Darkness(dir geom.Vector) float64
}

type Options struct {
	Cols   int     // characters per row
	Aspect float64 // picture width / height; default 16:9
	Ramp   string  // characters from lightest to darkest
	Edges  bool    // draw outlines with | - / \
	Invert bool    // light text on a dark page, lit surfaces dense
	Floor  float64 // tones lighter than this stay blank (default 0.3)
	Sky    Sky
}

func (o *Options) fill() {
	if o.Cols == 0 {
		o.Cols = 140
	}
	if o.Aspect == 0 {
		o.Aspect = 16.0 / 9
	}
	if o.Floor == 0 {
		o.Floor = 0.3
	}
	if o.Ramp == "" {
		o.Ramp = " .:-=+*#%@"
	}
}

// Rows is the number of text rows that keeps the picture's aspect ratio.
func (o Options) Rows() int {
	o.fill()
	return int(math.Round(float64(o.Cols*cellW) / (float64(cellH) * o.Aspect)))
}

// Art is a rendered grid of characters.
type Art struct {
	Cols, Rows int
	Cells      []rune // row-major, row 0 at the top
	Invert     bool
}

func (a *Art) String() string {
	var b strings.Builder
	for r := 0; r < a.Rows; r++ {
		b.WriteString(string(a.Cells[r*a.Cols : (r+1)*a.Cols]))
		b.WriteByte('\n')
	}
	return b.String()
}

// Draw gathers the scene and renders it as text.
func Draw(s *scene.Scene, cam buffer.Camera, rig light.Rig, opt Options) *Art {
	opt.fill()
	w := opt.Cols * probe
	h := int(math.Round(float64(w) / opt.Aspect))
	return Render(buffer.Gather(s, cam, w, h, rig), opt)
}

// Render turns gathered buffers into characters.
func Render(g *buffer.Buffers, opt Options) *Art {
	opt.fill()
	rows := opt.Rows()
	art := &Art{Cols: opt.Cols, Rows: rows, Cells: make([]rune, opt.Cols*rows), Invert: opt.Invert}
	ramp := []rune(opt.Ramp)
	far := 0.0
	for k, ok := range g.OK {
		if ok && g.T[k] > far {
			far = g.T[k]
		}
	}
	far = far*1.2 + 1
	depth := func(k int) float64 {
		if g.OK[k] {
			return g.T[k]
		}
		return far
	}
	for r := 0; r < rows; r++ {
		j0, j1 := (rows-1-r)*g.H/rows, (rows-r)*g.H/rows
		for c := 0; c < opt.Cols; c++ {
			i0, i1 := c*g.W/opt.Cols, (c+1)*g.W/opt.Cols
			var tone, n float64
			hit, jump := 0.0, false
			minDot := 1.0
			var gx, gy float64
			for j := j0; j < j1; j++ {
				for i := i0; i < i1; i++ {
					k := j*g.W + i
					n++
					if g.OK[k] {
						hit++
						tone += g.Dark[k]
						for _, o := range [2]int{k + 1, k + g.W} { // right and upper neighbours
							if o >= len(g.OK) || (o == k+1 && i+1 >= g.W) || !g.OK[o] {
								continue
							}
							minDot = math.Min(minDot, g.N[k].Dot(g.N[o]))
							if math.Abs(g.T[o]-g.T[k]) > 0.06*g.T[k] {
								jump = true // a real depth discontinuity, not a slanted surface
							}
						}
					} else if g.IsVoid(k) {
						tone++
					} else if opt.Sky != nil {
						tone += opt.Sky.Darkness(g.Dir[k])
					}
					if i > 0 && i+1 < g.W && j > 0 && j+1 < g.H {
						gx += depth(k+1) - depth(k-1)
						gy += depth(k+g.W) - depth(k-g.W)
					}
				}
			}
			d := math.Max(0, math.Min(1, (tone/n-opt.Floor)/(1-opt.Floor)))
			if opt.Invert {
				d = 1 - d
			}
			ch := ramp[int(math.Round(d*float64(len(ramp)-1)))]
			edge := hit > 0 && (hit < n || jump || minDot < 0.75)
			if opt.Edges && edge && math.Hypot(gx, gy) > 0 {
				ch = edgeChar(gx, gy)
			}
			art.Cells[r*opt.Cols+c] = ch
		}
	}
	return art
}

// edgeChar picks the line character perpendicular to the depth gradient.
// Buffer y points up, matching the page.
func edgeChar(gx, gy float64) rune {
	a := math.Atan2(gy, gx)*180/math.Pi + 90 // direction along the edge
	a = math.Mod(a+360, 180)
	switch {
	case a < 22.5 || a >= 157.5:
		return '-'
	case a < 67.5:
		return '/'
	case a < 112.5:
		return '|'
	}
	return '\\'
}

// Image draws the text onto a page. width and height rescale the natural size
// (Cols*7 x Rows*13); pass 0 to keep it.
func (a *Art) Image(width, height float64) image.Image {
	natW, natH := a.Cols*cellW, a.Rows*cellH
	dc := gg.NewContext(natW, natH)
	fg, bg := 0.05, 1.0
	if a.Invert {
		fg, bg = 0.92, 0.04
	}
	dc.SetRGB(bg, bg, bg)
	dc.Clear()
	dc.SetFontFace(basicfont.Face7x13)
	dc.SetRGB(fg, fg, fg)
	for r := 0; r < a.Rows; r++ {
		dc.DrawString(string(a.Cells[r*a.Cols:(r+1)*a.Cols]), 0, float64(r*cellH+cellH-3))
	}
	if int(width) == 0 || int(height) == 0 || (int(width) == natW && int(height) == natH) {
		return dc.Image()
	}
	out := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))
	draw.ApproxBiLinear.Scale(out, out.Bounds(), dc.Image(), dc.Image().Bounds(), draw.Src, nil)
	return out
}

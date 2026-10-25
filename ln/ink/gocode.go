package ink

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/loyality7/ln/ln/geom"
)

// GoSource writes drawings as an ordinary, readable Go program: one function per frame, each
// listing its colour patches and lines as plain coordinates you can read, edit and move.
// There is no data blob and no footage: running the program draws every frame again.
//
// Each shape is stored as a handful of control points (simplified to within eps pixels) and
// rounded off with ink.Curve when drawn, which keeps the file small and the shapes smooth.
// Positions are kept to a tenth of a pixel. fps is the speed the film plays at.
func GoSource(drawings []*Drawing, fps float64, title string, eps float64) string {
	return goSource(drawings, nil, 0, fps, title, eps)
}

// GoSourceGlow is GoSource for the glow-colour look: each frame is a drawing of colour patches (bases)
// with white glowing lines (lines, one drawing of lines per frame) added on top as light of the given weight.
func GoSourceGlow(bases, lines []*Drawing, weight, fps float64, title string, eps float64) string {
	return goSource(bases, lines, weight, fps, title, eps)
}

func goSource(drawings, lights []*Drawing, weight, fps float64, title string, eps float64) string {
	var b strings.Builder
	num := func(v float64) string {
		s := strconv.FormatFloat(math.Round(v*10)/10, 'f', 1, 64)
		s = strings.TrimSuffix(s, ".0")
		if s == "-0" {
			return "0"
		}
		return s
	}
	short := func(v float64) string { return strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64) }
	col := func(c Color) string {
		switch {
		case c.A == 0:
			return "ink.Color{}"
		case c.A == 1:
			return "ink.RGB(" + short(c.R) + "," + short(c.G) + "," + short(c.B) + ")"
		}
		return "ink.RGBA(" + short(c.R) + "," + short(c.G) + "," + short(c.B) + "," + short(c.A) + ")"
	}
	coords := func(p geom.Path, keep []int) string {
		parts := make([]string, 0, 2*len(keep))
		for _, i := range keep {
			parts = append(parts, num(p[i].X), num(p[i].Y))
		}
		return strings.Join(parts, ",")
	}

	w, h := 0.0, 0.0
	if len(drawings) > 0 {
		w, h = drawings[0].W, drawings[0].H
	}
	header := goHeader
	if lights != nil {
		header = strings.NewReplacer("func() *ink.Drawing", "func() anim.Frame",
			"\t\"github.com/loyality7/ln/ln/geom\"\n", "\t\"github.com/loyality7/ln/ln/geom\"\n\t\"github.com/loyality7/ln/ln/glow\"\n").Replace(header)
	}
	fmt.Fprintf(&b, header, title, len(drawings), fps, num(w), num(h))
	for i := range drawings {
		fmt.Fprintf(&b, "\tframe%d,\n", i)
	}
	b.WriteString("}\n")

	// emit writes a drawing's paper, colour patches and lines, then the return.
	emit := func(d *Drawing) {
		if d.Paper.A > 0 {
			fmt.Fprintf(&b, "\td.Paper = %s\n", col(d.Paper))
		}
		for _, l := range d.Layers {
			b.WriteString("\td.Layers = append(d.Layers, ink.Layer{\n")
			if len(l.Fills) > 0 {
				b.WriteString("\t\tFills: []ink.Fill{\n")
				for _, f := range l.Fills {
					var outlines []string
					for _, lp := range f.Loops {
						if len(lp) < 3 {
							continue
						}
						outlines = append(outlines, "P("+coords(lp, simplifyIndex(lp, eps))+")")
					}
					if len(outlines) > 0 {
						fmt.Fprintf(&b, "\t\t\tpatch(%s, %t, %s),\n", col(f.Color), f.White, strings.Join(outlines, ", "))
					}
				}
				b.WriteString("\t\t},\n")
			}
			if len(l.Lines) > 0 {
				b.WriteString("\t\tLines: []ink.Line{\n")
				for _, ln := range l.Lines {
					if len(ln.Pts) < 2 {
						continue
					}
					keep := simplifyIndex(ln.Pts, eps)
					c := col(ln.Color)
					if ln.White && ln.Color.A == 0 {
						c = "ink.RGB(1,1,1)"
					}
					widths := make([]float64, len(keep))
					for k, idx := range keep {
						switch {
						case ln.W != nil:
							widths[k] = ln.W[idx]
						default:
							widths[k] = ln.Width
						}
					}
					lo, hi := widths[0], widths[0]
					for _, v := range widths {
						lo, hi = math.Min(lo, v), math.Max(hi, v)
					}
					if hi-lo < 0.06 {
						fmt.Fprintf(&b, "\t\t\tstroke(%s, %s, P(%s)),\n", short(widths[0]), c, coords(ln.Pts, keep))
					} else {
						ws := make([]string, len(widths))
						for k, v := range widths {
							ws[k] = short(v)
						}
						fmt.Fprintf(&b, "\t\t\ttaper([]float64{%s}, %s, P(%s)),\n", strings.Join(ws, ","), c, coords(ln.Pts, keep))
					}
				}
				b.WriteString("\t\t},\n")
			}
			b.WriteString("\t})\n")
		}
		b.WriteString("\treturn d\n}\n")
	}

	for i, d := range drawings {
		if lights == nil {
			fmt.Fprintf(&b, "\nfunc frame%d() *ink.Drawing {\n\td := ink.NewDrawing(width, height)\n", i)
			emit(d)
			continue
		}
		fmt.Fprintf(&b, "\nfunc frame%d() anim.Frame {\n\treturn glow.Over{Base: colour%d(), Light: glow.InkLines(lines%d().Layers[0].Lines, width, height, %s)}\n}\n", i, i, i, short(weight))
		fmt.Fprintf(&b, "\nfunc colour%d() *ink.Drawing {\n\td := ink.NewDrawing(width, height)\n", i)
		emit(d)
		fmt.Fprintf(&b, "\nfunc lines%d() *ink.Drawing {\n\td := ink.NewDrawing(width, height)\n", i)
		emit(lights[i])
	}
	return b.String()
}

// simplifyIndex returns the indices of the points that keep the path within eps pixels of
// itself (Douglas-Peucker), always including both ends. eps <= 0 keeps every point.
func simplifyIndex(p geom.Path, eps float64) []int {
	if eps <= 0 || len(p) < 3 {
		all := make([]int, len(p))
		for i := range all {
			all[i] = i
		}
		return all
	}
	keep := make([]bool, len(p))
	keep[0], keep[len(p)-1] = true, true
	type span struct{ lo, hi int }
	stack := []span{{0, len(p) - 1}}
	for len(stack) > 0 {
		s := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		far, at := 0.0, -1
		for i := s.lo + 1; i < s.hi; i++ {
			if d := p[i].SegmentDistance(p[s.lo], p[s.hi]); d > far {
				far, at = d, i
			}
		}
		if at >= 0 && far > eps {
			keep[at] = true
			stack = append(stack, span{s.lo, at}, span{at, s.hi})
		}
	}
	var out []int
	for i, k := range keep {
		if k {
			out = append(out, i)
		}
	}
	return out
}

const goHeader = `//go:build ignore

// Code generated by ln (ink.GoSource). It draws the whole animation as plain shapes.
//
// %s
//
// %d frames, each a function below: frame0, frame1, ... Every colour patch and every line is
// written out as coordinates, so any of them can be read, moved, recoloured or deleted.
//
//	go run animation.go                       # writes out/animation.mp4 (needs ffmpeg)
//	go run animation.go -out clip.gif         # or .webm
//	go run animation.go -frame 10 -out f.png  # one picture
//
// It needs github.com/loyality7/ln in the module it is run from.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fogleman/gg"
	"github.com/loyality7/ln/ln/anim"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/ink"
)

const fps = %g

func main() {
	out := flag.String("out", "out/animation.mp4", "output: .mp4 .gif .webm, or .png for one frame")
	only := flag.Int("frame", -1, "write just this frame as a PNG")
	speed := flag.Float64("fps", fps, "frames per second to play at")
	flag.Parse()
	os.MkdirAll(filepath.Dir(*out), 0o755)
	if *only >= 0 {
		if *only >= len(frames) {
			fmt.Println("the film has", len(frames), "frames")
			os.Exit(1)
		}
		gg.SavePNG(*out, frames[*only]().Image(0, 0))
		fmt.Println("wrote", *out)
		return
	}
	film := anim.FrameAnimation(func(t float64) anim.Frame {
		return frames[min(len(frames)-1, int(t*float64(len(frames))))]()
	})
	if err := film.WriteVideo(*out, len(frames), *speed, width, height); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("wrote", *out, len(frames), "frames")
}

// P is a path from x,y pairs.
func P(xy ...float64) geom.Path {
	p := make(geom.Path, 0, len(xy)/2)
	for i := 0; i+1 < len(xy); i += 2 {
		p = append(p, geom.Vector{X: xy[i], Y: xy[i+1]})
	}
	return p
}

// patch is a flat colour shape: one or more closed outlines (a hole is a second outline).
func patch(c ink.Color, white bool, outlines ...geom.Path) ink.Fill {
	f := ink.Fill{Color: c, White: white}
	for _, o := range outlines {
		f.Loops = append(f.Loops, ink.Curve(append(append(geom.Path(nil), o...), o[0]), 3))
	}
	return f
}

// stroke is a line of one width, rounded through its control points.
func stroke(width float64, c ink.Color, p geom.Path) ink.Line {
	q := ink.Curve(p, 3)
	w := make([]float64, len(q))
	for i := range w {
		w[i] = width
	}
	return ink.Line{Pts: q, W: w, Color: c}
}

// taper is a line whose width changes along it: one width per control point.
func taper(widths []float64, c ink.Color, p geom.Path) ink.Line {
	q := ink.Curve(p, 3)
	w := make([]float64, len(q))
	for i := range w {
		w[i] = widths[min(len(widths)-1, i/3)]
	}
	return ink.Line{Pts: q, W: w, Color: c}
}

const width, height = %s, %s

var frames = []func() *ink.Drawing{
`

// SimplifyIndex returns the indices of the points that keep p within eps pixels of itself, always including
// both ends. eps <= 0 keeps every point.
func SimplifyIndex(p geom.Path, eps float64) []int { return simplifyIndex(p, eps) }

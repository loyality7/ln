// Command contour redraws any video as line art on a white page: bold ink contours of
// the shapes of light and dark, plus the sharp edges (roads, ridges, outlines).
//
//	go run ./examples/video/contour -in clip.mp4 -out out/art.mp4
//	go run ./examples/video/contour -in clip.mp4 -out out/art.png -frame 30   # one frame to tune with
package main

import (
	"flag"
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"

	"github.com/fogleman/gg"
	"github.com/loyality7/ln/ln/anim"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/ink"
	"github.com/loyality7/ln/ln/vision"
)

var (
	in      = flag.String("in", "", "input video")
	out     = flag.String("out", "out/contour.mp4", "output: .mp4 .gif .webm, or .png for a single frame")
	width   = flag.Int("w", 800, "working width in pixels")
	fps     = flag.Float64("fps", 12, "frames per second to sample and write")
	only    = flag.Int("frame", -1, "write just this frame as a PNG")
	keep    = flag.Float64("keep", 1, "keep only the top part of the frame (0.7 drops a car dashboard)")
	mode    = flag.String("mode", "tones", "tones: contours of brightness | edges: sharp edges | both")
	tones   = flag.Int("tones", 12, "brightness levels to outline")
	soften  = flag.Int("blur", 2, "blur radius before outlining tones: higher gives bigger, simpler shapes")
	edgeCut = flag.Float64("edge", 0.35, "how strong an edge must be to be drawn, 0..1 (lower draws more)")
	calm    = flag.Float64("calm", 0.5, "0..1 how much of the previous frame to keep: higher flickers less")
	minLen  = flag.Float64("minlen", 24, "drop lines shorter than this many pixels")
	minSize = flag.Float64("minsize", 8, "drop shapes smaller than this many pixels across")
	weight  = flag.Float64("weight", 1.8, "line weight in pixels")
	smooth  = flag.Float64("smooth", 0.5, "straighten lines to within this many pixels, then round them: higher is simpler")
	fills   = flag.Bool("fills", false, "lay flat colour patches from the footage under the lines (needs python with opencv-python and numpy)")
	goCode  = flag.String("gocode", "", "also write the finished animation as plain readable Go code to this path: every shape as coordinates, no footage needed")
	codeEps = flag.Float64("codeeps", 1, "with -gocode: keep each shape's points within this many pixels (higher: shorter file, simpler shapes)")
	palette = flag.Int("colours", 14, "with -fills: size of the colour palette (more keeps small colour differences, like dark clothes against a dark seat)")
	merge   = flag.Float64("merge", 26, "with -fills: fold palette colours closer than this together (lower keeps more tones)")
	pop     = flag.Float64("pop", 1.7, "how much to boost the saturation of the colour patches (1 = as filmed)")
	colour  = flag.Bool("colour", false, "colour each line from the footage (default: black ink)")
)

func main() {
	flag.Parse()
	var drawings []*ink.Drawing
	var patches [][]ink.Fill // per frame, when -fills is on
	if *fills {
		err := vision.Read(*in, vision.Options{Width: *width, FPS: *fps, Colours: *palette, Merge: *merge, Progress: os.Stderr}, func(f vision.Frame) error {
			fl := f.Fills(3)
			for k := range fl {
				fl[k].Color = boost(fl[k].Color, *pop)
			}
			patches = append(patches, fl)
			return nil
		})
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	var tone, edge []float64 // blended over time so lines do not flicker
	err := anim.ReadVideo(*in, *width, 0, *fps, func(i int, f image.Image) error {
		if *only >= 0 && i < *only {
			return nil
		}
		b := f.Bounds()
		W, H0 := b.Dx(), b.Dy()
		H := int(float64(H0) * *keep)
		luma := make([]float64, W*H)
		valid := make([]bool, W*H)
		for y := 0; y < H; y++ {
			for x := 0; x < W; x++ {
				r, g, bl, _ := f.At(b.Min.X+x, b.Max.Y-1-(y+H0-H)).RGBA() // row 0 at the bottom, like the buffers
				luma[y*W+x] = (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(bl)) / 65535
				valid[y*W+x] = true
			}
		}
		if tone == nil {
			tone, edge = make([]float64, W*H), make([]float64, W*H)
		}
		big := blur(luma, W, H, *soften)
		small := edges(blur(luma, W, H, 1), W, H)
		for k := range tone {
			tone[k] = *calm*tone[k] + (1-*calm)*big[k]
			edge[k] = *calm*edge[k] + (1-*calm)*small[k]
		}

		var lines []ink.Line
		add := func(field []float64, levels []float64, shaded bool) {
			for li, paths := range ink.IsoLines(field, valid, W, H, levels, 1, 1) {
				for _, p := range paths {
					if pathLen(p) < *minLen || math.Max(p.BoundingBox().Size().X, p.BoundingBox().Size().Y) < *minSize {
						continue
					}
					p = ink.Curve(p.Simplify(*smooth), 4)
					w := make([]float64, len(p))
					for j := range w {
						w[j] = *weight
						if shaded { // darker tones are drawn heavier
							w[j] *= 0.45 + 0.9*levels[li]
						}
					}
					if !*colour {
						lines = append(lines, ink.Line{Pts: p, W: w}) // unset colour: plain black ink
						continue
					}
					// colour runs along the line: each stretch takes the footage colour under it
					const stretch = 12
					for s := 0; s+1 < len(p); s += stretch {
						e := min(len(p), s+stretch+1)
						m := p[(s+e)/2]
						r, g, bl, _ := f.At(b.Min.X+min(W-1, int(m.X)), b.Max.Y-1-(min(H-1, int(m.Y))+H0-H)).RGBA()
						lines = append(lines, ink.Line{Pts: p[s:e], W: w[s:e], Color: vivid(float64(r)/65535, float64(g)/65535, float64(bl)/65535)})
					}
				}
			}
		}
		if *mode == "tones" || *mode == "both" {
			levels := make([]float64, *tones)
			for k := range levels {
				levels[k] = (float64(k) + 1) / float64(*tones+1)
			}
			add(tone, levels, true)
		}
		if *mode == "edges" || *mode == "both" {
			add(edge, []float64{*edgeCut}, false)
		}
		layers := []ink.Layer{{Lines: lines}}
		if i < len(patches) { // the colour goes underneath, the lines stay exactly as they were
			layers = []ink.Layer{{Fills: patches[i]}, {Lines: lines}}
		}
		drawings = append(drawings, ink.NewDrawing(float64(W), float64(H), layers...))
		if *only >= 0 {
			return errDone
		}
		return nil
	})
	if err != nil && err != errDone {
		fmt.Println(err)
		os.Exit(1)
	}
	if len(drawings) == 0 {
		fmt.Println("no frames read")
		os.Exit(1)
	}
	os.MkdirAll("out", 0o755)
	if *goCode != "" {
		src := ink.GoSource(drawings, *fps, "Drawn by examples/video/contour from "+filepath.Base(*in), *codeEps)
		if err := os.WriteFile(*goCode, []byte(src), 0o644); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("wrote", *goCode, len(src)/1024, "KB of Go")
	}
	d0 := drawings[0]
	if *only >= 0 {
		gg.SavePNG(*out, d0.Image(0, 0))
		fmt.Println("wrote", *out)
		return
	}
	film := anim.FrameAnimation(func(t float64) anim.Frame {
		return drawings[min(len(drawings)-1, int(math.Floor(t*float64(len(drawings)))))]
	})
	if err := film.WriteVideo(*out, len(drawings), *fps, d0.W, d0.H); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("wrote", *out, len(drawings), "frames")
}

var errDone = fmt.Errorf("done")

// boost pushes a colour away from its own grey by k, so dull footage colours become rich ones.
func boost(c ink.Color, k float64) ink.Color {
	grey := 0.2126*c.R + 0.7152*c.G + 0.0722*c.B
	cl := func(v float64) float64 { return math.Max(0, math.Min(1, v)) }
	return ink.RGB(cl(grey+(c.R-grey)*k), cl(grey+(c.G-grey)*k), cl(grey+(c.B-grey)*k))
}

// vivid turns a footage colour into an ink colour: more saturated, and dark enough to read
// on white paper, so the lines are colourful without going pale.
func vivid(r, g, b float64) ink.Color {
	hi, lo := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	if hi == 0 {
		return ink.RGB(0.1, 0.1, 0.12)
	}
	sat := math.Min(1, (hi-lo)/hi*2.4+0.2)
	val := math.Max(0.2, math.Min(0.58, hi*0.7))
	// rebuild the colour from the footage hue at the new saturation and value
	var h float64
	d := hi - lo
	switch {
	case d == 0:
		h = 0
	case hi == r:
		h = math.Mod((g-b)/d, 6)
	case hi == g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	if h < 0 {
		h += 6
	}
	c := val * sat
	x := c * (1 - math.Abs(math.Mod(h, 2)-1))
	m := val - c
	var rr, gg2, bb float64
	switch int(h) {
	case 0:
		rr, gg2, bb = c, x, 0
	case 1:
		rr, gg2, bb = x, c, 0
	case 2:
		rr, gg2, bb = 0, c, x
	case 3:
		rr, gg2, bb = 0, x, c
	case 4:
		rr, gg2, bb = x, 0, c
	default:
		rr, gg2, bb = c, 0, x
	}
	return ink.RGB(rr+m, gg2+m, bb+m)
}

func pathLen(p geom.Path) float64 {
	var n float64
	for i := 1; i < len(p); i++ {
		n += p[i].Distance(p[i-1])
	}
	return n
}

// blur is a box blur of radius r (two passes), to wash out texture and noise.
func blur(f []float64, w, h, r int) []float64 {
	if r <= 0 {
		return f
	}
	out, tmp := append([]float64(nil), f...), make([]float64, len(f))
	for pass := 0; pass < 2; pass++ {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				var s float64
				for d := -r; d <= r; d++ {
					s += out[y*w+min(w-1, max(0, x+d))]
				}
				tmp[y*w+x] = s / float64(2*r+1)
			}
		}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				var s float64
				for d := -r; d <= r; d++ {
					s += tmp[min(h-1, max(0, y+d))*w+x]
				}
				out[y*w+x] = s / float64(2*r+1)
			}
		}
	}
	return out
}

// edges is the strength of the brightness gradient at each pixel (Sobel), scaled to 0..1:
// near 0 on flat areas, near 1 on the sharpest edge.
func edges(f []float64, w, h int) []float64 {
	at := func(x, y int) float64 { return f[min(h-1, max(0, y))*w+min(w-1, max(0, x))] }
	out := make([]float64, len(f))
	var top float64
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			gx := at(x+1, y-1) + 2*at(x+1, y) + at(x+1, y+1) - at(x-1, y-1) - 2*at(x-1, y) - at(x-1, y+1)
			gy := at(x-1, y+1) + 2*at(x, y+1) + at(x+1, y+1) - at(x-1, y-1) - 2*at(x, y-1) - at(x+1, y-1)
			out[y*w+x] = math.Hypot(gx, gy)
			top = math.Max(top, out[y*w+x])
		}
	}
	for k := range out { // sqrt lifts faint edges so one threshold still catches them
		out[k] = math.Sqrt(out[k] / top)
	}
	return out
}

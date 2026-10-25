// Command art redraws any video as line art with flat colour under it: ink lines for the shapes of
// light and for the sharp edges, flat colour patches from the footage's palette beneath them. Lines and
// colour are both carried along the footage's motion, so the drawing moves with the picture. Nothing in
// it is specific to a video.
//
//	go run ./examples/video/art -in clip.mp4 -out out/art.mp4
//	go run ./examples/video/art -in clip.mp4 -out out/art.png -frame 30          # one frame to tune with
//	go run ./examples/video/art -in clip.mp4 -out out/art.mp4 -gocode out/art.go # also the animation as plain Go code
//	go run ./examples/video/art -in clip.mp4 -out out/art.mp4 -check             # measure it against the footage
//
// It needs python with opencv-python and numpy (see ln/vision), and ffmpeg for video.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"

	"github.com/fogleman/gg"
	"github.com/loyality7/ln/ln/anim"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/glow"
	"github.com/loyality7/ln/ln/ink"
	"github.com/loyality7/ln/ln/vision"
)

func main() {
	in := flag.String("in", "", "input video")
	out := flag.String("out", "out/art.mp4", "output: .mp4 .gif .webm, or .png for a single frame")
	only := flag.Int("frame", -1, "write just this frame as a PNG")
	weight := flag.Float64("weight", 1, "line weight multiplier")
	pop := flag.Float64("pop", 1.4, "how much to boost the saturation of the colour patches (1 = exactly as filmed)")
	white := flag.Bool("white", false, "white lines on a black page, no colour at all")
	autoLine := flag.Bool("autoline", false, "with colour: draw the lines pale where the footage is dark (default: always black)")
	ramp := flag.String("ramp", "", "recolour patches by brightness through a gradient: fire, neon, sunset, ocean (default: the footage's own colours)")
	noFill := flag.Bool("nofill", true, "black lines on white paper, no colour")
	goCode := flag.String("gocode", "", "also write the finished animation as plain readable Go code to this path")
	codeEps := flag.Float64("codeeps", 1, "with -gocode: keep each shape's points within this many pixels (higher: shorter file)")
	check := flag.Bool("check", false, "after writing the video, measure it against the footage frame by frame")
	o := vision.Options{Progress: os.Stderr}
	flag.StringVar(&o.Style, "style", "contour", "contour: thin lines from the footage's light and edges; pencil: one line per border between colour patches")
	flag.IntVar(&o.Width, "w", 800, "working width in pixels")
	flag.Float64Var(&o.FPS, "fps", 12, "frames per second to sample and write")
	flag.IntVar(&o.Colours, "colours", 22, "size of the colour palette")
	flag.Float64Var(&o.Merge, "merge", 14, "fold palette colours closer than this together: lower keeps more tones")
	flag.Float64Var(&o.Flat, "flat", 18, "how flat the colour patches are: higher is flatter")
	flag.Float64Var(&o.Calm, "calm", 0.85, "0..1 how much of the last frame, moved along the footage, to keep where nothing moves (less on moving things): higher flickers less")
	flag.IntVar(&o.Limit, "limit", 0, "stop after this many frames (0 = all), for quick tests")
	flag.Float64Var(&o.MinArea, "minarea", 220, "patches smaller than this many pixels fold into their neighbour")
	flag.Float64Var(&o.Border, "border", 5, "neighbouring patches meeting along an edge weaker than this join")
	flag.IntVar(&o.Tones, "tones", 8, "shade lines (brightness levels) to draw")
	flag.Float64Var(&o.Blur, "blur", 2, "blur before tracing brightness: higher gives bigger, simpler shapes")
	flag.Float64Var(&o.EdgeCut, "edge", 0.65, "how strong an edge must be to be drawn, 0..1: lower draws more")
	flag.Float64Var(&o.MergeColour, "mergecolour", 34, "a merged patch may hold colours no further apart than this in any channel: higher gives bigger flatter patches")
	flag.Float64Var(&o.DetailArea, "detailarea", 70, "smallest detail patch in pixels")
	flag.Float64Var(&o.DetailMin, "detailmin", 30, "a detail patch must differ this much from the patch under it")
	flag.IntVar(&o.DetailMax, "detailmax", 160, "most detail patches per frame (-1: none)")
	flag.Float64Var(&o.MinGrad, "mingrad", 0.02, "a brightness contour is drawn only where the light changes at least this much per pixel along it: higher drops more contours in flat areas")
	flag.Float64Var(&o.SilContrast, "silcontrast", 24, "two patches must differ this much (0..~100) to get a silhouette line between them")
	flag.Float64Var(&o.StandOut, "standout", 45, "a patch that differs this much from its surroundings is always outlined, even in busy texture")
	flag.Float64Var(&o.Clutter, "clutter", 45, "percentile of edge density above which a place counts as busy and is blurred much harder: lower flattens more of the picture")
	flag.Float64Var(&o.Dense, "dense", 100, "percentile of line density above which short lines are dropped as clutter: lower drops more")
	flag.Float64Var(&o.Busy, "busy", 0.5, "texture (0..1) above which short squiggles are dropped: lower keeps leaves and gravel flatter")
	flag.Float64Var(&o.MinLen, "minlen", 36, "drop lines shorter than this many pixels (less readily on moving things)")
	flag.Float64Var(&o.MinSize, "minsize", 10, "drop shapes smaller than this many pixels across")
	flag.Parse()

	var drawings []anim.Frame
	var inkDrawings []*ink.Drawing // the same frames when they are ink, for the Go-code export
	var fw, fh float64
	err := vision.Read(*in, o, func(f vision.Frame) error {
		if *only >= 0 && f.Index < *only {
			return nil
		}
		var layers []ink.Layer
		if !*noFill && !*white {
			fills := f.Fills(3)
			for k := range fills {
				if *ramp != "" {
					fills[k].Color = rampColour(*ramp, fills[k].Color)
				} else {
					fills[k].Color = boost(fills[k].Color, *pop)
				}
			}
			layers = append(layers, ink.Layer{Fills: fills})
		}
		lineCol := ink.Color{} // unset: black ink
		if !*noFill && *autoLine {
			// lines contrast with the colour under them: pale where the footage is dark, black where it is light
			var sum, area float64
			for _, r := range f.Regions {
				sum += (0.2126*r.Color.R + 0.7152*r.Color.G + 0.0722*r.Color.B) * r.Area
				area += r.Area
			}
			if area > 0 && sum/area < 0.3 {
				lineCol = ink.RGB(1, 0.86, 0.8)
			}
		}
		fw, fh = f.W, f.H
		if *white { // white lines drawn as light: on the footage's own colours (dark patches made black) or on black
			cv := glow.New(int(f.W), int(f.H))
			for _, l := range f.Strokes(1, 4, ink.Color{}) {
				var p geom.Path
				for _, q := range l.Pts {
					p = append(p, geom.Vector{X: q.X, Y: f.H - q.Y})
				}
				cv.Path(p, 0.35+0.45**weight, 1, 0.97, 0.94)
			}
			lines := glow.Frame{Canvas: cv, Look: glow.Look{Exposure: 1.15, Bloom: []float64{0.004, 0.014, 0.045}, Strength: 0.8, Vignette: 0}}
			if !*noFill {
				fills := f.Fills(3)
				for k := range fills {
					fills[k].Color = boost(fills[k].Color, *pop)
				}
				blackenDark(fills)
				drawings = append(drawings, lit{ink.NewDrawing(f.W, f.H, ink.Layer{Fills: fills}), lines})
			} else {
				drawings = append(drawings, lines)
			}
			if *only >= 0 {
				return errDone
			}
			return nil
		}
		layers = append(layers, ink.Layer{Lines: f.Strokes(*weight, 4, lineCol)})
		d := ink.NewDrawing(f.W, f.H, layers...)
		inkDrawings = append(inkDrawings, d)
		drawings = append(drawings, d)
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
	os.MkdirAll(filepath.Dir(*out), 0o755)
	if *goCode != "" && len(inkDrawings) == len(drawings) {
		src := ink.GoSource(inkDrawings, o.FPS, "Drawn by examples/video/art from "+filepath.Base(*in), *codeEps)
		if err := os.WriteFile(*goCode, []byte(src), 0o644); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("wrote", *goCode, len(src)/1024, "KB of Go")
	}
	if *only >= 0 {
		gg.SavePNG(*out, drawings[0].Image(0, 0))
		fmt.Println("wrote", *out)
		return
	}
	film := anim.FrameAnimation(func(t float64) anim.Frame {
		return drawings[min(len(drawings)-1, int(math.Floor(t*float64(len(drawings)))))]
	})
	if err := film.WriteVideo(*out, len(drawings), o.FPS, fw, fh); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("wrote", *out, len(drawings), "frames")
	if *check {
		report := *out + ".report.json"
		text, err := vision.Check(*in, *out, report, *out+".worst.png")
		fmt.Println(text)
		if err != nil {
			fmt.Println(err)
		}
	}
}

var errDone = fmt.Errorf("done")

// lit is a picture with glowing lines over it: the lines are light, so they add to what is underneath
// (screen blend) instead of covering it.
type lit struct {
	base  *ink.Drawing
	lines glow.Frame
}

func (l lit) Image(w, h float64) image.Image {
	a, b := l.base.Image(w, h), l.lines.Image(w, h)
	out := image.NewRGBA(a.Bounds())
	for y := a.Bounds().Min.Y; y < a.Bounds().Max.Y; y++ {
		for x := a.Bounds().Min.X; x < a.Bounds().Max.X; x++ {
			ar, ag, ab, _ := a.At(x, y).RGBA()
			br, bg, bb, _ := b.At(x, y).RGBA()
			scr := func(p, q uint32) uint8 {
				pf, qf := float64(p)/65535, float64(q)/65535
				return uint8(255*(1-(1-pf)*(1-qf)) + 0.5)
			}
			out.SetRGBA(x, y, color.RGBA{scr(ar, br), scr(ag, bg), scr(ab, bb), 255})
		}
	}
	return out
}

// blackenDark turns the dark patches of a frame pure black (the silhouette) and leaves the rest as filmed.
func blackenDark(fills []ink.Fill) {
	var top float64
	for _, f := range fills {
		top = math.Max(top, luma(f.Color))
	}
	for k := range fills {
		if luma(fills[k].Color) < 0.55*top {
			fills[k].Color = ink.RGB(0, 0, 0)
		}
	}
}

// boost pushes a colour away from its own grey by k, so dull footage colours become rich ones.
func boost(c ink.Color, k float64) ink.Color {
	grey := 0.2126*c.R + 0.7152*c.G + 0.0722*c.B
	cl := func(v float64) float64 { return math.Max(0, math.Min(1, v)) }
	return ink.RGB(cl(grey+(c.R-grey)*k), cl(grey+(c.G-grey)*k), cl(grey+(c.B-grey)*k))
}

// gradients run from the darkest patch to the lightest, each a list of colours.
var gradients = map[string][]ink.Color{
	"fire":   {ink.RGB(0.04, 0.0, 0.12), ink.RGB(0.45, 0.02, 0.35), ink.RGB(0.9, 0.18, 0.2), ink.RGB(1, 0.6, 0.1), ink.RGB(1, 0.95, 0.55)},
	"neon":   {ink.RGB(0.02, 0.0, 0.1), ink.RGB(0.2, 0.05, 0.6), ink.RGB(0.85, 0.1, 0.8), ink.RGB(0.1, 0.85, 0.95), ink.RGB(0.95, 1, 0.8)},
	"sunset": {ink.RGB(0.05, 0.03, 0.2), ink.RGB(0.4, 0.1, 0.5), ink.RGB(0.95, 0.35, 0.3), ink.RGB(1, 0.7, 0.25), ink.RGB(1, 0.93, 0.7)},
	"ocean":  {ink.RGB(0.0, 0.03, 0.12), ink.RGB(0.03, 0.2, 0.45), ink.RGB(0.0, 0.55, 0.65), ink.RGB(0.45, 0.9, 0.7), ink.RGB(0.95, 1, 0.9)},
}

// rampColour maps a patch's brightness onto the chosen gradient.
func rampColour(name string, c ink.Color) ink.Color {
	g, ok := gradients[name]
	if !ok {
		fmt.Fprintln(os.Stderr, "unknown ramp "+name+": use fire, neon, sunset or ocean")
		os.Exit(2)
	}
	l := math.Max(0, math.Min(1, 0.2126*c.R+0.7152*c.G+0.0722*c.B))
	f := l * float64(len(g)-1)
	i := min(int(f), len(g)-2)
	return g[i].Mix(g[i+1], f-float64(i))
}

func luma(c ink.Color) float64 { return 0.2126*c.R + 0.7152*c.G + 0.0722*c.B }

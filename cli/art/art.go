package art

import (
	"flag"
	"fmt"
	"github.com/loyality7/ln/cli/setup"
	"github.com/loyality7/ln/cli/ui"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/loyality7/ln/components/videoart"
)

// art is `ln art`: any video redrawn as line art.
func Run(args []string) int {
	var video string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		video, args = args[0], args[1:]
	}
	fs := flag.NewFlagSet("art", flag.ContinueOnError)
	look := fs.String("look", string(videoart.GlowColour), "lines | colour | glow | glow-colour")
	out := fs.String("o", "", "output file: .mp4 .gif .webm, or .png for one frame")
	frame := fs.Int("frame", -1, "draw only this frame, as a picture (the quick way to try a look)")
	limit := fs.Int("limit", 0, "draw only the first N frames")
	width := fs.Int("w", 0, "working width in pixels: smaller is faster")
	fps := fs.Float64("fps", 0, "frames per second to draw")
	weight := fs.Float64("weight", 0, "line weight multiplier")
	pop := fs.Float64("pop", 0, "boost the colour patches' saturation, 1 = as filmed")
	style := fs.String("style", "", "contour | pencil: one line per border between colour patches")
	silent := fs.Bool("nosound", false, "do not put the original sound back")
	code := fs.String("code", "", "also save the animation: a .json frames file (play it with `ln play`), or any other name for Go code (lines, colour and glow-colour looks)")
	check := fs.Bool("check", false, "measure the result against the footage when done")
	colours := fs.Int("colours", 0, "size of the colour palette: more keeps small colour differences")
	merge := fs.Float64("merge", 0, "fold palette colours closer than this together: lower keeps more tones")
	mergeColour := fs.Float64("mergecolour", 0, "a merged patch may hold colours no further apart than this: higher gives bigger flatter patches")
	flat := fs.Float64("flat", 0, "how flat the colour patches are: higher is flatter")
	calm := fs.Float64("calm", 0, "0..1 how much of the last frame to keep where nothing moves: higher flickers less")
	minArea := fs.Float64("minarea", 0, "patches smaller than this many pixels fold into their neighbour")
	border := fs.Float64("border", 0, "neighbouring patches meeting along an edge weaker than this join")
	tones := fs.Int("tones", 0, "shade lines (brightness levels) to draw")
	blur := fs.Float64("blur", 0, "blur before tracing brightness: higher gives bigger, simpler shapes")
	edge := fs.Float64("edge", 0, "how strong an edge must be to be drawn, 0..1: lower draws more lines")
	minGrad := fs.Float64("mingrad", 0, "a shade line is drawn only where the light changes at least this much")
	minLen := fs.Float64("minlen", 0, "drop lines shorter than this many pixels")
	minSize := fs.Float64("minsize", 0, "drop shapes smaller than this many pixels across")
	clutter := fs.Float64("clutter", 0, "edge-density percentile above which a place counts as busy and is blurred harder")
	dense := fs.Float64("dense", 0, "line-density percentile above which short lines are dropped as clutter")
	busy := fs.Float64("busy", 0, "texture 0..1 above which short squiggles are dropped")
	silContrast := fs.Float64("silcontrast", 0, "pencil style: two patches must differ this much to get a line between them")
	standOut := fs.Float64("standout", 0, "a patch that differs this much from its surroundings is always outlined")
	detailArea := fs.Float64("detailarea", 0, "smallest detail patch in pixels")
	detailMin := fs.Float64("detailmin", 0, "a detail patch must differ this much from the patch under it")
	detailMax := fs.Int("detailmax", 0, "most detail patches per frame (-1: none)")
	ui.Usage(fs, "ln art <video> [options]",
		"Redraw a video as line art. It finds the shapes and lines of every frame, keeps them steady from\nframe to frame, and draws them in one of four looks. The original sound is put back unless you say not to.",
		[]ui.Group{
			{"Look", []string{"look", "style", "weight", "pop"}},
			{"Output", []string{"o", "frame", "limit", "w", "fps", "nosound", "code", "check"}},
			{"Tuning: lines", []string{"edge", "mingrad", "tones", "blur", "minlen", "minsize", "clutter", "dense", "busy"}},
			{"Tuning: colour patches", []string{"colours", "merge", "mergecolour", "flat", "calm", "minarea", "border", "standout", "silcontrast", "detailarea", "detailmin", "detailmax"}},
		},
		"ln art clip.mp4                        glow-colour look, written next to the clip",
		"ln art clip.mp4 -frame 30              try the look on one frame first",
		"ln art clip.mp4 -look lines -w 480     black lines on white, smaller and faster",
		"ln art clip.mp4 -look glow -edge 0.5   white glowing lines on black, with more lines")
	if proceed, code := ui.ParseFlags(fs, args); !proceed {
		return code
	}
	if video == "" {
		video = fs.Arg(0)
	}
	if video == "" {
		fs.Usage()
		return 2
	}
	if _, err := os.Stat(video); err != nil {
		return ui.Fail("cannot find %s", video)
	}
	o, err := videoart.Default(videoart.Look(*look))
	if err != nil {
		ui.Fail("%v", err)
		ui.Hint("see `ln looks`")
		return 2
	}
	o.Frame, o.Sound, o.GoCode, o.Check = *frame, !*silent, *code, *check
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	v := &o.Vision
	if set["weight"] {
		o.Weight = *weight
	}
	if set["pop"] {
		o.Pop = *pop
	}
	if *style != "" {
		v.Style = *style
	}
	if *width > 0 {
		v.Width = *width
	}
	if *fps > 0 {
		v.FPS = *fps
	}
	v.Limit = *limit
	f64 := func(name string, dst *float64, val float64) {
		if set[name] {
			*dst = val
		}
	}
	num := func(name string, dst *int, val int) {
		if set[name] {
			*dst = val
		}
	}
	num("colours", &v.Colours, *colours)
	f64("merge", &v.Merge, *merge)
	f64("mergecolour", &v.MergeColour, *mergeColour)
	f64("flat", &v.Flat, *flat)
	f64("calm", &v.Calm, *calm)
	f64("minarea", &v.MinArea, *minArea)
	f64("border", &v.Border, *border)
	num("tones", &v.Tones, *tones)
	f64("blur", &v.Blur, *blur)
	f64("edge", &v.EdgeCut, *edge)
	f64("mingrad", &v.MinGrad, *minGrad)
	f64("minlen", &v.MinLen, *minLen)
	f64("minsize", &v.MinSize, *minSize)
	f64("clutter", &v.Clutter, *clutter)
	f64("dense", &v.Dense, *dense)
	f64("busy", &v.Busy, *busy)
	f64("silcontrast", &v.SilContrast, *silContrast)
	f64("standout", &v.StandOut, *standOut)
	f64("detailarea", &v.DetailArea, *detailArea)
	f64("detailmin", &v.DetailMin, *detailMin)
	num("detailmax", &v.DetailMax, *detailMax)
	if *out == "" {
		ext := ".mp4"
		if *frame >= 0 {
			ext = ".png"
		}
		*out = strings.TrimSuffix(filepath.Base(video), filepath.Ext(video)) + "_" + *look + ext
	}
	if problems := setup.Missing(); len(problems) > 0 {
		ui.Fail("something ln needs is missing:")
		for _, p := range problems {
			fmt.Fprintln(os.Stderr, "  - "+p)
		}
		ui.Hint("run `ln setup` to install it, or `ln doctor` to see what is wrong")
		return 1
	}
	prog := ui.NewProgress("reading the video")
	v.Progress = prog
	o.Status = func(m string) {
		prog.Done()
		ui.Step("%s", m)
	}
	started := time.Now()
	if err := videoart.Run(video, *out, o); err != nil {
		prog.Done()
		return ui.Fail("%v", err)
	}
	prog.Done()
	ui.OK("%s  %s", ui.Bold(*out), ui.Dim(fmt.Sprintf("(%s)", time.Since(started).Round(time.Second))))
	return 0
}

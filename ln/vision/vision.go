// Package vision reads the shapes out of a video, so ln can redraw any footage as ink or
// cel animation. It needs no model and knows nothing about what is in the video: it
// flattens each frame into patches of one colour from a single palette for the whole
// clip, finds the lines of its light and its edges, keeps all of it steady from frame to
// frame by carrying it along the footage's motion, and hands back the outlines.
//
// The image work is done by regions.py (OpenCV and NumPy, CPU only), which ships inside
// this package and runs as a child process, the same way ln/anim runs ffmpeg. You need
// python with opencv-python, numpy and scikit-image installed:
//
//	pip install opencv-python numpy scikit-image
//
// Montages work too: ln/vision finds the cuts, gives every shot its own palette, and forgets the last shot
// at each cut, so nothing smears across them.
//
// check.py, next to it, measures a finished drawing against its footage frame by frame:
// flicker, lost structure, and subjects that fade or blink out.
package vision

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/ink"
)

//go:embed regions.py
var script []byte

//go:embed check.py
var checker []byte

// Options tune the reading. The zero value of each field means its default.
type Options struct {
	Style       string  // "contour" (thin lines from the footage's light and edges, the default) or "pencil" (one line per border between colour patches)
	Width       int     // working width in pixels (640)
	FPS         float64 // frames per second to sample (12)
	Colours     int     // size of the palette (14)
	Merge       float64 // fold palette colours closer than this together (26)
	Flat        float64 // colour radius of the flattening: higher is flatter (18)
	Calm        float64 // 0..1 how much of the last frame, moved along the flow, to keep (0.6)
	MinArea     float64 // patches smaller than this many pixels fold into their neighbour (220)
	Border      float64 // neighbouring patches meeting along an edge weaker than this join (5)
	Smooth      float64 // outline simplification of colour patches in pixels (1)
	LineSmooth  float64 // simplification of lines in pixels (0.6)
	Tones       int     // brightness contours to draw (8)
	Blur        float64 // blur radius before tracing brightness: higher gives bigger, simpler shapes (2)
	EdgeCut     float64 // how strong an edge must be to be drawn, 0..1: lower draws more (0.65)
	MergeColour float64 // a merged patch may hold colours no further apart than this (34)
	DetailArea  float64 // smallest detail patch in pixels (70)
	DetailMin   float64 // a detail patch must differ this much (RGB distance) from the patch under it (30)
	DetailMax   int     // most detail patches per frame (160); -1 for none
	MinGrad     float64 // a shade line is drawn only where the light changes at least this much per pixel along it (0.02)
	SilContrast float64 // two patches must differ this much (0..~100) to get a silhouette line between them (24)
	StandOut    float64 // a patch that differs this much from its surroundings is always outlined (45)
	Clutter     float64 // percentile of edge density above which a place counts as busy and is blurred much harder (45)
	Dense       float64 // percentile of line density above which short lines are dropped as clutter: lower drops more (100, off)
	Busy        float64 // texture (0..1) above which short squiggles are dropped: leaves and gravel stay flat (0.5)
	MinLen      float64 // lines shorter than this many pixels are dropped, less readily on moving things (36)
	MinSize     float64 // shapes smaller than this many pixels across are dropped (10)
	Limit       int     // stop after this many frames (0 = all)
	Progress    io.Writer
}

// Region is one patch of a single colour in a single frame.
type Region struct {
	Color ink.Color
	Area  float64
	// ID follows the patch from frame to frame: the same patch has the same ID while it lasts.
	ID int
	// Contrast is how different the colours around the patch are from it: low is a soft
	// edge a painter would leave unlined, high a hard one.
	Contrast float64
	Loops    []geom.Path // the outline first, then any holes; y points up, in pixels
}

// Line is a line found in a frame.
type Line struct {
	Pen   Pen       // which pencil drew it
	Level float64   // for shade lines, the brightness they follow, 0 (dark) to 1 (light)
	Width float64   // suggested width in pixels: silhouettes heavier the more the patches differ
	Pts   geom.Path // y points up, in pixels
}

// Pen is the kind of line.
type Pen int

const (
	Silhouette Pen = iota // the border between two patches, drawn once
	Form                  // an inner edge no silhouette covers
	Shade                 // a thin line of brightness inside a shaded form
)

// Frame is everything found in one frame.
type Frame struct {
	Index   int
	W, H    float64
	Regions []Region // largest first
	Lines   []Line
}

type wireRegion struct {
	C     [3]float64     `json:"c"`
	Area  float64        `json:"area"`
	X     float64        `json:"x"`
	ID    int            `json:"id"`
	Loops [][][2]float64 `json:"loops"`
}

type wireLine struct {
	K int          `json:"k"`
	L float64      `json:"l"`
	W float64      `json:"w"`
	P [][2]float64 `json:"p"`
}

type wireFrame struct {
	I       int          `json:"i"`
	W       float64      `json:"w"`
	H       float64      `json:"h"`
	Regions []wireRegion `json:"regions"`
	Lines   []wireLine   `json:"lines"`
}

// Python finds a python interpreter: the one named in LN_PYTHON if set, else the first of python, python3 and py
// on PATH.
func Python() (string, error) {
	if p := os.Getenv("LN_PYTHON"); p != "" {
		return p, nil
	}
	for _, name := range []string{"python", "python3", "py"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("python not found on PATH (needed by ln/vision)")
}

// Check runs check.py on a source video and the drawing made from it, and returns what it
// printed (a summary of flicker, lost structure and blinking subjects). report is where the
// per-frame numbers are written as JSON, and sheet, if not empty, gets a picture of the
// worst frames.
func Check(source, drawing, report, sheet string) (string, error) {
	py, err := Python()
	if err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp("", "lnvision")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	file := filepath.Join(dir, "check.py")
	if err := os.WriteFile(file, checker, 0o644); err != nil {
		return "", err
	}
	args := []string{file, source, drawing, "--report", report}
	if sheet != "" {
		args = append(args, "--sheet", sheet)
	}
	out, err := exec.Command(py, args...).CombinedOutput()
	return string(out), err
}

// Read runs the vision step on a video and calls each with every frame, in order.
// Return an error from each to stop early.
func Read(video string, o Options, each func(Frame) error) error {
	py, err := Python()
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "lnvision")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	file := filepath.Join(dir, "regions.py")
	if err := os.WriteFile(file, script, 0o644); err != nil {
		return err
	}
	args := []string{file, video}
	if o.Style != "" {
		args = append(args, "--style", o.Style)
	}
	add := func(flag string, v float64, set bool) {
		if set {
			args = append(args, flag, strconv.FormatFloat(v, 'f', -1, 64))
		}
	}
	add("--width", float64(o.Width), o.Width > 0)
	add("--fps", o.FPS, o.FPS > 0)
	add("--colours", float64(o.Colours), o.Colours > 0)
	add("--merge", o.Merge, o.Merge > 0)
	add("--flat", o.Flat, o.Flat > 0)
	add("--calm", o.Calm, o.Calm > 0)
	add("--min-area", o.MinArea, o.MinArea > 0)
	add("--border", o.Border, o.Border > 0)
	add("--smooth", o.Smooth, o.Smooth > 0)
	add("--line-smooth", o.LineSmooth, o.LineSmooth > 0)
	add("--tones", float64(o.Tones), o.Tones > 0)
	add("--blur", o.Blur, o.Blur > 0)
	add("--edge-cut", o.EdgeCut, o.EdgeCut > 0)
	add("--merge-colour", o.MergeColour, o.MergeColour > 0)
	add("--detail-area", o.DetailArea, o.DetailArea > 0)
	add("--detail-min", o.DetailMin, o.DetailMin > 0)
	if o.DetailMax != 0 {
		args = append(args, "--detail-max", strconv.Itoa(max(o.DetailMax, 0)))
	}
	add("--min-grad", o.MinGrad, o.MinGrad > 0)
	add("--sil-contrast", o.SilContrast, o.SilContrast > 0)
	add("--stand-out", o.StandOut, o.StandOut > 0)
	add("--clutter", o.Clutter, o.Clutter > 0)
	add("--dense", o.Dense, o.Dense > 0)
	add("--busy", o.Busy, o.Busy > 0)
	add("--min-len", o.MinLen, o.MinLen > 0)
	add("--min-size", o.MinSize, o.MinSize > 0)
	add("--limit", float64(o.Limit), o.Limit > 0)
	cmd := exec.Command(py, args...)
	cmd.Stderr = o.Progress
	var stderr strings.Builder
	if o.Progress == nil {
		cmd.Stderr = &stderr
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	sc := bufio.NewScanner(out)
	sc.Buffer(make([]byte, 1<<20), 256<<20)
	var failed error
	for sc.Scan() {
		var w wireFrame
		if err := json.Unmarshal(sc.Bytes(), &w); err != nil {
			failed = err
			break
		}
		up := func(l [][2]float64) geom.Path {
			p := make(geom.Path, len(l))
			for i, q := range l {
				p[i] = geom.Vector{X: q[0], Y: w.H - q[1]} // image rows run down, ink runs up
			}
			return p
		}
		f := Frame{Index: w.I, W: w.W, H: w.H}
		for _, r := range w.Regions {
			reg := Region{Color: ink.RGB(r.C[0]/255, r.C[1]/255, r.C[2]/255), Area: r.Area, Contrast: r.X, ID: r.ID}
			for _, l := range r.Loops {
				reg.Loops = append(reg.Loops, up(l))
			}
			f.Regions = append(f.Regions, reg)
		}
		for _, l := range w.Lines {
			f.Lines = append(f.Lines, Line{Pen: Pen(l.K), Level: l.L, Width: l.W, Pts: up(l.P)})
		}
		if failed = each(f); failed != nil {
			break
		}
	}
	if failed == nil {
		failed = sc.Err()
	}
	if failed != nil {
		cmd.Process.Kill()
	}
	if err := cmd.Wait(); failed == nil && err != nil {
		failed = fmt.Errorf("vision: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return failed
}

// Fills are the flat colour patches of the frame, largest first, with rounded outlines and
// no lines: lay them under line art to colour it.
func (f Frame) Fills(smooth int) []ink.Fill {
	if smooth == 0 {
		smooth = 3
	}
	var out []ink.Fill
	for _, r := range f.Regions {
		fill := ink.Fill{Color: r.Color}
		for _, l := range r.Loops {
			fill.Loops = append(fill.Loops, ink.Curve(append(append(geom.Path(nil), l...), l[0]), smooth))
		}
		out = append(out, fill)
	}
	return out
}

// Strokes are the frame's lines as ink, each with the width the pencil chose (silhouettes heavier,
// shading thin), times scale, all rounded off.
func (f Frame) Strokes(scale float64, smooth int, c ink.Color) []ink.Line {
	if smooth == 0 {
		smooth = 4
	}
	var out []ink.Line
	for _, l := range f.Lines {
		if len(l.Pts) < 2 {
			continue
		}
		p := ink.Curve(l.Pts, smooth)
		ws := make([]float64, len(p))
		for i := range ws {
			ws[i] = l.Width * scale
		}
		out = append(out, ink.Line{Pts: p, W: ws, Color: c})
	}
	return out
}

// Style says how a Frame is drawn by Drawing.
type Style struct {
	Colour  bool    // fill patches with their own colour (false: manga ink, the darkest patches solid black)
	Line    float64 // outline weight in pixels (2)
	Smooth  int     // curve points per segment when rounding outlines (3)
	Quiet   float64 // leave a patch unoutlined when the contrast around it is below this (26)
	Paper   ink.Color
	Outline ink.Color // unset: black ink
}

// Drawing draws the frame as flat fills with bold outlines. In ink mode (Colour false)
// the darkest patches become solid black and the rest are left white.
func (f Frame) Drawing(s Style) *ink.Drawing {
	if s.Line == 0 {
		s.Line = 2
	}
	if s.Smooth == 0 {
		s.Smooth = 3
	}
	if s.Quiet == 0 {
		s.Quiet = 26
	}
	var fills []ink.Fill
	var lines []ink.Line
	for _, r := range f.Regions {
		fill := ink.Fill{White: true}
		luma := 0.2126*r.Color.R + 0.7152*r.Color.G + 0.0722*r.Color.B
		switch {
		case s.Colour:
			fill.Color = r.Color
		case luma < 0.22:
			fill.White = false // a spot black
		}
		var loops []geom.Path
		for _, l := range r.Loops {
			round := ink.Curve(append(append(geom.Path(nil), l...), l[0]), s.Smooth)
			loops = append(loops, round)
			if r.Contrast < s.Quiet {
				continue // a soft edge: the change of fill says enough
			}
			w := make([]float64, len(round))
			for i := range w { // harder edges get a heavier line
				w[i] = s.Line * math.Max(0.5, math.Min(1.7, r.Contrast/45))
			}
			lines = append(lines, ink.Line{Pts: round, W: w, Color: s.Outline})
		}
		fill.Loops = loops
		fills = append(fills, fill)
	}
	d := ink.NewDrawing(f.W, f.H, ink.Layer{Fills: fills, Lines: lines})
	d.Paper = s.Paper
	return d
}

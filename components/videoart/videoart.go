// Package videoart redraws any video as line art. It is the engine behind `ln art` and examples/video/art:
// ln/vision finds the lines and colour patches of every frame and carries them along the footage's motion,
// and this package draws them in one of four looks.
//
//	lines        thin black lines on white paper
//	colour       flat colour patches from the footage, with black lines on top
//	glow         white lines as light on black, each with a soft bloom
//	glow-colour  the footage's own colours with its dark shapes made black, and white glowing lines on top
package videoart

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fogleman/gg"
	"github.com/loyality7/ln/ln/anim"
	"github.com/loyality7/ln/ln/film"
	"github.com/loyality7/ln/ln/glow"
	"github.com/loyality7/ln/ln/ink"
	"github.com/loyality7/ln/ln/vision"
)

// Look is how the drawing is painted.
type Look string

const (
	Lines      Look = "lines"
	Colour     Look = "colour"
	Glow       Look = "glow"
	GlowColour Look = "glow-colour"
)

// Looks lists every look.
func Looks() []Look { return []Look{Lines, Colour, Glow, GlowColour} }

// Options say what to draw and how. Start from Default.
type Options struct {
	Look    Look
	Weight  float64 // line weight multiplier (1)
	Pop     float64 // saturation boost for the colour patches, 1 = exactly as filmed
	Frame   int     // a number 0 or more: draw only that frame, as a picture. Negative: the whole video
	Sound   bool    // put the original sound back into the finished video
	GoCode  string  // also write the animation as plain readable Go code to this path (lines, colour and glow-colour looks)
	CodeEps float64 // with GoCode: keep each shape's points within this many pixels (higher: shorter file)
	Check   bool    // after writing, measure the video against the footage frame by frame
	Vision  vision.Options
	Status  func(string) // told what is happening: analysing, encoding, adding sound, checking (may be nil)
}

// Default is the options for a look, with the settings that suit it.
func Default(look Look) (Options, error) {
	o := Options{Look: look, Weight: 1, Pop: 1.4, Frame: -1, Sound: true, CodeEps: 1}
	o.Vision = vision.Options{Width: 640, FPS: 12, Colours: 22, Merge: 14, Calm: 0.85, MergeColour: 34, EdgeCut: 0.65, MinGrad: 0.02, Tones: 8}
	switch look {
	case Lines, Colour:
	case Glow, GlowColour:
		o.Weight, o.Pop = 1.2, 1.1
		o.Vision.EdgeCut = 0.55 // fewer, firmer lines: a glow shows every stray one
	default:
		return o, fmt.Errorf("unknown look %q: choose from %v", look, Looks())
	}
	return o, nil
}

var errDone = fmt.Errorf("done")

// Run redraws the video in and writes the result to out (.mp4, .gif or .webm; .png for a single frame).
func Run(in, out string, o Options) error {
	say := func(m string) {
		if o.Status != nil {
			o.Status(m)
		}
	}
	say("analysing the video")
	var frames []anim.Frame
	var inks []*ink.Drawing // the same frames, when they are ink, for the Go-code export
	var bases, lines []*ink.Drawing // glow-colour: the colour patches and the lines of light, for the Go-code export
	var fw, fh float64
	err := vision.Read(in, o.Vision, func(f vision.Frame) error {
		if o.Frame >= 0 && f.Index < o.Frame {
			return nil
		}
		fw, fh = f.W, f.H
		switch o.Look {
		case Glow:
			frames = append(frames, glowLines(f, o.Weight))
		case GlowColour:
			fills := f.Fills(3)
			for k := range fills {
				fills[k].Color = boost(fills[k].Color, o.Pop)
			}
			blackenDark(fills)
			base := ink.NewDrawing(f.W, f.H, ink.Layer{Fills: fills})
			bases = append(bases, base)
			lines = append(lines, ink.NewDrawing(f.W, f.H, ink.Layer{Lines: f.Strokes(1, 4, ink.Color{})}))
			frames = append(frames, glow.Over{Base: base, Light: glowLines(f, o.Weight)})
		default:
			var layers []ink.Layer
			if o.Look == Colour {
				fills := f.Fills(3)
				for k := range fills {
					fills[k].Color = boost(fills[k].Color, o.Pop)
				}
				layers = append(layers, ink.Layer{Fills: fills})
			}
			layers = append(layers, ink.Layer{Lines: f.Strokes(o.Weight, 4, ink.Color{})})
			d := ink.NewDrawing(f.W, f.H, layers...)
			inks = append(inks, d)
			frames = append(frames, d)
		}
		if o.Frame >= 0 {
			return errDone
		}
		return nil
	})
	if err != nil && err != errDone {
		return err
	}
	if len(frames) == 0 {
		return fmt.Errorf("no frames could be read from %s", in)
	}
	if dir := filepath.Dir(out); dir != "." {
		os.MkdirAll(dir, 0o755)
	}
	if o.GoCode != "" {
		if err := writeCode(in, o, inks, bases, lines); err != nil {
			return err
		}
	}
	if o.Frame >= 0 || strings.EqualFold(filepath.Ext(out), ".png") {
		return gg.SavePNG(out, frames[0].Image(0, 0))
	}
	say("encoding the video")
	movie := anim.FrameAnimation(func(t float64) anim.Frame {
		return frames[min(len(frames)-1, int(math.Floor(t*float64(len(frames)))))]
	})
	if err := movie.WriteVideo(out, len(frames), o.Vision.FPS, fw, fh); err != nil {
		return err
	}
	if o.Sound {
		say("adding the sound")
		if err := addSound(in, out); err != nil {
			fmt.Fprintln(os.Stderr, "note: could not add the original sound:", err)
		}
	}
	if o.Check {
		say("checking the result against the footage")
		text, err := vision.Check(in, out, out+".report.json", out+".worst.png")
		fmt.Println(text)
		return err
	}
	return nil
}

// addSound puts the first audio track of src into the finished video, if src has one.
func addSound(src, video string) error {
	switch strings.ToLower(filepath.Ext(video)) {
	case ".mp4", ".webm":
	default:
		return nil // a gif has no sound
	}
	bin, err := anim.FFmpeg()
	if err != nil {
		return err
	}
	tmp := video + ".sound" + filepath.Ext(video)
	codec := "aac"
	if strings.EqualFold(filepath.Ext(video), ".webm") {
		codec = "libopus"
	}
	if out, err := exec.Command(bin, "-y", "-v", "error", "-i", video, "-i", src, "-map", "0:v", "-map", "1:a?", "-c:v", "copy", "-c:a", codec, "-shortest", tmp).CombinedOutput(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return os.Rename(tmp, video)
}

// glowLines draws the frame's lines as white light with a soft bloom, on black.
func glowLines(f vision.Frame, weight float64) glow.Frame {
	return glow.InkLines(f.Strokes(1, 4, ink.Color{}), f.W, f.H, weight)
}

func luma(c ink.Color) float64 { return 0.2126*c.R + 0.7152*c.G + 0.0722*c.B }

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
	grey := luma(c)
	cl := func(v float64) float64 { return math.Max(0, math.Min(1, v)) }
	return ink.RGB(cl(grey+(c.R-grey)*k), cl(grey+(c.G-grey)*k), cl(grey+(c.B-grey)*k))
}

// writeCode saves the animation as a frames file (.json, played by `ln play`) or as Go code (anything else).
func writeCode(in string, o Options, inks, bases, lines []*ink.Drawing) error {
	title := "Drawn by ln from " + filepath.Base(in)
	drawn, light, weight := inks, []*ink.Drawing(nil), 0.0
	if len(drawn) == 0 {
		drawn, light, weight = bases, lines, o.Weight
	}
	if len(drawn) == 0 {
		return fmt.Errorf("the code export needs the lines, colour or glow-colour look, not %s", o.Look)
	}
	if strings.EqualFold(filepath.Ext(o.GoCode), ".json") {
		f := film.New(drawn, light, weight, o.Vision.FPS, o.CodeEps)
		if err := f.Save(o.GoCode); err != nil {
			return err
		}
		fmt.Printf("wrote %s (%d frames)\n", o.GoCode, len(f.Frames))
		return nil
	}
	src := ink.GoSource(drawn, o.Vision.FPS, title, o.CodeEps)
	if light != nil {
		src = ink.GoSourceGlow(drawn, light, weight, o.Vision.FPS, title, o.CodeEps)
	}
	if err := os.WriteFile(o.GoCode, []byte(src), 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s (%d KB of Go)\n", o.GoCode, len(src)/1024)
	return nil
}

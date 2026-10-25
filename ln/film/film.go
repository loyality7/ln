// Package film saves an animation as one plain data file and plays it again, with no footage and no
// Python: every frame is its colour patches and lines written out as coordinates. `ln art -code x.json`
// writes one, and `ln play x.json` turns it back into a video.
//
// A frame is a drawing (patches and lines), and optionally lines of light added on top of it, which is how
// the glow-colour look is made.
package film

import (
	"encoding/json"
	"fmt"
	"math"
	"os"

	"github.com/loyality7/ln/ln/anim"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/glow"
	"github.com/loyality7/ln/ln/ink"
)

// Film is the whole animation.
type Film struct {
	FPS    float64 `json:"fps"`
	W      float64 `json:"width"`
	H      float64 `json:"height"`
	Weight float64 `json:"lightWeight,omitempty"` // thickness of the glowing lines (1 is normal)
	Frames []Frame `json:"frames"`
}

// Frame is one picture: layers of patches and lines, plus optional lines of light on top.
type Frame struct {
	Paper  []float64   `json:"paper,omitempty"`
	Layers []Layer     `json:"layers"`
	Light  [][]float64 `json:"light,omitempty"` // each line: x, y, x, y, ... (the glow look)
}

// Layer is patches first, then lines.
type Layer struct {
	Fills []Fill `json:"fills,omitempty"`
	Lines []Line `json:"lines,omitempty"`
}

// Fill is one colour patch: outlines (a second outline cuts a hole).
type Fill struct {
	C     []float64   `json:"c,omitempty"` // r, g, b, a
	White bool        `json:"white,omitempty"`
	Loops [][]float64 `json:"loops"`
}

// Line is one stroke through control points, with a width at each point.
type Line struct {
	C     []float64 `json:"c,omitempty"`
	White bool      `json:"white,omitempty"`
	W     []float64 `json:"w"`
	Pts   []float64 `json:"pts"`
}

func rnd(v float64) float64 { return math.Round(v*10) / 10 }

func colour(c ink.Color) []float64 {
	if c.A == 0 {
		return nil
	}
	r := func(v float64) float64 { return math.Round(v*100) / 100 }
	return []float64{r(c.R), r(c.G), r(c.B), r(c.A)}
}

func unColour(c []float64) ink.Color {
	if len(c) != 4 {
		return ink.Color{}
	}
	return ink.RGBA(c[0], c[1], c[2], c[3])
}

func flat(p geom.Path, keep []int) []float64 {
	out := make([]float64, 0, 2*len(keep))
	for _, i := range keep {
		out = append(out, rnd(p[i].X), rnd(p[i].Y))
	}
	return out
}

func path(xy []float64) geom.Path {
	p := make(geom.Path, 0, len(xy)/2)
	for i := 0; i+1 < len(xy); i += 2 {
		p = append(p, geom.Vector{X: xy[i], Y: xy[i+1]})
	}
	return p
}

// New writes drawings (and the lines of light for each, if lights is not nil) into a Film. Each shape is kept
// to within eps pixels of itself, which keeps the file small; 1 is a good value.
func New(drawings, lights []*ink.Drawing, weight, fps, eps float64) *Film {
	f := &Film{FPS: fps, Weight: weight}
	if len(drawings) > 0 {
		f.W, f.H = drawings[0].W, drawings[0].H
	}
	for i, d := range drawings {
		fr := Frame{Paper: colour(d.Paper)}
		for _, l := range d.Layers {
			var out Layer
			for _, fl := range l.Fills {
				var loops [][]float64
				for _, lp := range fl.Loops {
					if len(lp) >= 3 {
						loops = append(loops, flat(lp, ink.SimplifyIndex(lp, eps)))
					}
				}
				if len(loops) > 0 {
					out.Fills = append(out.Fills, Fill{C: colour(fl.Color), White: fl.White, Loops: loops})
				}
			}
			for _, ln := range l.Lines {
				if len(ln.Pts) < 2 {
					continue
				}
				keep := ink.SimplifyIndex(ln.Pts, eps)
				w := make([]float64, len(keep))
				for k, idx := range keep {
					w[k] = math.Round(ln.Width*100) / 100
					if ln.W != nil {
						w[k] = math.Round(ln.W[idx]*100) / 100
					}
				}
				out.Lines = append(out.Lines, Line{C: colour(ln.Color), White: ln.White, W: w, Pts: flat(ln.Pts, keep)})
			}
			fr.Layers = append(fr.Layers, out)
		}
		if lights != nil {
			for _, l := range lights[i].Layers {
				for _, ln := range l.Lines {
					if len(ln.Pts) >= 2 {
						fr.Light = append(fr.Light, flat(ln.Pts, ink.SimplifyIndex(ln.Pts, eps)))
					}
				}
			}
		}
		f.Frames = append(f.Frames, fr)
	}
	return f
}

// Save writes the film as JSON.
func (f *Film) Save(file string) error {
	data, err := json.Marshal(f)
	if err != nil {
		return err
	}
	return os.WriteFile(file, data, 0o644)
}

// Load reads a film written by Save.
func Load(file string) (*Film, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var f Film
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("%s is not a film file: %v", file, err)
	}
	if len(f.Frames) == 0 || f.W <= 0 || f.H <= 0 {
		return nil, fmt.Errorf("%s has no frames", file)
	}
	if f.FPS <= 0 {
		f.FPS = 12
	}
	return &f, nil
}

// Frame draws frame i as a picture source (it has an Image method).
func (f *Film) Frame(i int) anim.Frame {
	fr := f.Frames[i]
	d := ink.NewDrawing(f.W, f.H)
	if len(fr.Paper) == 4 {
		d.Paper = unColour(fr.Paper)
	}
	for _, l := range fr.Layers {
		var out ink.Layer
		for _, fl := range l.Fills {
			fill := ink.Fill{Color: unColour(fl.C), White: fl.White}
			for _, lp := range fl.Loops {
				o := path(lp)
				fill.Loops = append(fill.Loops, ink.Curve(append(append(geom.Path(nil), o...), o[0]), 3))
			}
			out.Fills = append(out.Fills, fill)
		}
		for _, ln := range l.Lines {
			q := ink.Curve(path(ln.Pts), 3)
			w := make([]float64, len(q))
			for k := range w {
				w[k] = ln.W[min(len(ln.W)-1, k/3)]
			}
			out.Lines = append(out.Lines, ink.Line{Pts: q, W: w, Color: unColour(ln.C), White: ln.White})
		}
		d.Layers = append(d.Layers, out)
	}
	if len(fr.Light) == 0 && f.Weight == 0 {
		return d
	}
	var lines []ink.Line
	for _, l := range fr.Light {
		lines = append(lines, ink.Line{Pts: ink.Curve(path(l), 3)})
	}
	return glow.Over{Base: d, Light: glow.InkLines(lines, f.W, f.H, f.Weight)}
}

// Write plays the whole film into a video file (.mp4, .gif or .webm).
func (f *Film) Write(out string, fps float64) error {
	if fps <= 0 {
		fps = f.FPS
	}
	n := len(f.Frames)
	return anim.FrameAnimation(func(t float64) anim.Frame {
		return f.Frame(min(n-1, int(t*float64(n))))
	}).WriteVideo(out, n, fps, f.W, f.H)
}

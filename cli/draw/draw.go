// Package draw is `ln draw`: a 2D drawing file to a picture, as pen-and-ink or as glowing light.
package draw

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fogleman/gg"
	"github.com/loyality7/ln/cli/ui"
	"github.com/loyality7/ln/ln/glow"
	"github.com/loyality7/ln/ln/ink"
)

// Run is `ln draw`.
func Run(args []string) int {
	fs := ui.NewFlagSet("draw")
	out := fs.String("o", "", "output picture: .png, or .svg for ink mode (default: <drawing>.png)")
	ui.Usage(fs, "ln draw <drawing.json> [options]",
		"Draw a picture from a JSON file: pen strokes, fills, hatching and halftone dots as ink, or\nspots, streaks and paths of light as a glow. `ln new -type drawing` writes a working example;\n`ln help drawing` lists every item.", nil,
		"ln draw drawing.json", "ln draw drawing.json -o art.svg")
	if proceed, code := ui.ParseFlags(fs, args); !proceed {
		return code
	}
	if fs.NArg() != 1 {
		return ui.Fail("give one drawing file, for example: ln draw drawing.json")
	}
	in := fs.Arg(0)
	data, err := os.ReadFile(in)
	if err != nil {
		return ui.Fail("%v", err)
	}
	var d file
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		return ui.Fail("%s: %v", in, err)
	}
	if *out == "" {
		*out = strings.TrimSuffix(in, filepath.Ext(in)) + ".png"
	}
	if err := d.write(*out); err != nil {
		return ui.Fail("%v", err)
	}
	ui.OK("wrote %s", *out)
	return 0
}

type file struct {
	Mode   string    `json:"mode"` // ink (default) or glow
	Width  float64   `json:"width"`
	Height float64   `json:"height"`
	Paper  []float64 `json:"paper"`
	Look   *lookJSON `json:"look"`
	Items  []item    `json:"items"`
}

type lookJSON struct {
	Exposure float64   `json:"exposure"`
	Bloom    []float64 `json:"bloom"`
	Strength float64   `json:"strength"`
	Vignette float64   `json:"vignette"`
}

func (d *file) write(out string) error {
	if d.Width <= 0 {
		d.Width = 800
	}
	if d.Height <= 0 {
		d.Height = 600
	}
	switch d.Mode {
	case "glow":
		if strings.EqualFold(filepath.Ext(out), ".svg") {
			return fmt.Errorf("glow pictures are pixels: use .png")
		}
		return d.glow(out)
	case "", "ink":
		return d.ink(out)
	}
	return fmt.Errorf("mode %q: use ink or glow", d.Mode)
}

func (d *file) ink(out string) error {
	var l ink.Layer
	for i, it := range d.Items {
		it.flip = d.Height
		if err := it.addInk(&l); err != nil {
			return fmt.Errorf("item %d (%s): %v", i+1, it.Type, err)
		}
	}
	dr := ink.NewDrawing(d.Width, d.Height, l)
	if len(d.Paper) == 3 {
		dr.Paper = ink.RGB(d.Paper[0], d.Paper[1], d.Paper[2])
	}
	if strings.EqualFold(filepath.Ext(out), ".svg") {
		return os.WriteFile(out, []byte(dr.SVG()), 0o644)
	}
	return gg.SavePNG(out, dr.Image(0, 0))
}

func (d *file) glow(out string) error {
	cv := glow.New(int(d.Width), int(d.Height))
	for i, it := range d.Items {
		if err := it.addGlow(cv); err != nil {
			return fmt.Errorf("item %d (%s): %v", i+1, it.Type, err)
		}
	}
	look := glow.Soft()
	if l := d.Look; l != nil {
		look = glow.Look{Exposure: l.Exposure, Bloom: l.Bloom, Strength: l.Strength, Vignette: l.Vignette}
	}
	return gg.SavePNG(out, cv.Image(look))
}

// Starter is a working drawing file.
const Starter = `{
  "mode": "ink",
  "width": 800,
  "height": 600,
  "paper": [0.97, 0.95, 0.9],
  "items": [
    { "type": "halftone", "box": [40, 40, 760, 560], "density": 0.25, "pitch": 12, "angle": 45, "color": [0.8, 0.75, 0.65] },
    { "type": "circle", "at": [400, 300], "r": 150, "width": 3 },
    { "type": "ellipse", "at": [400, 300], "rx": 230, "ry": 70, "width": 2 },
    { "type": "bezier", "pts": [[100, 500], [250, 350], [550, 650], [700, 500]], "width": 3 },
    { "type": "brush", "pts": [[120, 120], [250, 80], [400, 140], [560, 90], [680, 130]], "style": "broad", "width": 8 },
    { "type": "hatch", "box": [300, 220, 500, 380], "spacing": 8, "angle": 45, "width": 1 }
  ]
}
`

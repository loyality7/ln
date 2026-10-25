package draw

import (
	"fmt"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/glow"
	"github.com/loyality7/ln/ln/ink"
)

// item is one thing on the page. Type picks which fields matter; see `ln help drawing`.
type item struct {
	Type    string      `json:"type"`
	Pts     [][]float64 `json:"pts"`
	At      []float64   `json:"at"`
	R       float64     `json:"r"`
	RX      float64     `json:"rx"`
	RY      float64     `json:"ry"`
	Box     []float64   `json:"box"` // x0, y0, x1, y1
	Color   []float64   `json:"color"`
	White   bool        `json:"white"`
	Width   float64     `json:"width"`
	Style   string      `json:"style"` // brush: broad, dynamic, fine
	Steps   int         `json:"steps"`
	Spacing float64     `json:"spacing"`
	Angle   float64     `json:"angle"`
	Cell    float64     `json:"cell"`
	Lines   int         `json:"lines"`
	Density float64     `json:"density"`
	Pitch   float64     `json:"pitch"`
	Radius  float64     `json:"radius"`
	Power   float64     `json:"power"` // glow: brightness multiplier

	flip float64 // ink pictures have y pointing up; this is the page height used to turn y around (0 = leave)
}

func (it item) y(v float64) float64 {
	if it.flip > 0 {
		return it.flip - v
	}
	return v
}

func (it item) colour() ink.Color {
	if len(it.Color) == 3 {
		return ink.RGB(it.Color[0], it.Color[1], it.Color[2])
	}
	return ink.Color{}
}

func (it item) rgb() (r, g, b float64) {
	r, g, b = 1, 1, 1
	if len(it.Color) == 3 {
		r, g, b = it.Color[0], it.Color[1], it.Color[2]
	}
	p := it.Power
	if p == 0 {
		p = 1
	}
	return r * p, g * p, b * p
}

func (it item) path() (geom.Path, error) {
	steps := it.Steps
	if steps == 0 {
		steps = 64
	}
	switch it.Type {
	case "circle":
		if len(it.At) != 2 || it.R <= 0 {
			return nil, fmt.Errorf("needs at [x, y] and r")
		}
		return geom.Circle(it.At[0], it.y(it.At[1]), it.R, steps), nil
	case "ellipse":
		if len(it.At) != 2 || it.RX <= 0 || it.RY <= 0 {
			return nil, fmt.Errorf("needs at [x, y], rx and ry")
		}
		return geom.Ellipse(it.At[0], it.y(it.At[1]), it.RX, it.RY, steps), nil
	case "bezier":
		if len(it.Pts) != 4 {
			return nil, fmt.Errorf("needs exactly 4 points in pts")
		}
		v := make([]geom.Vector, 4)
		for i, p := range it.Pts {
			if len(p) != 2 {
				return nil, fmt.Errorf("points are [x, y]")
			}
			v[i] = geom.Vector{X: p[0], Y: it.y(p[1])}
		}
		return geom.CubicBezier(v[0], v[1], v[2], v[3], steps), nil
	default: // line, brush, fill, path
		if len(it.Pts) < 2 {
			return nil, fmt.Errorf("needs at least 2 points in pts")
		}
		var p geom.Path
		for _, q := range it.Pts {
			if len(q) != 2 {
				return nil, fmt.Errorf("points are [x, y]")
			}
			p = append(p, geom.Vector{X: q[0], Y: it.y(q[1])})
		}
		return p, nil
	}
}

func (it item) box() (geom.Box, error) {
	if len(it.Box) != 4 {
		return geom.Box{}, fmt.Errorf("needs box [x0, y0, x1, y1]")
	}
	y0, y1 := it.y(it.Box[1]), it.y(it.Box[3])
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	return geom.Box{Min: geom.Vector{X: it.Box[0], Y: y0}, Max: geom.Vector{X: it.Box[2], Y: y1}}, nil
}

func (it item) width(def float64) float64 {
	if it.Width > 0 {
		return it.Width
	}
	return def
}

func (it item) addInk(l *ink.Layer) error {
	tint := func(ls []ink.Line) {
		for i := range ls {
			ls[i].Color, ls[i].White = it.colour(), it.White
		}
	}
	switch it.Type {
	case "line", "circle", "ellipse", "bezier":
		p, err := it.path()
		if err != nil {
			return err
		}
		l.Lines = append(l.Lines, ink.Line{Pts: p, Width: it.width(2), Color: it.colour(), White: it.White})
	case "brush":
		p, err := it.path()
		if err != nil {
			return err
		}
		w := it.width(6)
		style := ink.BroadStyle(w)
		switch it.Style {
		case "", "broad":
		case "dynamic":
			style = ink.DynamicStyle(w)
		case "fine":
			style = ink.FineStyle(w)
		default:
			return fmt.Errorf("style %q: use broad, dynamic or fine", it.Style)
		}
		sm, widths := ink.SmoothStroke(p, style, 12)
		l.Lines = append(l.Lines, ink.Line{Pts: sm, W: widths, Color: it.colour(), White: it.White})
	case "fill":
		p, err := it.path()
		if err != nil {
			return err
		}
		l.Fills = append(l.Fills, ink.Fill{Loops: []geom.Path{p}, Color: it.colour(), White: it.White})
	case "hatch":
		b, err := it.box()
		if err != nil {
			return err
		}
		spacing := it.Spacing
		if spacing == 0 {
			spacing = 8
		}
		ls := ink.HatchBox(b, spacing, it.Angle, it.width(1))
		tint(ls)
		l.Lines = append(l.Lines, ls...)
	case "woven":
		b, err := it.box()
		if err != nil {
			return err
		}
		cell, n := it.Cell, it.Lines
		if cell == 0 {
			cell = 24
		}
		if n == 0 {
			n = 5
		}
		ls := ink.WovenHatch(b, cell, n, it.width(1))
		tint(ls)
		l.Lines = append(l.Lines, ls...)
	case "halftone":
		b, err := it.box()
		if err != nil {
			return err
		}
		pitch := it.Pitch
		if pitch == 0 {
			pitch = 10
		}
		fs := ink.HalftoneGrid(b, it.Density, pitch, it.Angle)
		for i := range fs {
			fs[i].Color, fs[i].White = it.colour(), it.White
		}
		l.Fills = append(l.Fills, fs...)
	default:
		return fmt.Errorf("unknown type: use line, brush, fill, circle, ellipse, bezier, hatch, woven or halftone")
	}
	return nil
}

func (it item) addGlow(c *glow.Canvas) error {
	r, g, b := it.rgb()
	switch it.Type {
	case "spot":
		if len(it.At) != 2 {
			return fmt.Errorf("needs at [x, y] and radius")
		}
		c.Spot(it.At[0], it.At[1], it.Radius, r, g, b)
	case "streak":
		if len(it.Pts) != 2 || len(it.Pts[0]) != 2 || len(it.Pts[1]) != 2 {
			return fmt.Errorf("needs two points in pts")
		}
		c.Streak(it.Pts[0][0], it.Pts[0][1], it.Pts[1][0], it.Pts[1][1], it.width(2), r, g, b)
	case "path", "line", "circle", "ellipse", "bezier":
		p, err := it.path()
		if err != nil {
			return err
		}
		rad := it.Radius
		if rad == 0 {
			rad = it.width(1)
		}
		c.Path(p, rad, r, g, b)
	default:
		return fmt.Errorf("unknown type: use spot, streak, path, line, circle, ellipse or bezier")
	}
	return nil
}

package render

import (
	"fmt"
	"github.com/loyality7/ln/ln/ink"
	"github.com/loyality7/ln/ln/stage"
)

// plotterStep is the plotter look's hidden-line step, from the scene's style.
var plotterStep float64

func setLook(st *stage.Stage, name string, sty *style, sky *[3]float64) error {
	apply := func(s *ink.Style, cur float64) float64 {
		if sty == nil {
			return cur
		}
		if sty.Outline > 0 {
			s.Outline = sty.Outline
		}
		if sty.Hatch > 0 {
			s.Hatch = sty.Hatch
		}
		if sty.Wobble != nil {
			s.Wobble = *sty.Wobble
		}
		if sty.Detail > 0 {
			s.Scale = sty.Detail
		}
		if sty.Boil != nil {
			return *sty.Boil
		}
		return cur
	}
	switch name {
	case "", "ink":
		if l, ok := st.Look.(*stage.InkLook); ok {
			l.BoilFPS = apply(&l.Style, l.BoilFPS)
		}
	case "ascii":
		cols := 130
		if sty != nil && sty.Columns > 0 {
			cols = sty.Columns
		}
		l := stage.NewASCIILook(cols)
		if sty != nil {
			if sty.Ramp != "" {
				l.Options.Ramp = sty.Ramp
			}
			l.Options.Invert = sty.Invert
			l.Options.Floor = sty.Floor
			l.Options.Edges = !sty.NoEdges
		}
		st.Look = l
	case "plotter":
		st.Look = &stage.PlotterLook{Step: plotterStep}
	case "colour", "color":
		l := stage.NewColourLook()
		l.BoilFPS = apply(&l.Style, l.BoilFPS)
		if sky != nil {
			c := ink.RGB(sky[0], sky[1], sky[2])
			l.Style.Sky = [3]ink.Color{c, c, c}
		}
		st.Look = l
	default:
		style, ok := ink.Preset(name)
		if !ok {
			return fmt.Errorf("unknown look %q: ink, manga, sketch, technical, ascii or colour", name)
		}
		if name == "manga" {
			l := &stage.ToonLook{Style: style, BoilFPS: 8}
			l.BoilFPS = apply(&l.Style, l.BoilFPS)
			st.Look = l
		} else {
			l := &stage.InkLook{Style: style, BoilFPS: 12}
			l.BoilFPS = apply(&l.Style, l.BoilFPS)
			st.Look = l
		}
	}
	return nil
}

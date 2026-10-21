package env

import (
	"math"
	"math/rand"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/ink"
)

var starPool = func() []geom.Vector {
	r := rand.New(rand.NewSource(7))
	var s []geom.Vector
	for len(s) < 4000 {
		v := geom.Vector{X: r.NormFloat64(), Y: r.NormFloat64(), Z: r.NormFloat64()}.Normalize()
		s = append(s, v)
	}
	return s
}()

func circle(c geom.Vector, r float64, n int) geom.Path {
	p := make(geom.Path, n+1)
	for i := range p {
		a := 2 * math.Pi * float64(i) / float64(n)
		p[i] = geom.Vector{X: c.X + r*math.Cos(a), Y: c.Y + r*math.Sin(a), Z: c.Z}
	}
	return p
}

// Marks draws stars at night, and the sun and moon discs when they are in view.
func (e *Environment) Marks(c *ink.Ctx) ink.Layer {
	var l ink.Layer
	k := e.sky()
	night := 1 - e.Daylight()
	if night > 0.5 {
		var dots ink.Fill
		dots.White = true
		for i, s := range starPool[:min(k.Stars, len(starPool))] {
			if s.Z < -0.02 {
				continue
			}
			p, ok := c.OnSky(s)
			if !ok {
				continue
			}
			size := 0.9 + 1.4*float64(i%5)/4
			dots.Loops = append(dots.Loops, circle(p, size, 7))
		}
		l.Fills = append(l.Fills, dots)
	}
	disc := func(dir geom.Vector, angular float64) (geom.Vector, float64, bool) {
		if dir.Z < -0.04 {
			return geom.Vector{}, 0, false
		}
		p, ok := c.OnSky(dir)
		return p, c.Focal() * math.Tan(angular), ok
	}
	if p, r, ok := disc(e.SunDir(), k.SunSize); ok {
		l.Fills = append(l.Fills, ink.Fill{Loops: []geom.Path{circle(p, r, 40)}, White: true})
		l.Lines = append(l.Lines, ink.Line{Pts: circle(p, r, 40), W: constW(41, 2)})
		for i := 0; i < 16; i++ { // short rays
			a := 2 * math.Pi * float64(i) / 16
			dx, dy := math.Cos(a), math.Sin(a)
			l.Lines = append(l.Lines, ink.Line{Pts: geom.Path{
				{X: p.X + dx*r*1.35, Y: p.Y + dy*r*1.35}, {X: p.X + dx*r*(1.7+0.5*float64(i%2)), Y: p.Y + dy*r*(1.7+0.5*float64(i%2))}}, W: constW(2, 1.6)})
		}
	}
	if night > 0.3 {
		if p, r, ok := disc(e.MoonDir(), k.MoonSize); ok {
			l.Fills = append(l.Fills, ink.Fill{Loops: []geom.Path{moonLit(p, r, e.MoonPhase)}, White: true})
			l.Lines = append(l.Lines, ink.Line{Pts: circle(p, r, 40), W: constW(41, 1.2)})
		}
	}
	return l
}

func constW(n int, w float64) []float64 {
	s := make([]float64, n)
	for i := range s {
		s[i] = w
	}
	return s
}

// moonLit is the lit part of the moon's disc for the phase (0 new .. 0.5 full .. 1 new).
func moonLit(c geom.Vector, r, phase float64) geom.Path {
	k := math.Cos(2 * math.Pi * phase)
	side := 1.0 // waxing: lit on the right
	if phase > 0.5 {
		side = -1
	}
	var p geom.Path
	const n = 24
	for i := 0; i <= n; i++ { // outer limb
		a := -math.Pi/2 + math.Pi*float64(i)/n
		p = append(p, geom.Vector{X: c.X + side*r*math.Cos(a), Y: c.Y + r*math.Sin(a)})
	}
	for i := n; i >= 0; i-- { // terminator
		a := -math.Pi/2 + math.Pi*float64(i)/n
		p = append(p, geom.Vector{X: c.X + side*r*k*math.Cos(a), Y: c.Y + r*math.Sin(a)})
	}
	return p
}

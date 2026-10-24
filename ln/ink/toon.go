package ink

import (
	"math"

	"github.com/loyality7/ln/ln/buffer"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/scene"
)

// Class is how a surface is inked.
type Class uint8

const (
	Paper Class = iota // white; light and shadow decide hatching and solid shadow
	Black              // always solid black (dark objects)
	White              // never shaded: lamps, glowing windows, highlights
)

// Material controls how one object (identified by its triangle tag) is drawn.
type Material struct {
	Class Class
	Tone  bool    // allow mid-tone hatching (Paper only)
	Line  bool    // draw a contour where this object ends
	LineW float64 // contour weight multiplier

	// BlackAt overrides the style's shadow threshold for this object: shade
	// deeper than this turns solid black. 0 keeps the style default; 2 never blackens.
	BlackAt float64

	// Color is the paint of the object for RenderColour (unset: left paper-white).
	Color Color
}

func DefaultMaterial() Material { return Material{Class: Paper, Tone: true, Line: true, LineW: 1} }

// Slot says where an overlay sits relative to the solid black fills.
type Slot int

const (
	UnderBlack Slot = iota // detail art: hidden wherever solid black is painted over it
	OverBlack              // cut-outs from black areas
	Backdrop               // behind everything: speed lines, panels; the figure gets a paper-white base over it
)

// Overlay is vector art computed from the 3D scene each frame (marks,
// highlights, labels). Draw projects 3D points through the Ctx.
type Overlay struct {
	Slot Slot
	Draw func(c *Ctx) Layer
}

// Ctx lets an overlay map 3D points to the page and ask what is visible.
type Ctx struct {
	W, H float64
	Eye  geom.Vector
	g    *buffer.Buffers
}

// Project maps a world point to page pixels (Z holds the distance from the eye).
func (c *Ctx) Project(p geom.Vector) geom.Vector { return c.g.Project(p, c.W, c.H) }

// Visible reports whether p is on the surface the camera sees (not hidden behind other geometry).
func (c *Ctx) Visible(p geom.Vector) bool { return c.g.Visible(p) }

// RenderToon draws the scene as clean manga ink: solid black regions, smooth
// brush-weight contours around each object, light hatching on mid-tones, plus
// any vector overlays. mats maps object tags to materials.
func RenderToon(s *scene.Scene, eye, center, up geom.Vector, width, height, fovy float64, st Style, mats map[int]Material, overlays []Overlay) *Drawing {
	s.Compile()
	mul := prepare(&st, eye, center)
	dr := &Drawing{W: width, H: height, mul: mul, taperMin: st.TaperMin}
	gw, gh := int(width*st.Scale), int(height*st.Scale)
	g := buffer.Gather(s, buffer.Camera{Eye: eye, Center: center, Up: up, Fovy: fovy}, gw, gh, st.rig())
	sx, sy := width/float64(gw), height/float64(gh)
	ctx := &Ctx{W: width, H: height, Eye: eye, g: g}
	matOf := func(id int) Material {
		if id < 0 { // seen in a mirror: same material as the real thing
			id = -id
		}
		if m, ok := mats[id]; ok {
			return m
		}
		return DefaultMaterial()
	}

	// backdrop art sits under everything; objects get paper-white over it
	hasBackdrop := false
	for _, o := range overlays {
		if o.Slot == Backdrop {
			dr.addLayer(o.Draw(ctx))
			hasBackdrop = true
		}
	}
	if hasBackdrop {
		seen := make([]float64, gw*gh)
		for k := range seen {
			if g.OK[k] {
				seen[k] = 1
			}
		}
		paper := maskFill(seen, gw, gh, sx, sy, st.Smooth)
		paper.White = true
		dr.addLayer(Layer{Fills: []Fill{paper}})
	}

	// which pixels are solid black, and which still get mid-tone hatching
	black := make([]float64, gw*gh)
	hatchDark := make([]float64, gw*gh)
	blackAt := st.BlackAt
	if blackAt <= 0 {
		blackAt = 2
	}
	for k := range black {
		if !g.OK[k] {
			continue
		}
		m := matOf(g.ID[k])
		thr := blackAt
		if m.BlackAt > 0 {
			thr = m.BlackAt
		}
		switch {
		case m.Class == White:
		case m.Class == Black, g.Dark[k] >= thr:
			black[k] = 1
		case m.Tone:
			hatchDark[k] = g.Dark[k]
		}
	}

	var skyBands []Line
	if st.Background != nil {
		skyBands = skyMask(g, st.Background, black, width, height, st)
	}

	// layer: mid-tone hatching
	shade := g.Dark
	g.Dark = hatchDark
	var hatchLayer Layer
	for _, p := range hatch(g, st, width, height) {
		hatchLayer.Lines = append(hatchLayer.Lines, Line{Pts: p, Width: st.Hatch, Hatch: true})
	}
	g.Dark = shade
	hatchLayer.Lines = append(hatchLayer.Lines, skyBands...)
	dr.addLayer(hatchLayer)

	// layer: detail art that solid black paints over
	for _, o := range overlays {
		if o.Slot == UnderBlack {
			dr.addLayer(o.Draw(ctx))
		}
	}

	// layer: solid black regions (shadows, dark objects, night sky) as smooth vector shapes
	fill := maskFill(black, gw, gh, sx, sy, st.Smooth)
	dr.addLayer(Layer{Fills: []Fill{fill}})

	if st.Background != nil {
		dr.addLayer(st.Background.Marks(ctx))
	}
	for _, o := range overlays {
		if o.Slot == OverBlack {
			dr.addLayer(o.Draw(ctx))
		}
	}

	// layer: contours around each object, heavier on the shadow side, plus creases
	var lines Layer
	type box struct{ x0, y0, x1, y1, n int }
	boxes := map[int]*box{}
	for k := range g.OK {
		if !g.OK[k] {
			continue
		}
		i, j := k%gw, k/gw
		b := boxes[g.ID[k]]
		if b == nil {
			b = &box{i, j, i, j, 0}
			boxes[g.ID[k]] = b
		}
		b.x0, b.y0, b.x1, b.y1, b.n = min(b.x0, i), min(b.y0, j), max(b.x1, i), max(b.y1, j), b.n+1
	}
	for id, b := range boxes {
		m := matOf(id)
		if !m.Line || b.n < 30 {
			continue
		}
		pad := 3
		x0, y0 := max(0, b.x0-pad), max(0, b.y0-pad)
		w, h := min(gw, b.x1+pad+1)-x0, min(gh, b.y1+pad+1)-y0
		mask := make([]float64, w*h)
		for j := 0; j < h; j++ {
			for i := 0; i < w; i++ {
				k := (y0+j)*gw + x0 + i
				if g.OK[k] && g.ID[k] == id {
					mask[j*w+i] = 1
				}
			}
		}
		for _, p := range trace(blur(mask, w, h, 1), w, h, 0.5, sx, sy, float64(x0), float64(y0), true) {
			wd := make([]float64, len(p))
			var lastT, lastD float64
			for i := range p {
				if t, d, ok := sampleObject(g, id, p[i].X/sx, p[i].Y/sy); ok {
					lastT, lastD = t, d
				}
				p[i].Z = lastT
				wd[i] = math.Max(0.5, st.Outline*m.LineW*mul(lastT)*(1+st.DarkBoost*lastD))
			}
			p, wd = smoothLoop(p, wd, max(1, st.Smooth-1))
			lines.Lines = append(lines.Lines, Line{Pts: p, W: wd})
		}
	}
	d := eye.Distance(center)
	for _, p := range s.RenderDepth(eye, center, up, width, height, fovy, 0.1, 1000, d/400) {
		wd := make([]float64, len(p))
		for i, q := range p {
			wd[i] = math.Max(0.4, st.Outline*0.45*mul(q.Z))
		}
		lines.Lines = append(lines.Lines, Line{Pts: p, W: wd})
	}
	dr.addLayer(lines)
	dr.wobbleAll(st.Wobble, st.WobbleSeed, st.WobbleScale)
	return dr
}

// sampleObject finds the nearest pixel of object id around a page-space point
// and returns its distance and shadow darkness.
func sampleObject(g *buffer.Buffers, id int, gx, gy float64) (t, dark float64, ok bool) {
	ci, cj := int(gx), int(gy)
	best := math.Inf(1)
	for dj := -2; dj <= 2; dj++ {
		for di := -2; di <= 2; di++ {
			x, y := ci+di, cj+dj
			if x < 0 || y < 0 || x >= g.W || y >= g.H {
				continue
			}
			k := y*g.W + x
			if !g.OK[k] || g.ID[k] != id {
				continue
			}
			if d := float64(di*di + dj*dj); d < best {
				best, t, dark, ok = d, g.T[k], g.Dark[k], true
			}
		}
	}
	return
}

func area2(p geom.Path) float64 {
	var a float64
	for i := range p {
		q := p[(i+1)%len(p)]
		a += p[i].X*q.Y - q.X*p[i].Y
	}
	return a
}

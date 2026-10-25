// Command rising is a 20 second silent film scene drawn only with glowing thin lines on black: a camera
// flies over layered mountains toward a stepped temple at dusk, its lamps light one by one, then the
// camera surges, embers and warp streaks rush past, a shockwave bursts, and a title settles.
//
// Everything is real 3D (terrain, tower, lamps, embers are points in space, projected through a
// perspective camera) but drawn as 2D lines: each mountain slice is a line that hides the ones behind it,
// colour runs from violet in the distance to gold up close, and every line is drawn three times, wide
// and faint, then thin and bright, so it glows.
//
//	go run ./examples/cinema/rising -still 11                 # one frame
//	go run ./examples/cinema/rising -sheet 24                 # a contact sheet of frames across the film
//	go run ./examples/cinema/rising                           # the video, out/cinema/rising.mp4
package main

import (
	"flag"
	"fmt"
	"image"
	"math"
	"math/rand"
	"os"

	"github.com/fogleman/gg"
	"github.com/loyality7/ln/components/nature"
	"github.com/loyality7/ln/ln/anim"
	"github.com/loyality7/ln/ln/fx"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/ink"
	xdraw "golang.org/x/image/draw"
)

const duration = 20.0

var (
	W, H   = 540.0, 960.0
	titleT = "RISE"
)

type vec = geom.Vector

// ---- the camera -------------------------------------------------------------------------------

type camera struct {
	pos   vec
	yaw   float64 // radians, turned toward +x
	pitch float64 // radians, looking down
	focal float64 // pixels
	shake [2]float64
}

// project maps a world point (x right, y forward, z up) to the page. depth is how far in front it is.
func (c camera) project(p vec) (x, y, depth float64, ok bool) {
	r := p.Sub(c.pos)
	cy, sy := math.Cos(c.yaw), math.Sin(c.yaw)
	rx, ry := r.X*cy-r.Y*sy, r.X*sy+r.Y*cy // turn into the camera's own frame
	cp, sp := math.Cos(c.pitch), math.Sin(c.pitch)
	depth = ry*cp - r.Z*sp
	if depth < 0.6 {
		return 0, 0, depth, false
	}
	up := ry*sp + r.Z*cp
	return W/2 + c.focal*rx/depth + c.shake[0], H/2 + c.focal*up/depth + c.shake[1], depth, true
}

func ease(t float64) float64      { t = math.Max(0, math.Min(1, t)); return t * t * (3 - 2*t) }
func seg(t, a, b float64) float64 { return math.Max(0, math.Min(1, (t-a)/(b-a))) }
func mix(a, b, t float64) float64 { return a + (b-a)*t }

// cameraAt is the whole camera move: a slow drift in, a longer approach, a fast surge, a settle.
func cameraAt(t float64) camera {
	var d float64
	switch {
	case t < 5:
		d = mix(130, 104, ease(seg(t, 0, 5)))
	case t < 10:
		d = mix(104, 62, ease(seg(t, 5, 10)))
	case t < 16:
		u := seg(t, 10, 16)
		d = mix(62, 15, u*u*(3-2*u)*0.5+u*u*u*0.5)
	default:
		d = mix(15, 12.5, ease(seg(t, 16, 20)))
	}
	z := mix(8, 3.2, ease(seg(t, 0, 16)))
	arc := 1 - ease(seg(t, 0, 11.5)) // swings in from the side
	c := camera{pos: vec{X: -13 * arc, Y: -d, Z: z + 5*arc}}
	c.yaw = math.Atan2(-c.pos.X, d)
	tz := mix(6, 24, ease(seg(t, 0, 8))) // tilts up from the ground to reveal the tower
	c.pitch = math.Atan2(z-tz, math.Hypot(d, c.pos.X))
	// the lens: eases in, then the vertigo shot (dolly in while zooming out) holds the tower's size
	// while the world behind it stretches, then it settles wide
	switch {
	case t < 11.5:
		c.focal = 720 + 380*ease(seg(t, 9, 11.5))
	case t < 14.8:
		c.focal = mix(1100, 430, ease(seg(t, 11.5, 14.8)))
	default:
		c.focal = mix(430, 520, ease(seg(t, 14.8, 18)))
	}
	if dt := t - 12.2; dt > 0 {
		a := 16 * math.Exp(-5*dt)
		c.shake = [2]float64{a * math.Sin(70*t), a * math.Cos(53*t)}
	}
	return c
}

// ---- colour and glow ---------------------------------------------------------------------------

func lerpC(a, b ink.Color, t float64) ink.Color {
	t = math.Max(0, math.Min(1, t))
	return ink.Color{R: mix(a.R, b.R, t), G: mix(a.G, b.G, t), B: mix(a.B, b.B, t), A: 1}
}

// depthColour runs violet (far) through crimson and amber to bright gold (near), and brightens with height.
func depthColour(d, z float64) ink.Color {
	stops := []ink.Color{{R: 1, G: 0.82, B: 0.4, A: 1}, {R: 0.97, G: 0.5, B: 0.12, A: 1}, {R: 0.72, G: 0.12, B: 0.18, A: 1}, {R: 0.3, G: 0.08, B: 0.4, A: 1}}
	f := math.Max(0, math.Min(1, d/230)) * float64(len(stops)-1)
	i := int(math.Min(f, float64(len(stops)-2)))
	c := lerpC(stops[i], stops[i+1], f-float64(i))
	lift := math.Max(0, math.Min(1, z/40)) * 0.35
	return lerpC(c, ink.Color{R: 1, G: 0.95, B: 0.8, A: 1}, lift)
}

func alpha(c ink.Color, a float64) ink.Color { c.A = a; return c }

// glowLine adds a line three times, wide and faint then thin and bright, so it glows.
func glowLine(l *ink.Layer, pts geom.Path, c ink.Color, w, bright float64) {
	if len(pts) < 2 {
		return
	}
	for _, p := range []struct{ k, a float64 }{{5, 0.025}, {2.4, 0.09}, {1, 0.85}} {
		if p.a*bright < 0.004 { // an opacity of 0 would mean "unset" and draw black ink
			continue
		}
		ws := make([]float64, len(pts))
		for i := range ws {
			ws[i] = w * p.k
		}
		l.Lines = append(l.Lines, ink.Line{Pts: pts, W: ws, Color: alpha(c, math.Min(1, p.a*bright))})
	}
}

func black() ink.Color { return ink.Color{R: 0.012, G: 0.006, B: 0.02, A: 1} }

// ---- the world ---------------------------------------------------------------------------------

func smooth(e0, e1, x float64) float64 { return ease((x - e0) / (e1 - e0)) }

func ground(x, y float64) float64 {
	plateau := smooth(26, 62, math.Hypot(x, y)) // flat around the temple
	side := smooth(16, 75, math.Abs(x))         // valley down the middle, mountains at the sides
	far := 22 * smooth(70, 190, y)              // a wall of peaks behind
	r := nature.Ridged(x*0.011+3, y*0.011, 5)
	f := nature.FBM(x*0.03, y*0.03, 4)
	return plateau * (62*r*side + far*(0.6+r) + 7*f*side)
}

type box struct {
	cx, cy, z0, w, d, h float64
	lit                 float64 // 0 dark .. 1 fully lit; light climbs the tower from the lamps to the top
}

// draw adds the box's visible faces, far to near, each filled black so it hides what is behind it.
func (b box) draw(cam camera, layers *[]ink.Layer) {
	type face struct {
		pts  [4]vec
		dist float64
	}
	x0, x1, y0, y1, z0, z1 := b.cx-b.w/2, b.cx+b.w/2, b.cy-b.d/2, b.cy+b.d/2, b.z0, b.z0+b.h
	fs := []face{
		{pts: [4]vec{{X: x0, Y: y0, Z: z0}, {X: x1, Y: y0, Z: z0}, {X: x1, Y: y0, Z: z1}, {X: x0, Y: y0, Z: z1}}},
		{pts: [4]vec{{X: x0, Y: y1, Z: z0}, {X: x1, Y: y1, Z: z0}, {X: x1, Y: y1, Z: z1}, {X: x0, Y: y1, Z: z1}}},
		{pts: [4]vec{{X: x0, Y: y0, Z: z0}, {X: x0, Y: y1, Z: z0}, {X: x0, Y: y1, Z: z1}, {X: x0, Y: y0, Z: z1}}},
		{pts: [4]vec{{X: x1, Y: y0, Z: z0}, {X: x1, Y: y1, Z: z0}, {X: x1, Y: y1, Z: z1}, {X: x1, Y: y0, Z: z1}}},
		{pts: [4]vec{{X: x0, Y: y0, Z: z1}, {X: x1, Y: y0, Z: z1}, {X: x1, Y: y1, Z: z1}, {X: x0, Y: y1, Z: z1}}},
	}
	normals := []vec{{Y: -1}, {Y: 1}, {X: -1}, {X: 1}, {Z: 1}}
	centre := vec{X: b.cx, Y: b.cy, Z: b.z0 + b.h/2}
	var vis []face
	for i, f := range fs {
		mid := f.pts[0].Add(f.pts[2]).MulScalar(0.5)
		if mid.Sub(cam.pos).Dot(normals[i]) >= 0 { // faces away from the camera
			continue
		}
		f.dist = mid.Distance(cam.pos)
		vis = append(vis, f)
	}
	_ = centre
	for i := range vis { // far to near
		for j := i + 1; j < len(vis); j++ {
			if vis[j].dist > vis[i].dist {
				vis[i], vis[j] = vis[j], vis[i]
			}
		}
	}
	for _, f := range vis {
		var poly geom.Path
		ok := true
		for _, p := range f.pts {
			x, y, _, k := cam.project(p)
			ok = ok && k
			poly = append(poly, vec{X: x, Y: y})
		}
		if !ok {
			continue
		}
		var layer ink.Layer
		layer.Fills = []ink.Fill{{Loops: []geom.Path{poly}, Color: black()}}
		edge := append(append(geom.Path(nil), poly...), poly[0])
		mid := f.pts[0].Add(f.pts[2]).MulScalar(0.5)
		pulse := 4 * b.lit * (1 - b.lit) // a flash of white as the wave of light passes
		glowLine(&layer, edge, lerpC(depthColour(f.dist, mid.Z+b.h), ink.Color{R: 1, G: 0.97, B: 0.85, A: 1}, 0.8*pulse), 1.3, 0.1+0.9*b.lit)
		// grooves across the face give the tier its carved look
		if f.pts[0].Z == f.pts[1].Z && f.pts[0].Y == f.pts[1].Y {
			for k := 1; k <= 2; k++ {
				u := float64(k) / 3
				a, _, _, _ := cam.project(vec{X: x0, Y: f.pts[0].Y, Z: z0 + b.h*u})
				c, d, _, _ := cam.project(vec{X: x1, Y: f.pts[0].Y, Z: z0 + b.h*u})
				_, ya, _, _ := cam.project(vec{X: x0, Y: f.pts[0].Y, Z: z0 + b.h*u})
				glowLine(&layer, geom.Path{{X: a, Y: ya}, {X: c, Y: d}}, alpha(depthColour(f.dist, mid.Z), 0.7), 0.8, 0.7*b.lit)
			}
		}
		*layers = append(*layers, layer)
	}
}

// ---- one frame ---------------------------------------------------------------------------------

var (
	stars  [70][3]float64
	ember  [420][5]float64
	streak [240][3]float64
	lamps  []vec
)

func init() {
	r := rand.New(rand.NewSource(7))
	for i := range stars {
		stars[i] = [3]float64{r.Float64(), 0.45 + 0.55*r.Float64(), r.Float64()}
	}
	for i := range ember {
		ember[i] = [5]float64{-70 + 140*r.Float64(), -120 + 160*r.Float64(), 60 * r.Float64(), 4 + 8*r.Float64(), r.Float64()}
	}
	for i := range streak {
		streak[i] = [3]float64{r.Float64() * 2 * math.Pi, r.Float64(), 0.4 + 0.6*r.Float64()}
	}
	for i := 0; i < 15; i++ {
		x := -22 + 44*float64(i)/14
		lamps = append(lamps, vec{X: x, Y: -10.5, Z: 2.3})
	}
}

// intensity of the surge: embers and streaks.
func surge(t float64) float64 {
	return ease(seg(t, 9.6, 12)) * (1 - ease(seg(t, 16, 18.5)))
}

func frame(t float64) *ink.Drawing {
	cam := cameraAt(t)
	d := ink.NewDrawing(W, H)
	d.Paper = black()
	add := func(l ink.Layer) { d.Layers = append(d.Layers, l) }

	// sky: faint stars, and a great sun low on the horizon made of rings
	var sky ink.Layer
	for _, s := range stars {
		a := 0.25 + 0.5*s[2]
		x, y := s[0]*W, s[1]*H
		sky.Lines = append(sky.Lines, ink.Line{Pts: geom.Path{{X: x, Y: y}, {X: x + 1.2, Y: y}}, W: []float64{1.8, 1.8}, Color: ink.RGBA(1, 0.9, 0.8, a)})
	}
	sx, sy, sd, ok := cam.project(vec{X: 0, Y: 260, Z: 80})
	if ok {
		grow := 1 + 1.3*ease(seg(t, 10, 13))
		R := 85 * cam.focal / sd * grow
		for k := 0; k >= 1; k-- { // (no halo: the sky stays dark)
			rr := R * (1 + 0.16*float64(k))
			var disc geom.Path
			for i := 0; i < 64; i++ {
				a := 2 * math.Pi * float64(i) / 64
				disc = append(disc, vec{X: sx + rr*math.Cos(a), Y: sy + rr*math.Sin(a)})
			}
			sky.Fills = append(sky.Fills, ink.Fill{Loops: []geom.Path{disc}, Color: ink.RGBA(0.8, 0.27, 0.1, 0.02)})
		}
		for k := 1; k <= 6; k++ {
			rr := R * float64(k) / 6
			var ring geom.Path
			for i := 0; i <= 90; i++ {
				a := 2 * math.Pi * float64(i) / 90
				ring = append(ring, vec{X: sx + rr*math.Cos(a), Y: sy + rr*math.Sin(a)})
			}
			c := lerpC(ink.Color{R: 1, G: 0.92, B: 0.6, A: 1}, ink.Color{R: 0.85, G: 0.2, B: 0.15, A: 1}, float64(k)/6)
			glowLine(&sky, ring, c, 1.0, 0.3+0.5*(1-float64(k)/6))
		}
	}
	// long thin clouds drifting across the sky, in crimson and violet
	for k := 0; k < 3; k++ {
		yy := H*0.52 + float64(k)*H*0.1 + 8*math.Sin(float64(k)*3)
		var cl geom.Path
		for i := 0; i <= 80; i++ {
			x := -40 + (W+80)*float64(i)/80
			cl = append(cl, vec{X: x + 12*math.Sin(t*0.25+float64(k)), Y: yy + 7*math.Sin(x*0.02+float64(k)*2+t*0.3) + 3*math.Sin(x*0.07+float64(k))})
		}
		glowLine(&sky, cl, lerpC(ink.Color{R: 0.95, G: 0.4, B: 0.2, A: 1}, ink.Color{R: 0.45, G: 0.15, B: 0.6, A: 1}, float64(k)/2), 0.8, 0.3)
	}
	add(sky)

	// terrain slices far to near; the temple is drawn between the ones behind it and the ones in front
	templeBack := 7.0
	var tiers []box
	tiers = append(tiers, box{0, 0, 0, 44, 30, 2, ease(seg(t, 5.2, 6.4))}, box{0, 0, 2, 38, 25, 1.2, ease(seg(t, 5.6, 7))})
	z := 3.2
	for i := 0; i < 9; i++ {
		tiers = append(tiers, box{0, 0, z, 22 - 2.0*float64(i), 16 - 1.3*float64(i), 3.2 - 0.12*float64(i), ease(seg(t, 6.4+0.55*float64(i), 7.6+0.55*float64(i)))})
		z += 3.2 - 0.12*float64(i)
	}
	drawTemple := func() {
		var layers []ink.Layer
		for _, b := range tiers {
			b.draw(cam, &layers)
		}
		for _, l := range layers {
			add(l)
		}
		// the golden finial
		var top ink.Layer
		x0, y0, _, ok0 := cam.project(vec{Z: z})
		x1, y1, _, ok1 := cam.project(vec{Z: z + 4.2})
		if ok0 && ok1 {
			glowLine(&top, geom.Path{{X: x0, Y: y0}, {X: x1, Y: y1}}, ink.Color{R: 1, G: 0.9, B: 0.5, A: 1}, 2, 0.05+0.95*ease(seg(t, 11.4, 12.2)))
			add(top)
		}
	}
	nearY := cam.pos.Y + 1.5
	step := 3.4
	templeDone := false
	for y := 230.0; y >= nearY; y -= step {
		if !templeDone && y < templeBack {
			drawTemple()
			templeDone = true
		}
		var pts geom.Path
		var line geom.Path
		var avgZ, depthAt float64
		n := 0
		for x := -95.0; x <= 95; x += 1.4 {
			h := ground(x, y)
			px, py, dp, k := cam.project(vec{X: x, Y: y, Z: h})
			if !k {
				continue
			}
			line = append(line, vec{X: px, Y: py})
			avgZ += h
			depthAt += dp
			n++
		}
		if n < 4 {
			continue
		}
		avgZ /= float64(n)
		depthAt /= float64(n)
		pts = append(append(geom.Path(nil), line...), vec{X: line[len(line)-1].X, Y: -20}, vec{X: line[0].X, Y: -20})
		var layer ink.Layer
		layer.Fills = []ink.Fill{{Loops: []geom.Path{pts}, Color: black()}}
		b := 0.85
		if avgZ < 1.2 { // flat ground stays dark: only the mountains are drawn
			continue
		}
		b = 0.55
		glowLine(&layer, line, depthColour(depthAt, avgZ), 1.15, b)
		add(layer)
	}
	if !templeDone {
		drawTemple()
	}

	// lamps on the platform's front edge light one by one
	var lampLayer ink.Layer
	for i, p := range lamps {
		on := ease(seg(t, 5.4+0.32*float64(i), 5.9+0.32*float64(i))) * (1 - ease(seg(t, 14.2, 15.2)))
		if on <= 0 {
			continue
		}
		x, y, dp, ok := cam.project(p)
		if !ok {
			continue
		}
		s := cam.focal / dp
		fl := 1 + 0.12*math.Sin(t*17+float64(i)*2.1) + 0.08*math.Sin(t*29+float64(i))
		for k := 5; k >= 1; k-- {
			rr := s * 1.1 * float64(k) * fl * on
			var ring geom.Path
			for j := 0; j <= 28; j++ {
				a := 2 * math.Pi * float64(j) / 28
				ring = append(ring, vec{X: x + rr*math.Cos(a), Y: y + rr*math.Sin(a)*1.25})
			}
			lampLayer.Fills = append(lampLayer.Fills, ink.Fill{Loops: []geom.Path{ring[:28]}, Color: ink.RGBA(1, 0.55, 0.12, 0.04*on)})
		}
		var flame geom.Path
		for j := 0; j <= 16; j++ {
			u := float64(j) / 16
			wob := 0.18 * math.Sin(t*13+float64(i)+u*5)
			flame = append(flame, vec{X: x + s*(0.5*math.Sin(math.Pi*u)*(1-0.55*u)+wob*u)*on*fl, Y: y + s*2.4*u*on*fl})
		}
		for j := 16; j >= 0; j-- {
			u := float64(j) / 16
			wob := 0.18 * math.Sin(t*13+float64(i)+u*5)
			flame = append(flame, vec{X: x - s*(0.5*math.Sin(math.Pi*u)*(1-0.55*u)-wob*u)*on*fl, Y: y + s*2.4*u*on*fl})
		}
		lampLayer.Fills = append(lampLayer.Fills, ink.Fill{Loops: []geom.Path{flame}, Color: ink.RGBA(1, 0.62, 0.15, 0.9*on)})
		core := make(geom.Path, len(flame))
		for k, q := range flame {
			core[k] = vec{X: x + (q.X-x)*0.45, Y: y + (q.Y-y)*0.7}
		}
		lampLayer.Fills = append(lampLayer.Fills, ink.Fill{Loops: []geom.Path{core}, Color: ink.RGBA(1, 0.95, 0.7, on)})
	}
	add(lampLayer)

	// embers: points in space, streaking toward the camera
	in := surge(t)
	var em ink.Layer
	for _, e := range ember {
		if e[4] > 0.02+0.3*in {
			continue
		}
		zz := math.Mod(e[2]+e[3]*t, 60)
		p := vec{X: e[0] + 3*math.Sin(t*0.9+e[2]), Y: e[1] + 30*t*0.3*in, Z: zz}
		q := vec{X: p.X, Y: p.Y - 0.5 - 4*in, Z: zz - 0.5}
		x0, y0, d0, ok0 := cam.project(p)
		x1, y1, _, ok1 := cam.project(q)
		if !ok0 || !ok1 || d0 > 220 {
			continue
		}
		c := depthColour(d0*0.5, 20)
		glowLine(&em, geom.Path{{X: x0, Y: y0}, {X: x1, Y: y1}}, c, 1.1+3/math.Max(d0, 6)*6, 0.9)
	}
	add(em)

	// warp streaks rushing out of the temple's top during the surge
	if in > 0 {
		cx, cy, _, ok := cam.project(vec{Z: 26})
		if ok {
			var st ink.Layer
			for si, s := range streak {
				if si%3 != 0 {
					continue
				}
				r0 := 40 + 700*math.Mod(s[1]+t*(0.6+1.4*s[2])*in*1.2, 1)
				l := (18 + 220*s[2]) * in * (0.3 + r0/700)
				c0 := lerpC(ink.Color{R: 1, G: 0.9, B: 0.7, A: 1}, ink.Color{R: 0.75, G: 0.85, B: 1, A: 1}, s[2])
				glowLine(&st, geom.Path{{X: cx + r0*math.Cos(s[0]), Y: cy + r0*math.Sin(s[0])}, {X: cx + (r0+l)*math.Cos(s[0]), Y: cy + (r0+l)*math.Sin(s[0])}}, c0, 1.0, 0.4*in)
			}
			add(st)
		}
	}

	// the shockwave on the beat
	if dt := t - 12.2; dt > 0 && dt < 1.8 {
		cx, cy, _, ok := cam.project(vec{Z: 26})
		if ok {
			p := dt / 1.8
			var sw ink.Layer
			for k, rs := range []float64{1000} {
				r := rs * math.Pow(p, 0.7) * (1 - 0.15*float64(k))
				var ring geom.Path
				for i := 0; i <= 120; i++ {
					a := 2 * math.Pi * float64(i) / 120
					ring = append(ring, vec{X: cx + r*math.Cos(a), Y: cy + r*math.Sin(a)})
				}
				glowLine(&sw, ring, ink.Color{R: 1, G: 0.93, B: 0.72, A: 1}, 4*(1-p)+0.8, math.Min(1, 1.1*(1-p))*(1-0.35*float64(k)))
			}
			add(sw)
			if dt < 0.5 {
				add(ink.Layer{Fills: []ink.Fill{{Loops: []geom.Path{{{X: 0, Y: 0}, {X: W, Y: 0}, {X: W, Y: H}, {X: 0, Y: H}}}, Color: ink.RGBA(1, 0.92, 0.75, 0.25*math.Pow(1-dt/0.5, 2))}}})
			}
		}
	}

	// the title, and the closing fade
	if t > 16 {
		f := ease(seg(t, 16, 17.2))
		add(ink.Layer{Fills: []ink.Fill{{Loops: []geom.Path{{{X: 0, Y: 0}, {X: W, Y: 0}, {X: W, Y: H}, {X: 0, Y: H}}}, Color: ink.RGBA(0.012, 0.006, 0.02, 0.8*f)}}})
		scale := 11.0
		width := float64(len(titleT)) * 7 * scale
		loops := fx.PixelText(titleT, (W-width)/2, H*0.5-6*scale, scale)
		g := ease(seg(t, 16.5, 17.6))
		if g > 0.004 {
			add(ink.Layer{Fills: []ink.Fill{{Loops: loops, Color: ink.RGBA(1, 0.82, 0.4, g)}}})
		}
		bar := geom.Path{{X: (W - width) / 2, Y: H*0.5 - 9*scale}, {X: (W-width)/2 + width*ease(seg(t, 17.2, 18.6)), Y: H*0.5 - 9*scale}}
		var u ink.Layer
		glowLine(&u, bar, ink.Color{R: 1, G: 0.8, B: 0.35, A: 1}, 1.6, g)
		add(u)
	}
	return d
}

func main() {
	still := flag.Float64("still", -1, "write one frame at this time in seconds")
	sheet := flag.Int("sheet", 0, "write a contact sheet of this many frames across the film")
	fps := flag.Float64("fps", 24, "frames per second")
	title := flag.String("title", "RISE", "the closing title, in your own words")
	flag.Float64Var(&W, "w", 540, "width in pixels")
	flag.Float64Var(&H, "h", 960, "height in pixels")
	out := flag.String("out", "out/cinema/rising", "output path without extension")
	flag.Parse()
	titleT = *title
	os.MkdirAll("out/cinema", 0o755)
	switch {
	case *still >= 0:
		gg.SavePNG(*out+".png", frame(*still).Image(0, 0))
		fmt.Println("wrote", *out+".png")
	case *sheet > 0:
		cols := 6
		rows := (*sheet + cols - 1) / cols
		tw, th := int(W/3), int(H/3)
		dc := gg.NewContext(cols*tw, rows*th)
		dc.SetRGB(0, 0, 0)
		dc.Clear()
		for i := 0; i < *sheet; i++ {
			t := (float64(i) + 0.5) * duration / float64(*sheet)
			small := image.NewRGBA(image.Rect(0, 0, tw, th))
			xdraw.ApproxBiLinear.Scale(small, small.Bounds(), frame(t).Image(0, 0), image.Rect(0, 0, int(W), int(H)), xdraw.Src, nil)
			dc.DrawImage(small, (i%cols)*tw, (i/cols)*th)
			dc.SetRGB(1, 1, 1)
			dc.DrawString(fmt.Sprintf("%.1fs", t), float64((i%cols)*tw+4), float64((i/cols)*th+12))
		}
		gg.SavePNG(*out+"_sheet.png", dc.Image())
		fmt.Println("wrote", *out+"_sheet.png")
	default:
		n := int(duration * *fps)
		film := anim.FrameAnimation(func(u float64) anim.Frame { return frame(u * duration) })
		if err := film.WriteVideo(*out+".mp4", n, *fps, W, H); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("wrote", *out+".mp4", n, "frames")
	}
}

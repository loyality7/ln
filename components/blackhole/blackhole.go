// Package blackhole draws a black hole as ln ink: white line art on black space.
//
// It is an ordinary ln scene: a flat disk (shapes.Annulus) in a scene whose Bend field
// curves every camera ray (ln/scene, ln/physics), rendered through buffer.Gather like
// any other. The bent rays reach the far side of the disk twice, once directly and once
// arched over the shadow, so Gather is asked to keep two crossings per pixel. This
// package then only styles the result: each crossing's radius and angle become
// contour lines (ink.IsoLines), stars are dots, the shadow is left black.
package blackhole

import (
	"math"

	"github.com/loyality7/ln/ln/buffer"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/ink"
	"github.com/loyality7/ln/ln/light"
	"github.com/loyality7/ln/ln/physics"
	"github.com/loyality7/ln/ln/scene"
	"github.com/loyality7/ln/ln/shapes"
)

// Params describes the camera, the disk and the drawing. The mass is 1 (geometric
// units), so the horizon is at radius 2.
type Params struct {
	W, H     int
	FOV      float64 // vertical field of view, radians
	Distance float64 // camera distance from the hole
	Incline  float64 // camera height above the disk plane, radians (0 = edge on)
	Azimuth  float64 // camera position around the hole, radians
	Roll     float64 // camera roll, radians (0 keeps the disk level)
	DiskIn   float64 // inner and outer radius of the accretion disk
	DiskOut  float64
	Time     float64 // seconds; the disk turns, inner rings faster than outer
	Spin     float64 // how fast time turns the disk (1 = true orbital speed in these units)
	Rings    float64 // ring spacing: each ring is this much (e.g. 1.05 = 5%) farther out than the last
	Spokes   int     // spokes drawn around the disk
	Stars    float64 // 0..1, how many stars
	Colour   bool    // glowing colour (heat of the gas, Doppler shift, tinted stars) instead of white ink
	Heat     float64 // temperature of the innermost gas in kelvin when Colour is on (0 = 9000)
}

// Default is a level, near edge-on view.
func Default(w, h int) Params {
	return Params{W: w, H: h, FOV: 0.50, Distance: 235, Incline: 0.075, DiskIn: 6, DiskOut: 30,
		Spin: 4, Rings: 1.085, Spokes: 22, Stars: 1}
}

type vec = geom.Vector

// criticalImpact is how close to a mass a ray of light must aim to be captured (3*sqrt(3)*M).
func criticalImpact(g physics.Gravity) float64 { return 3 * math.Sqrt(3) * g.M }

// glow is the colour of gas at radius r seen with Doppler factor g: hotter inside, and
// shifted bluer where the gas comes toward us and redder where it moves away.
func (p Params) glow(r, g float64) ink.Color {
	t := p.Heat
	if t == 0 {
		t = 9000
	}
	red, green, blue := physics.Blackbody(physics.PowerLaw(r, p.DiskIn, t, 0.75) * math.Max(0.35, math.Min(2.2, g)))
	return ink.RGB(red, green, blue)
}

func sqr(x float64) float64 { return x * x }

func hash2(x, y float64) float64 {
	n := uint32(int32(x))*374761393 + uint32(int32(y))*668265263
	n = (n ^ (n >> 13)) * 1274126177
	return float64((n^(n>>16))&0xffffff) / float64(0xffffff)
}

// crossing is where a ray met the disk: its radius and azimuth, and how the Doppler
// shift of the gas there brightens or dims it.
type crossing struct {
	r, psi, g float64
	ok        bool
}

// sample is everything one pixel's ray found.
type sample struct {
	disk [2]crossing // the first and second times it crossed the disk plane
	star float64     // brightness of the star it ended on (0 if it fell into the hole)
}

// star is the brightness of the star, if any, in direction n. pix is the angle one
// pixel covers, so stars stay about a pixel across.
func star(n vec, pix, density float64) float64 {
	az, el := math.Atan2(n.Y, n.X), math.Asin(math.Max(-1, math.Min(1, n.Z)))
	const cell = 0.0055
	ci, cj := math.Floor(az/cell), math.Floor(el/cell)
	var best float64
	for di := -1.0; di <= 1; di++ {
		for dj := -1.0; dj <= 1; dj++ {
			h := hash2(ci+di, cj+dj)
			if h > 0.22*density {
				continue
			}
			sx, sy := (ci+di+hash2(ci+di+7, cj+dj-3))*cell, (cj+dj+hash2(ci+di-5, cj+dj+11))*cell
			d2 := sqr(az-sx)*sqr(math.Cos(el)) + sqr(el-sy)
			b := (math.Pow(h/(0.22*density), 2.4)*0.95 + 0.1) * math.Exp(-d2/(2*sqr(pix*0.75)))
			best = math.Max(best, b)
		}
	}
	return best
}

// Draw returns the black hole at p.Time as ink: white lines on black.
func Draw(p Params) *ink.Drawing {
	grav := physics.Gravity{M: 1}
	// the whole scene: a flat disk, seen through a gravity field that bends every camera ray
	sc := &scene.Scene{Bend: &physics.Field{Law: grav.Light(), Capture: grav.SchwarzschildRadius(), Escape: 80}}
	sc.Add(shapes.NewAnnulus(vec{}, p.DiskIn-1, p.DiskOut))

	e := p.Incline
	cam := geom.V(p.Distance*math.Cos(e)*math.Cos(p.Azimuth), p.Distance*math.Cos(e)*math.Sin(p.Azimuth), p.Distance*math.Sin(e))
	f := cam.Normalize().MulScalar(-1)
	right := f.Cross(geom.V(0, 0, 1)).Normalize()
	up := right.Cross(f)
	cr, sr := math.Cos(p.Roll), math.Sin(p.Roll)
	up = up.MulScalar(cr).Sub(right.MulScalar(sr)) // rolled camera up
	th := math.Tan(p.FOV / 2)
	pix := 2 * th / float64(p.H)

	g := buffer.Gather(sc, buffer.Camera{Eye: cam, Up: up, Fovy: geom.Degrees(p.FOV), Layers: 2}, p.W, p.H, light.Rig{})
	samples := make([]sample, p.W*p.H)
	for idx := range samples {
		for n, c := range g.Crosses[idx] {
			rc := math.Hypot(c.P.X, c.P.Y)
			// the gas orbits counter-clockwise at the circular speed; the light heads back toward us
			vel := geom.V(-c.P.Y, c.P.X, 0).Normalize().MulScalar(grav.CircularSpeed(rc))
			samples[idx].disk[n] = crossing{r: rc, psi: math.Atan2(c.P.Y, c.P.X), g: physics.Doppler(vel, c.Heading.MulScalar(-1)), ok: true}
		}
		if g.Exit[idx] != (vec{}) {
			samples[idx].star = star(g.Exit[idx], pix, p.Stars)
		}
	}

	k := float64(p.W) / 576 // line weights in pixels scale with the picture
	var space, lines ink.Layer
	space.Fills = append(space.Fills, ink.Fill{Loops: []geom.Path{{
		{X: 0, Y: 0}, {X: float64(p.W), Y: 0}, {X: float64(p.W), Y: float64(p.H)}, {X: 0, Y: float64(p.H)}}}})

	// the stars: small white dots, one fill each so overlapping dots do not cancel
	for idx, s := range samples {
		if s.star < 0.22 {
			continue
		}
		cx, cy := float64(idx%p.W)+0.5, float64(idx/p.W)+0.5
		r := (0.55 + 1.0*math.Min(1, s.star)) * k
		ring := make(geom.Path, 8)
		for q := range ring {
			a := 2 * math.Pi * float64(q) / 8
			ring[q] = geom.Vector{X: cx + r*math.Cos(a), Y: cy + r*math.Sin(a)}
		}
		starCol := ink.Color{}
		if p.Colour { // a few warm and cool stars, most near white
			red, green, blue := physics.Blackbody(3500 + 9000*hash2(cx, cy))
			starCol = ink.RGB(0.55+0.45*red, 0.55+0.45*green, 0.55+0.45*blue)
		}
		space.Fills = append(space.Fills, ink.Fill{Loops: []geom.Path{ring}, White: true, Color: starCol})
	}

	// the disk, for each of its two images (the near face, and the far side bent over
	// the shadow): rings of constant radius and spokes of constant angle as contour lines
	var levels []float64
	for r := p.DiskIn; r < p.DiskOut; r *= p.Rings {
		levels = append(levels, r)
	}
	for img := 0; img < 2; img++ {
		rf, sf, gf := make([]float64, len(samples)), make([]float64, len(samples)), make([]float64, len(samples))
		valid := make([]bool, len(samples))
		for i, s := range samples {
			c := s.disk[img]
			if !c.ok {
				continue
			}
			valid[i], rf[i], gf[i] = true, c.r, c.g
			// sin(K*angle) is zero on K evenly spaced spokes; inner gas laps the outer
			psi := c.psi + p.Time*p.Spin*grav.AngularVelocity(c.r)
			sf[i] = math.Sin(float64(p.Spokes) * psi)
		}
		weight := 1.0
		if img == 1 {
			weight = 0.8
		}
		// the side of the disk coming toward us is brighter (Doppler beaming), so its lines are heavier
		beamed := func(path geom.Path, base float64) []float64 {
			w := make([]float64, len(path))
			for i, pt := range path {
				x, y := min(p.W-1, max(0, int(pt.X))), min(p.H-1, max(0, int(pt.Y)))
				w[i] = base * math.Max(0.6, math.Min(1.7, physics.Beaming(gf[y*p.W+x], 1.5)))
			}
			return w
		}
		for li, paths := range ink.IsoLines(rf, valid, p.W, p.H, levels, 1, 1) {
			hot := physics.Radiance(physics.PowerLaw(levels[li], p.DiskIn, 1, 0.75)) // hotter, so heavier, toward the middle
			base := (0.45 + 1.15*math.Pow(hot, 0.3)) * k * weight
			for _, path := range paths {
				line := ink.Line{Pts: path, W: beamed(path, base), White: true}
				if p.Colour {
					m := path[len(path)/2]
					line.Color = p.glow(levels[li], gf[min(p.H-1, max(0, int(m.Y)))*p.W+min(p.W-1, max(0, int(m.X)))])
				}
				lines.Lines = append(lines.Lines, line)
			}
		}
		if p.Spokes > 0 && img == 0 {
			// spokes wound tighter than a pixel are noise, not lines: leave those cells out
			loose := append([]bool(nil), valid...)
			for y := 0; y+1 < p.H; y++ {
				for x := 0; x+1 < p.W; x++ {
					i := y*p.W + x
					if math.Abs(sf[i+1]-sf[i]) > 0.7 || math.Abs(sf[i+p.W]-sf[i]) > 0.7 {
						loose[i], loose[i+1], loose[i+p.W] = false, false, false
					}
				}
			}
			for _, path := range ink.IsoLines(sf, loose, p.W, p.H, []float64{0}, 1, 1)[0] {
				line := ink.Line{Pts: path, W: beamed(path, 0.4*k), White: true}
				if p.Colour {
					line.Color = ink.RGBA(1, 0.62, 0.3, 0.55)
				}
				lines.Lines = append(lines.Lines, line)
			}
		}
	}

	// the photon ring: a thin bright circle where light nearly orbits the hole
	ringR := math.Tan(criticalImpact(grav)/p.Distance) / th * float64(p.H) / 2
	ring := make(geom.Path, 121)
	w := make([]float64, len(ring))
	for q := range ring {
		a := 2 * math.Pi * float64(q) / 120
		ring[q] = geom.Vector{X: float64(p.W)/2 + ringR*math.Cos(a), Y: float64(p.H)/2 + ringR*math.Sin(a)}
		w[q] = 1.4 * k
	}
	photon := ink.Line{Pts: ring, W: w, White: true}
	if p.Colour {
		photon.Color = ink.RGB(1, 0.93, 0.8)
	}
	lines.Lines = append(lines.Lines, photon)

	return ink.NewDrawing(float64(p.W), float64(p.H), space, lines)
}

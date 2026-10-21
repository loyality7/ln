// Package blackhole draws a black hole as ln ink: white line art on black space.
//
// This package is only the scene: a disk, a camera and the drawing. The physics lives
// in ln/physics, which any other scene can reuse: gravity bending each pixel's ray,
// orbital speeds, the Doppler shift. Each ray is followed to find which ring and spoke
// of the disk it lands on, so the far side of the disk arches over the top and under
// the shadow. Rings of constant radius and spokes of constant angle then become contour
// lines (ink.IsoLines); stars are dots; the shadow is left black.
package blackhole

import (
	"math"

	"github.com/loyality7/ln/ln/buffer"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/ink"
	"github.com/loyality7/ln/ln/physics"
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
}

// Default is a level, near edge-on view.
func Default(w, h int) Params {
	return Params{W: w, H: h, FOV: 0.50, Distance: 235, Incline: 0.075, DiskIn: 6, DiskOut: 30,
		Spin: 25, Rings: 1.085, Spokes: 22, Stars: 1}
}

type vec = geom.Vector

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

// trace follows one pixel's ray through the field and records where it meets the disk.
func (p Params) trace(field physics.Field, cam, d vec, pix float64) sample {
	var out sample
	hits := 0
	outcome, exit := field.Trace(cam, d, func(s physics.Segment) bool {
		if hits < 2 {
			if at, _, ok := s.CrossPlane(vec{}, geom.V(0, 0, 1)); ok {
				rc := math.Hypot(at.X, at.Y)
				if rc >= p.DiskIn-1 && rc <= p.DiskOut {
					// the gas orbits counter-clockwise at the circular speed; the light heads back toward us
					vel := geom.V(-at.Y, at.X, 0).Normalize().MulScalar(physics.Gravity{M: 1}.CircularSpeed(rc))
					g := physics.Doppler(vel, s.From.Sub(s.To).Normalize())
					out.disk[hits] = crossing{r: rc, psi: math.Atan2(at.Y, at.X), g: g, ok: true}
					hits++
				}
			}
		}
		return true
	})
	if outcome == physics.Escaped {
		out.star = star(exit, pix, p.Stars)
	}
	return out
}

// Draw returns the black hole at p.Time as ink: white lines on black.
func Draw(p Params) *ink.Drawing {
	grav := physics.Gravity{M: 1}
	field := physics.Field{Law: grav.Light(), Capture: grav.SchwarzschildRadius(), Escape: 80}

	e := p.Incline
	cam := geom.V(p.Distance*math.Cos(e)*math.Cos(p.Azimuth), p.Distance*math.Cos(e)*math.Sin(p.Azimuth), p.Distance*math.Sin(e))
	f := cam.Normalize().MulScalar(-1)
	right := f.Cross(geom.V(0, 0, 1)).Normalize()
	up := right.Cross(f)
	cr, sr := math.Cos(p.Roll), math.Sin(p.Roll)
	right, up = right.MulScalar(cr).Add(up.MulScalar(sr)), up.MulScalar(cr).Sub(right.MulScalar(sr))
	th := math.Tan(p.FOV / 2)
	aspect := float64(p.W) / float64(p.H)
	pix := 2 * th / float64(p.H)

	// ray-cast every pixel (row 0 at the bottom, like the other buffers)
	samples := make([]sample, p.W*p.H)
	buffer.Rows(p.H, func(j int) {
		for i := 0; i < p.W; i++ {
			x := (2*(float64(i)+0.5)/float64(p.W) - 1) * aspect * th
			y := (2*(float64(j)+0.5)/float64(p.H) - 1) * th
			d := f.Add(right.MulScalar(x)).Add(up.MulScalar(y)).Normalize()
			samples[j*p.W+i] = p.trace(field, cam, d, pix)
		}
	})

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
		space.Fills = append(space.Fills, ink.Fill{Loops: []geom.Path{ring}, White: true})
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
			hot := physics.Radiance(physics.ThinDiskTemperature(levels[li], p.DiskIn, 1)) // hotter, so heavier, toward the middle
			base := (0.45 + 1.15*math.Pow(hot, 0.3)) * k * weight
			for _, path := range paths {
				lines.Lines = append(lines.Lines, ink.Line{Pts: path, W: beamed(path, base), White: true})
			}
		}
		if p.Spokes > 0 && img == 0 {
			for _, path := range ink.IsoLines(sf, valid, p.W, p.H, []float64{0}, 1, 1)[0] {
				lines.Lines = append(lines.Lines, ink.Line{Pts: path, W: beamed(path, 0.4*k), White: true})
			}
		}
	}

	// the photon ring: a thin bright circle where light nearly orbits the hole
	ringR := math.Tan(grav.CriticalImpactParameter()/p.Distance) / th * float64(p.H) / 2
	ring := make(geom.Path, 121)
	w := make([]float64, len(ring))
	for q := range ring {
		a := 2 * math.Pi * float64(q) / 120
		ring[q] = geom.Vector{X: float64(p.W)/2 + ringR*math.Cos(a), Y: float64(p.H)/2 + ringR*math.Sin(a)}
		w[q] = 1.4 * k
	}
	lines.Lines = append(lines.Lines, ink.Line{Pts: ring, W: w, White: true})

	return ink.NewDrawing(float64(p.W), float64(p.H), space, lines)
}

package blackhole

import (
	"math"

	"github.com/loyality7/ln/ln/buffer"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/glow"
	"github.com/loyality7/ln/ln/light"
	"github.com/loyality7/ln/ln/physics"
	"github.com/loyality7/ln/ln/scene"
	"github.com/loyality7/ln/ln/shapes"
)

// Emit draws the black hole as light on a glow canvas: the same bent rays as Draw, but each pixel gets the
// colour of the gas it sees instead of a line. Gas at radius r glows at the temperature of a thin heated disk
// (hotter inside), shifted bluer where it moves toward us and redder where it moves away (Doppler), dimmed
// by gravity near the horizon, and brightened on the approaching side (beaming). The disk is seen twice,
// directly and bent over the shadow, which makes the photon ring. Stars behind the hole are bent too.
// Wrap it with glow.Frame to play it as video, or call .Image(glow.Soft()).
func Emit(p Params) *glow.Canvas {
	grav := physics.Gravity{M: 1}
	sc := &scene.Scene{Bend: &physics.Field{Law: grav.Light(), Capture: grav.SchwarzschildRadius(), Escape: 80}}
	sc.Add(shapes.NewAnnulus(vec{}, p.DiskIn-1, p.DiskOut))

	e := p.Incline
	cam := geom.V(p.Distance*math.Cos(e)*math.Cos(p.Azimuth), p.Distance*math.Cos(e)*math.Sin(p.Azimuth), p.Distance*math.Sin(e))
	f := cam.Normalize().MulScalar(-1)
	right := f.Cross(geom.V(0, 0, 1)).Normalize()
	up := right.Cross(f)
	cr, sr := math.Cos(p.Roll), math.Sin(p.Roll)
	up = up.MulScalar(cr).Sub(right.MulScalar(sr))
	th := math.Tan(p.FOV / 2)
	pix := 2 * th / float64(p.H)

	g := buffer.Gather(sc, buffer.Camera{Eye: cam, Up: up, Fovy: geom.Degrees(p.FOV), Layers: 2}, p.W, p.H, light.Rig{})
	c := glow.New(p.W, p.H)
	heat := p.Heat
	if heat == 0 {
		heat = 5000 // deep amber in the middle, deep red outside: hot enough to glow gold without washing out to white
	}
	for k := range g.OK {
		x, y := k%p.W, p.H-1-k/p.W // buffers run bottom-up, canvases top-down
		for n, s := range g.Crosses[k] {
			r := math.Hypot(s.P.X, s.P.Y)
			// the heat of the gas depends on its radius alone, so every side of every ring glows
			kelvin := physics.PowerLaw(r, p.DiskIn, heat, 0.75)
			// Doppler gently brightens the side coming toward us and dims the side moving away
			dop := physics.Doppler(geom.V(-s.P.Y, s.P.X, 0).Normalize().MulScalar(grav.CircularSpeed(r)), s.Heading.MulScalar(-1))
			dop = math.Max(0.7, math.Min(1.45, dop))
			// wide smooth bands of light around the hole, a few per octave of radius
			ring := 0.5 + 0.5*math.Cos(2*math.Pi*math.Log(r/p.DiskIn)*2.1)
			ring = math.Pow(ring, 1.4)
			// the gas shears: inner gas laps the outer, so the bands wander as time passes
			psi := math.Atan2(s.P.Y, s.P.X) + p.Time*p.Spin*grav.AngularVelocity(r)
			swirl := 0.8 + 0.14*math.Sin(5*psi+9*math.Log(r)) + 0.06*math.Sin(13*psi-17*math.Log(r))
			v := math.Pow(kelvin/heat, 1.9) * (0.28 + 0.72*ring) * swirl * math.Pow(dop, 1.5)
			if n == 1 {
				v *= 0.9 // the bent image is a little dimmer
			}
			red, green, blue := ramp(v)
			power := 0.15 + 2.1*math.Pow(math.Min(v, 1.4), 1.6)
			c.Add(x, y, red*power, green*power, blue*power)
		}
		if exit := g.Exit[k]; exit != (vec{}) {
			if b := star(exit, pix, p.Stars); b > 0.2 {
				c.Add(x, y, 1.3*b, 1.3*b, 1.5*b)
			}
		}
	}
	// the photon ring: a thin bright orange circle where light nearly orbits the hole
	th2 := math.Tan(p.FOV / 2)
	ringR := math.Tan(criticalImpact(grav)/p.Distance) / th2 * float64(p.H) / 2
	for i := 0; i < 720; i++ {
		a := 2 * math.Pi * float64(i) / 720
		c.Spot(float64(p.W)/2+ringR*math.Cos(a), float64(p.H)/2+ringR*math.Sin(a), 0.9, 0.9, 0.5, 0.12)
	}
	if p.Streak > 0 { // starlight pulled into lines as the camera dives in
		for i := 0; i < 260; i++ {
			a := 2 * math.Pi * hash2(float64(i), 3)
			r0 := (0.12 + 0.9*math.Mod(hash2(float64(i), 9)+p.Time*(0.15+0.7*hash2(float64(i), 5))*p.Streak, 1)) * float64(p.H) * 0.6
			l := (0.02 + 0.09*hash2(float64(i), 11)) * float64(p.H) * p.Streak
			cx, cy := float64(p.W)/2, float64(p.H)/2
			bright := 0.5 + 0.9*hash2(float64(i), 13)
			c.Streak(cx+r0*math.Cos(a), cy+r0*math.Sin(a), cx+(r0+l)*math.Cos(a), cy+(r0+l)*math.Sin(a), 0.7, 0.8*bright, 0.85*bright, bright)
		}
	}
	return c
}

// ramp is the colour of glowing gas of brightness v: dark brown, then orange, gold, and cream-white at the
// brightest, like heated metal.
func ramp(v float64) (r, g, b float64) {
	stops := [...][4]float64{{0, 0.03, 0.01, 0.0}, {0.18, 0.4, 0.1, 0.01}, {0.4, 0.85, 0.3, 0.04}, {0.65, 1, 0.62, 0.16}, {0.9, 1, 0.86, 0.5}, {1.3, 1, 0.97, 0.88}}
	v = math.Max(0, math.Min(1.3, v))
	for i := 1; i < len(stops); i++ {
		if v <= stops[i][0] {
			t := (v - stops[i-1][0]) / (stops[i][0] - stops[i-1][0])
			return stops[i-1][1] + t*(stops[i][1]-stops[i-1][1]), stops[i-1][2] + t*(stops[i][2]-stops[i-1][2]), stops[i-1][3] + t*(stops[i][3]-stops[i-1][3])
		}
	}
	return 1, 0.97, 0.88
}

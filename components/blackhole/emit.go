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
			// Doppler only tints and gently brightens: bluer and a little brighter coming toward us
			dop := physics.Doppler(geom.V(-s.P.Y, s.P.X, 0).Normalize().MulScalar(grav.CircularSpeed(r)), s.Heading.MulScalar(-1))
			dop = math.Max(0.75, math.Min(1.3, dop))
			red, green, blue := physics.Blackbody(kelvin * 0.55 * dop)
			// the rings of the line version, as bands of light: one every few percent of radius
			ring := 0.5 + 0.5*math.Cos(2*math.Pi*math.Log(r/p.DiskIn)/math.Log(p.Rings*1.9))
			// swirling bands in the gas: inner gas laps the outer, so the pattern shears as time passes
			psi := math.Atan2(s.P.Y, s.P.X) + p.Time*p.Spin*grav.AngularVelocity(r)
			swirl := 0.7 + 0.2*math.Sin(11*psi+7*math.Log(r)) + 0.1*math.Sin(23*psi-13*math.Log(r))
			power := math.Pow(kelvin/heat, 2.6) * (0.35 + 0.65*ring) * swirl * math.Pow(dop, 1.2) * 1.5
			if n == 1 {
				power *= 0.85 // the bent image is a little dimmer
			}
			c.Add(x, y, red*power, green*power, blue*power)
		}
		if exit := g.Exit[k]; exit != (vec{}) {
			if b := star(exit, pix, p.Stars); b > 0.2 {
				c.Add(x, y, 1.3*b, 1.3*b, 1.5*b)
			}
		}
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

// Package env models the world's time of day: where the sun and moon are, how
// much light they give, and what the sky looks like. Animate Hour and the whole
// scene moves through sunrise, noon, sunset and night with correct shadows.
//
// World axes: x east, y north, z up.
package env

import (
	"math"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/light"
)

// SkyStyle is how the sky is drawn: tones are darkness, 0 white paper .. 1 black.
// The zero value means "use DefaultSky".
type SkyStyle struct {
	DayTone, DayZenith     float64 // daytime horizon tone, and extra darkness toward the zenith
	NightTone, NightZenith float64
	CloudTone              float64 // extra zenith darkness added by overcast
	Glow, GlowSpread       float64 // brightening near the sun at twilight: strength, width (radians^2)
	Stars                  int
	SunSize, MoonSize      float64 // angular radius, radians
}

func DefaultSky() SkyStyle {
	return SkyStyle{DayTone: 0.03, DayZenith: 0.07, NightTone: 0.88, NightZenith: 0.10, CloudTone: 0.25,
		Glow: 0.62, GlowSpread: 0.9, Stars: 420, SunSize: 0.045, MoonSize: 0.04}
}

func (e *Environment) sky() SkyStyle {
	if e.Sky == (SkyStyle{}) {
		return DefaultSky()
	}
	return e.Sky
}

type Environment struct {
	Sky       SkyStyle // how the sky looks; zero value = DefaultSky
	Hour      float64  // local solar time, 0-24 (values past 24 wrap, so 17 -> 30 is dusk to dawn)
	Latitude  float64  // degrees north
	DayOfYear float64  // 1-365; sets the season (80 = equinox, 172 = long summer days)
	MoonPhase float64  // 0 new, 0.25 first quarter, 0.5 full, 0.75 last quarter
	Clouds    float64  // 0 clear .. 1 overcast; dims direct sun
	Night     float64  // extra darkness added to the whole world, 0-1 (mood control)
	Exposure  float64  // brightness gain, like a camera; 0 means 1. Twilight is boosted automatically
}

// New is a clear equinox noon at latitude 35.
func New() *Environment {
	return &Environment{Hour: 12, Latitude: 35, DayOfYear: 80, MoonPhase: 0.5}
}

// At is New set to the given hour.
func At(hour float64) *Environment {
	e := New()
	e.Hour = hour
	return e
}

// Named times of day, handy for stills and cuts.
func Sunrise() *Environment    { return At(6) }
func Morning() *Environment    { return At(9) }
func Noon() *Environment       { return At(12) }
func Afternoon() *Environment  { return At(15) }
func GoldenHour() *Environment { return At(17.2) }
func Sunset() *Environment     { return At(18) }
func Dusk() *Environment       { return At(18.7) }
func Night() *Environment      { return At(23) }
func Midnight() *Environment   { return At(0) }

func rad(d float64) float64 { return d * math.Pi / 180 }

// body returns the unit direction toward a celestial body with the given hour
// angle (radians from the meridian) and declination.
func (e *Environment) body(hourAngle, decl float64) geom.Vector {
	lat := rad(e.Latitude)
	east := -math.Cos(decl) * math.Sin(hourAngle)
	north := math.Sin(decl)*math.Cos(lat) - math.Cos(decl)*math.Sin(lat)*math.Cos(hourAngle)
	up := math.Sin(lat)*math.Sin(decl) + math.Cos(lat)*math.Cos(decl)*math.Cos(hourAngle)
	return geom.Vector{X: east, Y: north, Z: up}.Normalize()
}

func (e *Environment) declination() float64 {
	return rad(23.44) * math.Sin(2*math.Pi*(284+e.DayOfYear)/365)
}

// SunDir is the unit direction toward the sun (Z < 0 means below the horizon).
func (e *Environment) SunDir() geom.Vector {
	return e.body(rad((e.Hour-12)*15), e.declination())
}

// MoonDir is the unit direction toward the moon. It trails the sun by its phase.
func (e *Environment) MoonDir() geom.Vector {
	return e.body(rad((e.Hour-12)*15)+e.MoonPhase*2*math.Pi, -e.declination()*0.5)
}

func smoothstep(a, b, x float64) float64 {
	t := math.Max(0, math.Min(1, (x-a)/(b-a)))
	return t * t * (3 - 2*t)
}

// Daylight is 0 at night, 1 in full day, easing through twilight. It drives
// how much light the sun gives the ground.
func (e *Environment) Daylight() float64 {
	return smoothstep(-0.10, 0.28, e.SunDir().Z)
}

// skyLight is how bright the sky is. The sky glows before the ground is lit
// (dawn) and stays lit after the sun has set (dusk), so it turns on earlier.
func (e *Environment) skyLight() float64 {
	return smoothstep(-0.20, 0.10, e.SunDir().Z)
}

func (e *Environment) moonLit() float64 { return (1 - math.Cos(2*math.Pi*e.MoonPhase)) / 2 }

// Rig is the lighting for this time of day: sun, moon and sky ambient light.
// Extra lights (lamps, windows) are added on top.
func (e *Environment) Rig(extra ...light.Light) light.Rig {
	day := e.Daylight()
	sun := day * (1 - 0.55*e.Clouds)
	moon := 0.14 * e.moonLit() * smoothstep(-0.05, 0.2, e.MoonDir().Z) * (1 - day)
	gain := e.Exposure
	if gain == 0 {
		gain = 1
	}
	// low sun lights surfaces weakly by the cosine law; compensate so dusk stays drawable
	r := light.Rig{Shadows: true, Exposure: gain * (1 + 1.3*4*day*(1-day))}
	if sun > 0.01 {
		r.Lights = append(r.Lights, light.Directional{Dir: e.SunDir(), Power: sun})
	}
	if moon > 0.01 {
		r.Lights = append(r.Lights, light.Directional{Dir: e.MoonDir(), Power: moon})
	}
	r.Lights = append(r.Lights, extra...)
	r.Ambient = math.Max(0, math.Min(0.9, 0.04+0.33*day*(1-0.3*e.Clouds)+0.03*e.moonLit()*(1-day)-0.4*e.Night))
	return r
}

// Darkness is the sky's tone looking along dir: 0 white paper .. 1 black.
// It blends a day sky and a night sky, with a bright band near the sun at twilight.
func (e *Environment) Darkness(dir geom.Vector) float64 {
	k := e.sky()
	day := e.skyLight()
	zen := math.Max(0, dir.Z)
	dayTone := k.DayTone + k.DayZenith*zen + k.CloudTone*e.Clouds*zen
	nightTone := k.NightTone + k.NightZenith*zen
	d := nightTone + (dayTone-nightTone)*day
	twilight := 4 * day * (1 - day) // peaks while the sun is at the horizon
	sun := e.SunDir()
	sun.Z = math.Max(sun.Z, -0.05)
	ang := math.Acos(math.Max(-1, math.Min(1, dir.Normalize().Dot(sun.Normalize()))))
	d -= k.Glow * twilight * math.Exp(-ang*ang/k.GlowSpread)
	return math.Max(0, math.Min(1, d+e.Night*0.5))
}

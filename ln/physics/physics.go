// Package physics holds the general laws behind ln's physical scenes, so any new
// scene can reuse them: numerical integration of motion, gravity bending the path of
// light or matter, orbital mechanics, the relativistic Doppler shift and beaming, and
// thermal radiation. Nothing here knows about any particular object.
//
// Gravity uses geometric units (G = c = 1): mass, distance and time all share one
// unit, so a mass of 1 has a Schwarzschild radius of 2. Pick any unit you like for
// the scene and the ratios stay right.
package physics

import (
	"math"

	"github.com/loyality7/ln/ln/geom"
)

// ---- numerical integration ------------------------------------------------------

// Step2 advances the second-order equation y” = f(y, y') by h with the classic
// fourth-order Runge-Kutta method. It returns the new y and y'.
func Step2(f func(y, v float64) float64, y, v, h float64) (float64, float64) {
	k1y, k1v := v, f(y, v)
	k2y, k2v := v+0.5*h*k1v, f(y+0.5*h*k1y, v+0.5*h*k1v)
	k3y, k3v := v+0.5*h*k2v, f(y+0.5*h*k2y, v+0.5*h*k2v)
	k4y, k4v := v+h*k3v, f(y+h*k3y, v+h*k3v)
	return y + h/6*(k1y+2*k2y+2*k3y+k4y), v + h/6*(k1v+2*k2v+2*k3v+k4v)
}

// Step3 advances a particle (or a ray treated as one) under an acceleration that
// depends on where it is and how fast it moves, p” = a(p, v), by h with RK4. It
// returns the new position and velocity. This is the general tool for any force field.
func Step3(a func(p, v geom.Vector) geom.Vector, p, v geom.Vector, h float64) (geom.Vector, geom.Vector) {
	k1p, k1v := v, a(p, v)
	k2p, k2v := v.Add(k1v.MulScalar(h/2)), a(p.Add(k1p.MulScalar(h/2)), v.Add(k1v.MulScalar(h/2)))
	k3p, k3v := v.Add(k2v.MulScalar(h/2)), a(p.Add(k2p.MulScalar(h/2)), v.Add(k2v.MulScalar(h/2)))
	k4p, k4v := v.Add(k3v.MulScalar(h)), a(p.Add(k3p.MulScalar(h)), v.Add(k3v.MulScalar(h)))
	dp := k1p.Add(k2p.MulScalar(2)).Add(k3p.MulScalar(2)).Add(k4p).MulScalar(h / 6)
	dv := k1v.Add(k2v.MulScalar(2)).Add(k3v.MulScalar(2)).Add(k4v).MulScalar(h / 6)
	return p.Add(dp), v.Add(dv)
}

// ---- gravity --------------------------------------------------------------------

// Gravity is a point mass in geometric units (see the package comment).
type Gravity struct{ M float64 }

// SchwarzschildRadius is the radius a mass must be squeezed inside to trap light.
func (g Gravity) SchwarzschildRadius() float64 { return 2 * g.M }

// CircularSpeed is the speed of a circular orbit at radius r (Kepler).
func (g Gravity) CircularSpeed(r float64) float64 { return math.Sqrt(g.M / r) }

// AngularVelocity is how fast a circular orbit at radius r turns, radians per unit
// time. It falls with r, so inner orbits lap outer ones (differential rotation).
func (g Gravity) AngularVelocity(r float64) float64 { return math.Sqrt(g.M / (r * r * r)) }

// Period is the time one circular orbit at radius r takes.
func (g Gravity) Period(r float64) float64 { return 2 * math.Pi / g.AngularVelocity(r) }

// EscapeSpeed is the speed needed at radius r to leave for good.
func (g Gravity) EscapeSpeed(r float64) float64 { return math.Sqrt(2 * g.M / r) }

// TimeDilation is how slowly a clock at radius r ticks compared with one far away
// (1 far away, 0 at the Schwarzschild radius).
func (g Gravity) TimeDilation(r float64) float64 { return math.Sqrt(math.Max(0, 1-2*g.M/r)) }

// Deflection is the angle gravity bends a passing ray of light through, for a ray
// that stays far from the mass (impact parameter b much larger than M): 4M/b.
func (g Gravity) Deflection(b float64) float64 { return 4 * g.M / b }

// PhotonRadius is the radius where light can orbit the mass, if the mass is compact enough.
func (g Gravity) PhotonRadius() float64 { return 3 * g.M }

// CriticalImpactParameter is how close to the mass a ray must aim to be captured.
func (g Gravity) CriticalImpactParameter() float64 { return 3 * math.Sqrt(3) * g.M }

// InnermostStableOrbit is the smallest radius a circular orbit stays stable at.
func (g Gravity) InnermostStableOrbit() float64 { return 6 * g.M }

// ---- orbits: how a path bends around a central mass -------------------------------

// OrbitLaw is the physics of a central force written in the form that makes orbits
// simple: with u = 1/r and phi the angle swept around the centre, the path obeys
// u” = Law(u) (derivatives with respect to phi). The same integrator then handles
// every kind of orbit, and light, by changing only the law.
type OrbitLaw func(u float64) float64

// Flat is empty space: u” = -u, whose solutions are straight lines.
func Flat() OrbitLaw { return func(u float64) float64 { return -u } }

// Newtonian is an inverse-square pull: u” = -u + mu, where mu = GM / L^2 and L is
// the orbit's angular momentum per unit mass. Paths are the conic sections.
func Newtonian(mu float64) OrbitLaw { return func(u float64) float64 { return -u + mu } }

// Light is how gravity bends light, with general relativity's correction: u” = -u + 3Mu^2.
func (g Gravity) Light() OrbitLaw { return func(u float64) float64 { return -u + 3*g.M*u*u } }

// Matter is a massive particle with angular momentum l per unit mass:
// u” = -u + M/l^2 + 3Mu^2. Orbits precess instead of closing.
func (g Gravity) Matter(l float64) OrbitLaw {
	return func(u float64) float64 { return -u + g.M/(l*l) + 3*g.M*u*u }
}

// ---- relativity -----------------------------------------------------------------

// Gamma is the Lorentz factor for a speed v as a fraction of light speed (c = 1).
func Gamma(v float64) float64 { return 1 / math.Sqrt(math.Max(1e-12, 1-v*v)) }

// Doppler is the frequency ratio g = observed / emitted for light from a source moving
// with velocity vel (fractions of c), where travel is the unit direction the light
// takes toward the observer. g > 1 for a source approaching the observer (blueshift),
// g < 1 receding (redshift).
func Doppler(vel, travel geom.Vector) float64 {
	v2 := math.Min(vel.LengthSquared(), 0.9999)
	return math.Sqrt(1-v2) / math.Max(0.15, 1-vel.Dot(travel))
}

// Beaming is the brightness change of a moving source: intensity scales as g^power
// (about 3 for a continuous glow, 4 for a single blob, 2 for a photon count).
func Beaming(g, power float64) float64 { return math.Pow(g, power) }

// ---- thermal radiation ----------------------------------------------------------

// ThinDiskTemperature is the temperature of a thin heated disk at radius r, falling
// as r^(-3/4) from tIn at its inner edge rIn.
func ThinDiskTemperature(r, rIn, tIn float64) float64 { return tIn * math.Pow(rIn/r, 0.75) }

// Radiance is the power a surface at temperature t gives off per unit area, in
// units where the Stefan-Boltzmann constant is 1: it grows as t^4.
func Radiance(t float64) float64 { return t * t * t * t }

// Blackbody is the colour (each channel 0..1) of a glowing surface at t kelvin, from
// deep red near 1000 K through orange and white to blue-white near 10000 K.
func Blackbody(kelvin float64) (r, g, b float64) {
	t := math.Max(1000, math.Min(40000, kelvin)) / 100
	clamp := func(x float64) float64 { return math.Max(0, math.Min(1, x/255)) }
	if t <= 66 {
		r = 255
		g = 99.4708025861*math.Log(t) - 161.1195681661
	} else {
		r = 329.698727446 * math.Pow(t-60, -0.1332047592)
		g = 288.1221695283 * math.Pow(t-60, -0.0755148492)
	}
	switch {
	case t >= 66:
		b = 255
	case t <= 19:
		b = 0
	default:
		b = 138.5177312231*math.Log(t-10) - 305.0447927307
	}
	return clamp(r), clamp(g), clamp(b)
}

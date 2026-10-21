// Package light models illumination behind one small interface, so a scene can
// mix suns, moons, lamps and ambient sky light without the renderer knowing which.
package light

import (
	"math"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/scene"
)

// Light says where it is and how strong it is, as seen from a surface point.
type Light interface {
	// Toward returns the unit direction from p to the light, the distance to it
	// (geom.INF for directional lights) and its power at p in [0,1+].
	Toward(p geom.Vector) (dir geom.Vector, dist float64, power float64)
}

// Directional is light from infinitely far away (sun, moon).
type Directional struct {
	Dir   geom.Vector // direction from the scene toward the light
	Power float64
}

func (d Directional) Toward(geom.Vector) (geom.Vector, float64, float64) {
	return d.Dir.Normalize(), geom.INF, d.Power
}

// Point is a lamp that fades with distance: full power at the lamp, half at Radius.
type Point struct {
	Pos    geom.Vector
	Power  float64
	Radius float64
	Size   float64 // physical size of the lamp: shadow rays stop this far short so it does not shade itself
}

func (l Point) Toward(p geom.Vector) (geom.Vector, float64, float64) {
	v := l.Pos.Sub(p)
	d := v.Length()
	r := math.Max(l.Radius, 1e-6)
	return v.MulScalar(1 / math.Max(d, 1e-9)), math.Max(0, d-l.Size), l.Power / (1 + (d/r)*(d/r))
}

// Spot is a Point restricted to a cone around Axis (half-angle Cone, radians).
type Spot struct {
	Point
	Axis geom.Vector
	Cone float64
}

func (s Spot) Toward(p geom.Vector) (geom.Vector, float64, float64) {
	dir, dist, power := s.Point.Toward(p)
	if dir.Dot(s.Axis.Normalize().MulScalar(-1)) < math.Cos(s.Cone) {
		power = 0
	}
	return dir, dist, power
}

// Rig is everything that lights a scene.
type Rig struct {
	Lights   []Light
	Ambient  float64 // light that reaches everywhere, even in shadow
	Shadows  bool    // trace shadow rays
	Exposure float64 // brightness gain applied to all light, like a camera's exposure (0 = 1)
}

// Sun is the simplest rig: one directional light plus a little ambient.
func Sun(dir geom.Vector) Rig {
	return Rig{Lights: []Light{Directional{Dir: dir, Power: 1}}, Ambient: 0.2, Shadows: true}
}

// Dark is how dark a surface at p with normal n looks: 0 fully lit, 1 black.
func (r Rig) Dark(s *scene.Scene, p, n geom.Vector) float64 {
	var lit float64
	for _, l := range r.Lights {
		dir, dist, power := l.Toward(p)
		facing := math.Max(0, n.Dot(dir))
		if facing*power <= 0 {
			continue
		}
		if r.Shadows {
			origin := p.Add(n.MulScalar(1e-3 * math.Max(1, p.Length()*0.01)))
			if h := s.Intersect(geom.Ray{Origin: origin, Direction: dir}); h.Ok() && h.T < dist {
				continue
			}
		}
		lit += facing * power
	}
	gain := r.Exposure
	if gain == 0 {
		gain = 1
	}
	return 1 - math.Min(1, gain*(r.Ambient+(1-r.Ambient)*lit))
}

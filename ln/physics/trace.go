package physics

import (
	"math"

	"github.com/loyality7/ln/ln/geom"
)

// Field is a central force acting on a path: a centre, the law that bends paths around
// it, and the radii at which a path counts as captured or escaped.
type Field struct {
	Center  geom.Vector
	Law     OrbitLaw
	Capture float64 // a path that falls inside this radius is absorbed (a horizon, a surface)
	Escape  float64 // a path that leaves beyond this radius, heading outward, is free
	MaxTurn float64 // stop after sweeping this much angle around the centre (0 = 3*pi)
}

// Outcome is how a traced path ended.
type Outcome int

const (
	Escaped  Outcome = iota // left to infinity; Exit is the direction it was heading
	Captured                // fell inside the Capture radius
	Stopped                 // the visitor ended it, or it swept the whole MaxTurn
)

// Segment is one small step of a traced path, in world space.
type Segment struct {
	From, To   geom.Vector
	R0, R1     float64 // distance from the centre at each end
	Phi0, Phi1 float64 // angle swept around the centre so far
}

// CrossPlane reports where the segment crosses the plane through point on with the
// given unit normal, and how far along the segment (0..1) it does.
func (s Segment) CrossPlane(on, normal geom.Vector) (point geom.Vector, t float64, ok bool) {
	d0, d1 := s.From.Sub(on).Dot(normal), s.To.Sub(on).Dot(normal)
	if d0*d1 >= 0 {
		return geom.Vector{}, 0, false
	}
	t = d0 / (d0 - d1)
	return s.From.Add(s.To.Sub(s.From).MulScalar(t)), t, true
}

// Trace follows a ray from origin heading along dir through the field. A path in a
// central field stays in one plane (the one holding the centre, the origin and dir),
// so the bending is integrated as a 1-D orbit equation and each small step is handed to
// visit as world-space points. visit returns false to stop early. Trace returns how the
// path ended and, for an escaped path, the direction it was last heading.
func (f Field) Trace(origin, dir geom.Vector, visit func(Segment) bool) (Outcome, geom.Vector) {
	rel := origin.Sub(f.Center)
	r0 := rel.Length()
	e1 := rel.MulScalar(1 / r0) // outward from the centre at the start
	cosA := dir.Dot(e1)
	perp := dir.Sub(e1.MulScalar(cosA))
	sinA := perp.Length()
	if sinA < 1e-9 { // aimed straight at (or away from) the centre
		if cosA < 0 {
			return Captured, geom.Vector{}
		}
		return Escaped, dir
	}
	e2 := perp.MulScalar(1 / sinA)
	point := func(phi, u float64) geom.Vector {
		return f.Center.Add(e1.MulScalar(math.Cos(phi)).Add(e2.MulScalar(math.Sin(phi))).MulScalar(1 / u))
	}
	maxTurn := f.MaxTurn
	if maxTurn == 0 {
		maxTurn = 3 * math.Pi
	}
	capU, escU := 1/f.Capture, 1/f.Escape
	law := func(u, _ float64) float64 { return f.Law(u) }
	u, v, phi := 1/r0, -(1/r0)*cosA/sinA, 0.0 // v = du/dphi
	if u < escU && v < 0 {                    // started beyond the escape radius, already heading out
		return Escaped, dir
	}
	heading := dir // the direction of the last step, for paths that run off to infinity

	for steps := 0; phi < maxTurn && steps < 3000; steps++ {
		h := 0.03 // take longer steps far from the centre and shorter ones close in
		switch {
		case u < 0.06*capU/0.5:
			h = 0.07
		case u > 0.5*capU:
			h = 0.012
		}
		un, vn := Step2(law, u, v, h)
		phin := phi + h
		if un >= capU {
			return Captured, geom.Vector{}
		}
		if un <= 0 { // one long step ran past infinity
			return Escaped, heading
		}
		from, to := point(phi, u), point(phin, un)
		escaping := un < escU && vn < 0
		if escaping && u > escU { // stop the step exactly at the escape radius
			to = point(phi+h*(u-escU)/(u-un), escU)
		}
		if step := to.Sub(from); step.LengthSquared() > 0 {
			heading = step.Normalize()
		}
		if !visit(Segment{from, to, 1 / u, 1 / un, phi, phin}) {
			return Stopped, geom.Vector{}
		}
		if escaping {
			return Escaped, heading
		}
		u, v, phi = un, vn, phin
	}
	return Stopped, geom.Vector{}
}

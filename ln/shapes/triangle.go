package shapes

import (
	"github.com/loyality7/ln/ln/geom"
)

type Triangle struct {
	V1, V2, V3 geom.Vector
	Box        geom.Box
	ID         int // object tag for renderers that need per-object materials

	// Optional smooth shading: when Smooth is set, N holds a unit normal for
	// each corner and hits interpolate them instead of using the flat face normal.
	N      [3]geom.Vector
	Smooth bool
}

// FaceNormal is the unit normal from the winding order.
func (t *Triangle) FaceNormal() geom.Vector {
	return t.V2.Sub(t.V1).Cross(t.V3.Sub(t.V1)).Normalize()
}

// Tag identifies the object this triangle belongs to (0 = untagged).
func (t *Triangle) Tag() int { return t.ID }

func NewTriangle(v1, v2, v3 geom.Vector) *Triangle {
	t := Triangle{}
	t.V1 = v1
	t.V2 = v2
	t.V3 = v3
	t.UpdateBoundingBox()
	return &t
}

func (t *Triangle) UpdateBoundingBox() {
	min := t.V1.Min(t.V2).Min(t.V3)
	max := t.V1.Max(t.V2).Max(t.V3)
	t.Box = geom.Box{min, max}
}

func (t *Triangle) Compile() {
}

func (t *Triangle) BoundingBox() geom.Box {
	return t.Box
}

func (t *Triangle) Contains(v geom.Vector, f float64) bool {
	return false
}

func (t *Triangle) Intersect(r geom.Ray) Hit {
	e1x := t.V2.X - t.V1.X
	e1y := t.V2.Y - t.V1.Y
	e1z := t.V2.Z - t.V1.Z
	e2x := t.V3.X - t.V1.X
	e2y := t.V3.Y - t.V1.Y
	e2z := t.V3.Z - t.V1.Z
	px := r.Direction.Y*e2z - r.Direction.Z*e2y
	py := r.Direction.Z*e2x - r.Direction.X*e2z
	pz := r.Direction.X*e2y - r.Direction.Y*e2x
	det := e1x*px + e1y*py + e1z*pz
	if det > -geom.EPS && det < geom.EPS {
		return NoHit
	}
	inv := 1 / det
	tx := r.Origin.X - t.V1.X
	ty := r.Origin.Y - t.V1.Y
	tz := r.Origin.Z - t.V1.Z
	u := (tx*px + ty*py + tz*pz) * inv
	if u < 0 || u > 1 {
		return NoHit
	}
	qx := ty*e1z - tz*e1y
	qy := tz*e1x - tx*e1z
	qz := tx*e1y - ty*e1x
	v := (r.Direction.X*qx + r.Direction.Y*qy + r.Direction.Z*qz) * inv
	if v < 0 || u+v > 1 {
		return NoHit
	}
	d := (e2x*qx + e2y*qy + e2z*qz) * inv
	if d < geom.EPS {
		return NoHit
	}
	n := t.FaceNormal()
	if t.Smooth {
		n = t.N[0].MulScalar(1 - u - v).Add(t.N[1].MulScalar(u)).Add(t.N[2].MulScalar(v)).Normalize()
	}
	return Hit{t, d, n}
}

func (t *Triangle) Paths() geom.Paths {
	return geom.Paths{
		{t.V1, t.V2},
		{t.V2, t.V3},
		{t.V3, t.V1},
	}
}

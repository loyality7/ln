package shapes

import (
	"github.com/loyality7/ln/ln/geom"
)

type Shape interface {
	Compile()
	BoundingBox() geom.Box
	Contains(geom.Vector, float64) bool
	Intersect(geom.Ray) Hit
	Paths() geom.Paths
}

type EmptyShape struct {
}

func (s *EmptyShape) Compile() {
}

func (s *EmptyShape) BoundingBox() geom.Box {
	return geom.Box{geom.Vector{}, geom.Vector{}}
}

func (s *EmptyShape) Contains(v geom.Vector, f float64) bool {
	return false
}

func (s *EmptyShape) Intersect(r geom.Ray) Hit {
	return NoHit
}

func (s *EmptyShape) Paths() geom.Paths {
	return nil
}

type TransformedShape struct {
	Shape
	Matrix  geom.Matrix
	Inverse geom.Matrix
	normal  geom.Matrix // inverse transpose, for carrying normals to world space
}

func NewTransformedShape(s Shape, m geom.Matrix) Shape {
	inv := m.Inverse()
	return &TransformedShape{s, m, inv, inv.Transpose()}
}

func (s *TransformedShape) BoundingBox() geom.Box {
	return s.Matrix.MulBox(s.Shape.BoundingBox())
}

func (s *TransformedShape) Contains(v geom.Vector, f float64) bool {
	return s.Shape.Contains(s.Inverse.MulPosition(v), f)
}

// Intersect casts in the shape's own space, then reports the hit in world space
// (distance along the world ray, world normal) so scaling does not distort either.
func (s *TransformedShape) Intersect(r geom.Ray) Hit {
	local := s.Inverse.MulRay(r)
	h := s.Shape.Intersect(local)
	if !h.Ok() {
		return h
	}
	world := s.Matrix.MulPosition(local.Position(h.T))
	h.T = world.Distance(r.Origin)
	if h.Normal != (geom.Vector{}) {
		h.Normal = s.normal.MulDirection(h.Normal)
	}
	return h
}

func (s *TransformedShape) Paths() geom.Paths {
	return s.Shape.Paths().Transform(s.Matrix)
}

package scene

import (
	"github.com/loyality7/ln/ln/accel"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/physics"
	"github.com/loyality7/ln/ln/shapes"
)

type Scene struct {
	Shapes []shapes.Shape
	Tree   *accel.Tree
	// Bend, if set, curves every camera ray around a central mass or field, so the
	// scene is seen through gravitational lensing. Nil means straight rays.
	Bend *physics.Field
}

func (s *Scene) Compile() {
	for _, shape := range s.Shapes {
		shape.Compile()
	}
	if s.Tree == nil {
		s.Tree = accel.NewTree(s.Shapes)
	}
}

func (s *Scene) Add(shape shapes.Shape) {
	s.Shapes = append(s.Shapes, shape)
}

func (s *Scene) Intersect(r geom.Ray) shapes.Hit {
	return s.Tree.Intersect(r)
}

func (s *Scene) Visible(eye, point geom.Vector) bool {
	v := eye.Sub(point)
	r := geom.Ray{point, v.Normalize()}
	hit := s.Intersect(r)
	return hit.T >= v.Length()
}

func (s *Scene) Paths() geom.Paths {
	var result geom.Paths
	for _, shape := range s.Shapes {
		result = append(result, shape.Paths()...)
	}
	return result
}

func (s *Scene) Render(eye, center, up geom.Vector, width, height, fovy, near, far, step float64) geom.Paths {
	aspect := width / height
	matrix := geom.LookAt(eye, center, up)
	matrix = matrix.Perspective(fovy, aspect, near, far)
	return s.RenderWithMatrix(matrix, eye, width, height, step)
}

func (s *Scene) RenderWithMatrix(matrix geom.Matrix, eye geom.Vector, width, height, step float64) geom.Paths {
	return s.render(matrix, eye, width, height, step, false)
}

// RenderDepth is Render, but each output point's Z holds its distance from the eye.
func (s *Scene) RenderDepth(eye, center, up geom.Vector, width, height, fovy, near, far, step float64) geom.Paths {
	matrix := geom.LookAt(eye, center, up).Perspective(fovy, width/height, near, far)
	return s.render(matrix, eye, width, height, step, true)
}

func (s *Scene) render(matrix geom.Matrix, eye geom.Vector, width, height, step float64, depth bool) geom.Paths {
	s.Compile()
	paths := s.Paths()
	if step > 0 {
		paths = paths.Chop(step)
	}
	paths = paths.Filter(&ClipFilter{Matrix: matrix, Eye: eye, Scene: s, Depth: depth})
	if step > 0 {
		paths = paths.Simplify(1e-6)
	}
	zs := 0.0
	if depth {
		zs = 1
	}
	matrix = geom.Translate(geom.Vector{X: 1, Y: 1}).Scale(geom.Vector{X: width / 2, Y: height / 2, Z: zs})
	return paths.Transform(matrix)
}

// March follows a camera ray through the scene, calling visit for each surface it
// crosses, nearest first: the hit (with T the distance travelled along the possibly
// curved path), the point, and the unit direction the ray was heading there. visit
// returns false to stop. It returns how the ray ended and, if it escaped a Bend field,
// the direction it left along. Without Bend the ray is straight and visit is called
// at most once.
func (s *Scene) March(r geom.Ray, visit func(h shapes.Hit, at, heading geom.Vector) bool) (physics.Outcome, geom.Vector) {
	if s.Bend == nil {
		if h := s.Intersect(r); h.Ok() {
			visit(h, r.Position(h.T), r.Direction)
		}
		return physics.Escaped, r.Direction
	}
	box := s.Tree.Box
	travelled := 0.0
	return s.Bend.Trace(r.Origin, r.Direction, func(seg physics.Segment) bool {
		step := seg.To.Sub(seg.From)
		length := step.Length()
		if length == 0 {
			return true
		}
		dir := step.MulScalar(1 / length)
		ray := geom.Ray{Origin: seg.From, Direction: dir}
		if t1, t2 := box.Intersect(ray); !(t2 < t1 || t2 <= 0 || t1 > length+1e-3) { // written so a NaN (axis-parallel ray) is not rejected
			if h := s.Tree.Intersect(ray); h.Ok() && h.T <= length+1e-3 { // shapes ignore hits closer than 1e-3, so a crossing at a joint belongs to the earlier segment
				h.T += travelled
				if !visit(h, ray.Position(h.T-travelled), dir) {
					travelled += length
					return false
				}
			}
		}
		travelled += length
		return true
	})
}

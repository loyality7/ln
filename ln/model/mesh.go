package model

import (
	"math"

	"github.com/loyality7/ln/ln/accel"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/shapes"
)

type Mesh struct {
	Box       geom.Box
	Triangles []*shapes.Triangle
	Tree      *accel.Tree
}

func NewMesh(triangles []*shapes.Triangle) *Mesh {
	box := shapes.BoxForTriangles(triangles)
	return &Mesh{box, triangles, nil}
}

func (m *Mesh) Compile() {
	if m.Tree == nil {
		shapes := make([]shapes.Shape, len(m.Triangles))
		for i, triangle := range m.Triangles {
			shapes[i] = triangle
		}
		m.Tree = accel.NewTree(shapes)
	}
}

func (m *Mesh) BoundingBox() geom.Box {
	return m.Box
}

func (m *Mesh) Contains(v geom.Vector, f float64) bool {
	return false
}

func (m *Mesh) Intersect(r geom.Ray) shapes.Hit {
	return m.Tree.Intersect(r)
}

func (m *Mesh) Paths() geom.Paths {
	var result geom.Paths
	for _, t := range m.Triangles {
		result = append(result, t.Paths()...)
	}
	return result
}

func (m *Mesh) UpdateBoundingBox() {
	m.Box = shapes.BoxForTriangles(m.Triangles)
}

func (m *Mesh) UnitCube() {
	m.FitInside(geom.Box{geom.Vector{}, geom.Vector{1, 1, 1}}, geom.Vector{})
	m.MoveTo(geom.Vector{}, geom.Vector{0.5, 0.5, 0.5})
}

func (m *Mesh) MoveTo(position, anchor geom.Vector) {
	matrix := geom.Translate(position.Sub(m.Box.Anchor(anchor)))
	m.Transform(matrix)
}

func (m *Mesh) FitInside(box geom.Box, anchor geom.Vector) {
	scale := box.Size().Div(m.BoundingBox().Size()).MinComponent()
	extra := box.Size().Sub(m.BoundingBox().Size().MulScalar(scale))
	matrix := geom.Identity()
	matrix = matrix.Translate(m.BoundingBox().Min.MulScalar(-1))
	matrix = matrix.Scale(geom.Vector{scale, scale, scale})
	matrix = matrix.Translate(box.Min.Add(extra.Mul(anchor)))
	m.Transform(matrix)
}

func (m *Mesh) Transform(matrix geom.Matrix) {
	for _, t := range m.Triangles {
		t.V1 = matrix.MulPosition(t.V1)
		t.V2 = matrix.MulPosition(t.V2)
		t.V3 = matrix.MulPosition(t.V3)
		t.UpdateBoundingBox()
	}
	m.UpdateBoundingBox()
	m.Tree = nil // dirty
}

func (m *Mesh) SaveBinarySTL(path string) error {
	return SaveBinarySTL(path, m)
}

func (m *Mesh) Voxelize(size float64) []*shapes.Cube {
	z1 := m.Box.Min.Z
	z2 := m.Box.Max.Z
	set := make(map[geom.Vector]bool)
	for z := z1; z <= z2; z += size {
		plane := shapes.Plane{geom.Vector{0, 0, z}, geom.Vector{0, 0, 1}}
		paths := IntersectPlane(&plane, m)
		for _, path := range paths {
			for _, v := range path {
				x := math.Floor(v.X/size+0.5) * size
				y := math.Floor(v.Y/size+0.5) * size
				z := math.Floor(v.Z/size+0.5) * size
				set[geom.Vector{x, y, z}] = true
			}
		}
	}
	var result []*shapes.Cube
	for v, _ := range set {
		cube := shapes.NewCube(v.SubScalar(size/2), v.AddScalar(size/2))
		result = append(result, cube)
	}
	return result
}

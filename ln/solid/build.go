// Package solid is the shape forge: it builds any 3D shape from code (primitives,
// parametric surfaces, lathes, extrusions, tubes) as triangle meshes that
// the renderer can ray-cast, shade and outline.
package solid

import (
	"math"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/model"
	"github.com/loyality7/ln/ln/shapes"
)

// Pt is a 2D point, used for lathe profiles and extrusion outlines.
type Pt struct{ X, Y float64 }

func tri(a, b, c geom.Vector) *shapes.Triangle {
	if b.Sub(a).Cross(c.Sub(a)).LengthSquared() < 1e-18 {
		return nil // degenerate (e.g. at a pole)
	}
	return shapes.NewTriangle(a, b, c)
}

type builder struct{ tris []*shapes.Triangle }

func (b *builder) add(a, c, d geom.Vector) {
	if t := tri(a, c, d); t != nil {
		b.tris = append(b.tris, t)
	}
}

func (b *builder) mesh() *model.Mesh { return Shade(model.NewMesh(b.tris), 50) }

// Surface samples f over the unit square on an nu x nv grid. It builds an open
// sheet; if f is periodic the seam closes by itself. Call Orient for closed solids.
func Surface(nu, nv int, f func(u, v float64) geom.Vector) *model.Mesh {
	p := make([][]geom.Vector, nu+1)
	for i := range p {
		p[i] = make([]geom.Vector, nv+1)
		for j := range p[i] {
			p[i][j] = f(float64(i)/float64(nu), float64(j)/float64(nv))
		}
	}
	var b builder
	for i := 0; i < nu; i++ {
		for j := 0; j < nv; j++ {
			b.add(p[i][j], p[i+1][j], p[i+1][j+1])
			b.add(p[i][j], p[i+1][j+1], p[i][j+1])
		}
	}
	return b.mesh()
}

// Heightfield is the surface z = f(x,y) over [-sx/2,sx/2] x [-sy/2,sy/2].
func Heightfield(sx, sy float64, nx, ny int, f func(x, y float64) float64) *model.Mesh {
	return Surface(nx, ny, func(u, v float64) geom.Vector {
		x, y := (u-0.5)*sx, (v-0.5)*sy
		return geom.V(x, y, f(x, y))
	})
}

// Orient makes triangle winding point outward for a closed mesh (the
// renderer's silhouette detection relies on it).
func Orient(m *model.Mesh) *model.Mesh {
	var vol float64
	for _, t := range m.Triangles {
		vol += t.V1.Dot(t.V2.Cross(t.V3))
	}
	if vol < 0 {
		for _, t := range m.Triangles {
			t.V2, t.V3 = t.V3, t.V2
			if t.Smooth { // keep the shading normals outward too
				t.N[1], t.N[2] = t.N[2], t.N[1]
				for k := range t.N {
					t.N[k] = t.N[k].MulScalar(-1)
				}
			}
		}
		m.Tree = nil
	}
	return m
}

// Merge combines meshes into one.
func Merge(ms ...*model.Mesh) *model.Mesh {
	var all []*shapes.Triangle
	for _, m := range ms {
		all = append(all, m.Triangles...)
	}
	return model.NewMesh(all)
}

// Place applies matrix to the mesh in place and returns it.
func Place(m *model.Mesh, matrix geom.Matrix) *model.Mesh {
	m.Transform(matrix)
	return m
}

// ---- primitives ----

func Box(sx, sy, sz float64) *model.Mesh {
	x, y, z := sx/2, sy/2, sz/2
	c := [8]geom.Vector{geom.V(-x, -y, -z), geom.V(x, -y, -z), geom.V(x, y, -z), geom.V(-x, y, -z), geom.V(-x, -y, z), geom.V(x, -y, z), geom.V(x, y, z), geom.V(-x, y, z)}
	var b builder
	for _, q := range [6][4]int{{0, 3, 2, 1}, {4, 5, 6, 7}, {0, 1, 5, 4}, {2, 3, 7, 6}, {1, 2, 6, 5}, {3, 0, 4, 7}} {
		b.add(c[q[0]], c[q[1]], c[q[2]])
		b.add(c[q[0]], c[q[2]], c[q[3]])
	}
	return Orient(b.mesh())
}

func Sphere(r float64) *model.Mesh { return Ellipsoid(r, r, r) }

func Ellipsoid(rx, ry, rz float64) *model.Mesh {
	return Orient(Surface(48, 24, func(u, v float64) geom.Vector {
		th, ph := 2*math.Pi*u, math.Pi*v
		return geom.V(rx*math.Sin(ph)*math.Cos(th), ry*math.Sin(ph)*math.Sin(th), rz*math.Cos(ph))
	}))
}

// Torus has ring radius R and tube radius r, lying in the xy plane.
func Torus(R, r float64) *model.Mesh {
	return Orient(Surface(48, 24, func(u, v float64) geom.Vector {
		th, ph := 2*math.Pi*u, 2*math.Pi*v
		return geom.V((R+r*math.Cos(ph))*math.Cos(th), (R+r*math.Cos(ph))*math.Sin(th), r*math.Sin(ph))
	}))
}

// Cylinder stands on z=0 with a flat top and bottom.
func Cylinder(r, h float64) *model.Mesh {
	return Lathe([]Pt{{0, 0}, {r, 0}, {r, h}, {0, h}}, 48)
}

// Cone has its base on z=0 and its tip at z=h.
func Cone(r, h float64) *model.Mesh { return Lathe([]Pt{{0, 0}, {r, 0}, {0, h}}, 48) }

// Capsule is a cylinder of length l (between sphere centres) with round ends.
func Capsule(r, l float64) *model.Mesh {
	var p []Pt
	const n = 12
	for i := 0; i <= n; i++ { // bottom hemisphere, pole to equator
		a := math.Pi/2*float64(i)/n - math.Pi/2
		p = append(p, Pt{r * math.Cos(a), r + r*math.Sin(a)})
	}
	for i := 0; i <= n; i++ { // top hemisphere
		a := math.Pi / 2 * float64(i) / n
		p = append(p, Pt{r * math.Cos(a), r + l + r*math.Sin(a)})
	}
	return Lathe(p, 32)
}

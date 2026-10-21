package solid

import (
	"math"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/model"
)

type geomVec = geom.Vector

// Shade gives m smooth vertex normals: at each welded vertex, the normals of
// neighbouring faces within creaseDegrees of a triangle's own normal are
// averaged (area-weighted), so curved surfaces shade smoothly and real edges
// stay sharp.
func Shade(m *model.Mesh, creaseDegrees float64) *model.Mesh {
	type corner struct{ tri, k int }
	at := map[vkey][]corner{}
	face := make([]struct{ n, area geomVec }, len(m.Triangles))
	for i, t := range m.Triangles {
		c := t.V2.Sub(t.V1).Cross(t.V3.Sub(t.V1))
		face[i].area = c
		face[i].n = c.Normalize()
		for k, v := range [3]geomVec{t.V1, t.V2, t.V3} {
			at[keyOf(v)] = append(at[keyOf(v)], corner{i, k})
		}
	}
	cos := math.Cos(creaseDegrees * math.Pi / 180)
	for i, t := range m.Triangles {
		for k, v := range [3]geomVec{t.V1, t.V2, t.V3} {
			var sum geomVec
			for _, o := range at[keyOf(v)] {
				if face[i].n.Dot(face[o.tri].n) >= cos {
					sum = sum.Add(face[o.tri].area)
				}
			}
			if sum.LengthSquared() == 0 {
				sum = face[i].n
			}
			t.N[k] = sum.Normalize()
		}
		t.Smooth = true
	}
	return m
}

package solid

import (
	"math"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/model"
)

type edge struct {
	a, b   geom.Vector
	t1, t2 int // adjacent triangles; t2 = -1 on an open boundary
}

// Outlined is a mesh that draws only its meaningful lines: open borders,
// creases sharper than Crease degrees, and the silhouette as seen from Eye.
// Set Eye (in the mesh's own space) before rendering each frame.
type Outlined struct {
	*model.Mesh
	Eye    geom.Vector
	Crease float64
	edges  []edge
	norms  []geom.Vector
}

type vkey [3]int64

func keyOf(v geom.Vector) vkey {
	const s = 1e5
	return vkey{int64(math.Round(v.X * s)), int64(math.Round(v.Y * s)), int64(math.Round(v.Z * s))}
}

// Outline wraps m, welding shared vertices to find which triangles meet at each edge.
func Outline(m *model.Mesh, creaseDegrees float64) *Outlined {
	o := &Outlined{Mesh: m, Crease: creaseDegrees}
	type ek [2]vkey
	seen := map[ek]int{} // edge -> index in o.edges
	o.norms = make([]geom.Vector, len(m.Triangles))
	for i, t := range m.Triangles {
		o.norms[i] = t.V2.Sub(t.V1).Cross(t.V3.Sub(t.V1)).Normalize()
		vs := [3]geom.Vector{t.V1, t.V2, t.V3}
		for k := 0; k < 3; k++ {
			a, b := vs[k], vs[(k+1)%3]
			ka, kb := keyOf(a), keyOf(b)
			if ka == kb {
				continue
			}
			key := ek{ka, kb}
			if lessKey(kb, ka) {
				key = ek{kb, ka}
			}
			if j, ok := seen[key]; ok {
				o.edges[j].t2 = i
			} else {
				seen[key] = len(o.edges)
				o.edges = append(o.edges, edge{a, b, i, -1})
			}
		}
	}
	return o
}

func lessKey(a, b vkey) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

// Creases returns only open borders and sharp creases (no silhouette), for
// renderers that draw silhouettes themselves.
func (o *Outlined) Creases() geom.Paths {
	cos := math.Cos(geom.Radians(o.Crease))
	var out geom.Paths
	for _, e := range o.edges {
		if e.t2 < 0 || o.norms[e.t1].Dot(o.norms[e.t2]) < cos {
			out = append(out, geom.Path{e.a, e.b})
		}
	}
	return out
}

// Paths draws the outline as seen from o.Eye.
func (o *Outlined) Paths() geom.Paths { return o.PathsFrom(o.Eye) }

// PathsFrom is Paths for an explicit eye, so one Outlined can be shared by
// many placed copies.
func (o *Outlined) PathsFrom(eye geom.Vector) geom.Paths {
	cos := math.Cos(geom.Radians(o.Crease))
	var out geom.Paths
	for _, e := range o.edges {
		draw := e.t2 < 0 || o.norms[e.t1].Dot(o.norms[e.t2]) < cos
		if !draw {
			mid := e.a.Add(e.b).MulScalar(0.5)
			view := eye.Sub(mid)
			draw = (o.norms[e.t1].Dot(view) > 0) != (o.norms[e.t2].Dot(view) > 0)
		}
		if draw {
			out = append(out, geom.Path{e.a, e.b})
		}
	}
	return out
}

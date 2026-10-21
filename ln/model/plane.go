package model

import (
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/shapes"
)

// IntersectPlane returns the segments where plane p cuts mesh m.
func IntersectPlane(p *shapes.Plane, m *Mesh) geom.Paths {
	var result geom.Paths
	for _, t := range m.Triangles {
		if v1, v2, ok := p.IntersectTriangle(t); ok {
			result = append(result, geom.Path{v1, v2})
		}
	}
	return result
}

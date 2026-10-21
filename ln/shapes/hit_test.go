package shapes

import (
	"math"
	"testing"

	"github.com/loyality7/ln/ln/geom"
)

func v(x, y, z float64) geom.Vector { return geom.Vector{X: x, Y: y, Z: z} }

func TestHitNormals(t *testing.T) {
	cases := []struct {
		name string
		s    Shape
		ray  geom.Ray
		want geom.Vector
	}{
		{"sphere", NewSphere(v(0, 0, 0), 1), geom.Ray{Origin: v(0, -5, 0), Direction: v(0, 1, 0)}, v(0, -1, 0)},
		{"cube", NewCube(v(-1, -1, -1), v(1, 1, 1)), geom.Ray{Origin: v(5, 0.2, 0.1), Direction: v(-1, 0, 0)}, v(1, 0, 0)},
		{"cylinder", NewCylinder(1, 0, 2), geom.Ray{Origin: v(0, -5, 1), Direction: v(0, 1, 0)}, v(0, -1, 0)},
		{"triangle", NewTriangle(v(0, 0, 0), v(1, 0, 0), v(0, 1, 0)), geom.Ray{Origin: v(0.2, 0.2, 3), Direction: v(0, 0, -1)}, v(0, 0, 1)},
	}
	for _, c := range cases {
		h := c.s.Intersect(c.ray)
		if !h.Ok() || h.Normal.Distance(c.want) > 1e-9 {
			t.Errorf("%s: hit=%v normal=%v want %v", c.name, h.Ok(), h.Normal, c.want)
		}
	}
}

func TestTransformedShapeReportsWorldDistanceAndNormal(t *testing.T) {
	// unit sphere stretched 4x along y: ray along +y from y=-10 hits at y=-4, normal -y
	m := geom.Scale(v(1, 4, 1))
	s := NewTransformedShape(NewSphere(v(0, 0, 0), 1), m)
	h := s.Intersect(geom.Ray{Origin: v(0, -10, 0), Direction: v(0, 1, 0)})
	if !h.Ok() || math.Abs(h.T-6) > 1e-6 || h.Normal.Distance(v(0, -1, 0)) > 1e-6 {
		t.Fatalf("T=%v normal=%v", h.T, h.Normal)
	}
}

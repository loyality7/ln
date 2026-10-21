package solid

import (
	"math"
	"testing"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/model"
)

func volume(m *model.Mesh) float64 {
	var v float64
	for _, t := range m.Triangles {
		v += t.V1.Dot(t.V2.Cross(t.V3)) / 6
	}
	return v
}

func TestClosedSolidsHavePositiveVolume(t *testing.T) {
	cases := map[string]struct {
		m    *model.Mesh
		want float64
	}{
		"box":      {Box(2, 3, 4), 24},
		"sphere":   {Sphere(1), 4.0 / 3 * math.Pi},
		"cylinder": {Cylinder(1, 2), 2 * math.Pi},
		"cone":     {Cone(1, 3), math.Pi},
		"torus":    {Torus(2, 0.5), 2 * math.Pi * math.Pi * 2 * 0.25},
	}
	for name, c := range cases {
		if got := volume(c.m); math.Abs(got-c.want)/c.want > 0.02 {
			t.Errorf("%s: volume %.3f want %.3f", name, got, c.want)
		}
	}
}

func TestExtrudeConcavePolygon(t *testing.T) {
	l := []Pt{{0, 0}, {2, 0}, {2, 1}, {1, 1}, {1, 2}, {0, 2}} // L shape, area 3
	if got := volume(Extrude(l, 2)); math.Abs(got-6) > 1e-6 {
		t.Fatalf("volume %v want 6", got)
	}
}

func TestOutlineBoxDrawsOnlyEdges(t *testing.T) {
	o := Outline(Box(1, 1, 1), 35)
	o.Eye = geom.Vector{X: 5, Y: 6, Z: 7}
	// all 12 cube edges are creases; the 6 face diagonals must not appear
	if n := len(o.Paths()); n != 12 {
		t.Fatalf("paths=%d want 12", n)
	}
}

func TestOutlineSphereSilhouette(t *testing.T) {
	o := Outline(Sphere(1), 35)
	o.Eye = geom.Vector{Z: 10}
	if len(o.Paths()) == 0 {
		t.Fatal("no silhouette")
	}
}

package scene_test

import (
	"testing"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/physics"
	"github.com/loyality7/ln/ln/scene"
	"github.com/loyality7/ln/ln/shapes"
)

func lensedDisk() *scene.Scene {
	g := physics.Gravity{M: 1}
	s := &scene.Scene{Bend: &physics.Field{Law: g.Light(), Capture: g.SchwarzschildRadius(), Escape: 80}}
	s.Add(shapes.NewAnnulus(geom.Vector{}, 6, 30))
	s.Compile()
	return s
}

func TestMarchStraightSeesTheDiskOnce(t *testing.T) {
	s := &scene.Scene{}
	s.Add(shapes.NewAnnulus(geom.Vector{}, 6, 30))
	s.Compile()
	n := 0
	s.March(geom.Ray{Origin: geom.V(10, 0, 20), Direction: geom.V(0, 0, -1)}, func(h shapes.Hit, at, heading geom.Vector) bool {
		n++
		if at.Distance(geom.V(10, 0, 0)) > 1e-6 || h.T < 19.99 || h.T > 20.01 {
			t.Fatalf("hit at %v after %v", at, h.T)
		}
		return true
	})
	if n != 1 {
		t.Fatalf("%d hits", n)
	}
}

func TestMarchAimedAtTheMassIsCaptured(t *testing.T) {
	out, _ := lensedDisk().March(geom.Ray{Origin: geom.V(200, 0, 0), Direction: geom.V(-1, 0, 0)}, func(shapes.Hit, geom.Vector, geom.Vector) bool { return true })
	if out != physics.Captured {
		t.Fatalf("outcome %v", out)
	}
}

func TestMarchBendsLightOverTheTopOfTheShadow(t *testing.T) {
	s := lensedDisk()
	cam := geom.V(200, 0, 15) // near edge-on
	// aimed just above the shadow: gravity bends it down onto the far side of the disk
	d := geom.V(-1, 0, 0).Add(geom.V(0, 0, -15.0/200)).Add(geom.V(0, 0, 8.0/200)).Normalize()
	var hits []geom.Vector
	s.March(geom.Ray{Origin: cam, Direction: d}, func(h shapes.Hit, at, heading geom.Vector) bool {
		hits = append(hits, at)
		return true
	})
	if len(hits) == 0 {
		t.Fatal("expected the bent ray to meet the disk")
	}
}

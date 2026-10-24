package blackhole

import (
	"testing"

	"github.com/loyality7/ln/ln/buffer"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/light"
	"github.com/loyality7/ln/ln/physics"
	"github.com/loyality7/ln/ln/scene"
	"github.com/loyality7/ln/ln/shapes"
)

func TestShadowIsVoidAndTheDiskShowsTwice(t *testing.T) {
	grav := physics.Gravity{M: 1}
	sc := &scene.Scene{Bend: &physics.Field{Law: grav.Light(), Capture: grav.SchwarzschildRadius(), Escape: 80}}
	sc.Add(shapes.NewAnnulus(geom.Vector{}, 5, 30))
	cam := buffer.Camera{Eye: geom.V(200, 0, 15), Up: geom.V(0, 0, 1), Fovy: 4, Layers: 2}
	g := buffer.Gather(sc, cam, 64, 64, light.Rig{})
	if !g.IsVoid(32*64 + 32) {
		t.Fatal("the pixel at the centre of the shadow must be void")
	}
	twice := 0
	for _, c := range g.Crosses {
		if len(c) == 2 {
			twice++
		}
	}
	if twice == 0 {
		t.Fatal("some rays must meet the disk twice: directly and bent over the shadow")
	}
}

func TestDrawProducesInk(t *testing.T) {
	p := Default(96, 160)
	dr := Draw(p)
	var lines, fills int
	for _, l := range dr.Layers {
		lines += len(l.Lines)
		fills += len(l.Fills)
	}
	if lines < 20 || fills < 2 {
		t.Fatalf("expected rings and stars, got %d lines %d fills", lines, fills)
	}
	if dr.Image(96, 160).Bounds().Dx() != 96 {
		t.Fatal("image size")
	}
}

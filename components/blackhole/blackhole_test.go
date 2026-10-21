package blackhole

import (
	"testing"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/physics"
)

func TestShadowIsDarkAndFarSideOfDiskBendsOverIt(t *testing.T) {
	p := Default(120, 200)
	grav := physics.Gravity{M: 1}
	field := physics.Field{Law: grav.Light(), Capture: grav.SchwarzschildRadius(), Escape: 80}
	cam := geom.V(p.Distance, 0, p.Distance*0.075)
	f := cam.Normalize().MulScalar(-1)
	// the ray through the very middle of the picture falls into the hole: no star, no disk
	s := p.trace(field, cam, f, 0.001)
	if s.star != 0 || s.disk[0].ok {
		t.Fatalf("the shadow must be empty, got %+v", s)
	}
	// a ray aimed just above the shadow, seen near edge-on, meets the far side of the disk
	// twice: once on the near face and once bent over the top
	up := geom.V(0, 0, 1)
	crit := grav.CriticalImpactParameter() / p.Distance
	d := f.Add(up.MulScalar(crit * 1.6)).Normalize()
	if s := p.trace(field, cam, d, 0.001); !s.disk[0].ok {
		t.Fatalf("a ray skimming above the shadow should hit the disk, got %+v", s)
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

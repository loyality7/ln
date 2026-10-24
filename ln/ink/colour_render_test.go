package ink

import (
	"testing"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/scene"
	"github.com/loyality7/ln/ln/shapes"
)

func TestRenderColourPaintsObjectsInTheirColour(t *testing.T) {
	cube := shapes.NewCube(geom.V(-1, -1, -1), geom.V(1, 1, 1))
	s := &scene.Scene{}
	s.Add(cube)
	s.Compile()
	// the scene tags objects through triangles; here a cube has no tag, so the colour is looked up by tag 0
	mats := map[int]Material{0: {Color: RGB(0.9, 0.1, 0.1), Line: true, LineW: 1}}
	st := ToonStyle()
	d := RenderColour(s, geom.V(5, -6, 4), geom.Vector{}, geom.V(0, 0, 1), 240, 240, 40, st, mats, nil)
	img := d.Image(0, 0)
	r, g, b, _ := img.At(120, 120).RGBA()
	if !(r>>8 > 120 && g>>8 < 120 && b>>8 < 120) {
		t.Fatalf("centre pixel %d %d %d is not the painted red", r>>8, g>>8, b>>8)
	}
	if corner, _, _, _ := img.At(2, 2).RGBA(); corner>>8 < 240 {
		t.Fatal("the background must stay paper")
	}
}

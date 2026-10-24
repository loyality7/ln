package ink

import (
	"strings"
	"testing"

	"github.com/loyality7/ln/ln/geom"
)

func TestColoursReachThePixelsAndTheSVG(t *testing.T) {
	loop := geom.Path{{X: 2, Y: 2}, {X: 18, Y: 2}, {X: 18, Y: 18}, {X: 2, Y: 18}}
	red := RGB(1, 0, 0)
	dr := NewDrawing(20, 20, Layer{Fills: []Fill{{Loops: []geom.Path{loop}, Color: red}}})
	dr.Paper = RGB(0, 0, 1)
	r, g, b, _ := dr.Image(0, 0).At(10, 10).RGBA()
	if r>>8 != 255 || g != 0 || b != 0 {
		t.Fatalf("fill pixel is %d %d %d, want red", r>>8, g>>8, b>>8)
	}
	if _, _, b, _ := dr.Image(0, 0).At(0, 0).RGBA(); b>>8 != 255 {
		t.Fatal("paper colour not applied")
	}
	if svg := dr.SVG(); !strings.Contains(svg, "rgb(255,0,0)") || !strings.Contains(svg, "rgb(0,0,255)") {
		t.Fatalf("svg missing colours:\n%s", svg)
	}
	if c := RGB(0, 0, 0).Mix(RGB(1, 1, 1), 0.5); c.R != 0.5 || c.A != 1 {
		t.Fatalf("mix %v", c)
	}
}

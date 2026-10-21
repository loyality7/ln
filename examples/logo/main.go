// Command logo draws the project's logo with the engine itself: the primitives
// (cube, sphere, cone, torus) are rendered by the ink renderer with real hatching
// and cast shadows, and the hand-inked "ln" wordmark and tagline are marks added on
// top. It writes assets/logo.svg and assets/logo.png.
//
//	go run ./examples/logo
package main

import (
	"math"
	"os"

	"github.com/fogleman/gg"
	"github.com/loyality7/ln/ln/fx"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/ink"
	"github.com/loyality7/ln/ln/light"
	"github.com/loyality7/ln/ln/stage"
)

const w, h = 960.0, 540.0

func main() {
	st := stage.New(w, h, 24)
	style, _ := ink.Preset("ink")
	style.Outline, style.Hatch, style.Wobble, style.Scale = 3.4, 1.15, 0.5, 0.9
	st.Look = &stage.InkLook{Style: style}
	st.Atmosphere = stage.Studio{Lights: light.Rig{
		Lights:  []light.Light{light.Directional{Dir: geom.V(-0.55, -0.7, 0.75), Power: 1}},
		Ambient: 0.25, Shadows: true,
	}}

	// the primitives, standing on a floor
	st.Root.Box(60, 60, 0.2).At(0, 0, -0.1).NoOutline()
	st.Root.Box(1.5, 1.5, 1.5).At(0, 0, 0.75).Turn(0, 0, 0.5)
	st.Root.Sphere(0.85).At(1.9, -0.9, 0.85)
	st.Root.Cone(0.85, 2.0).At(-1.6, 0.9, 0)
	st.Root.Torus(0.75, 0.24).At(0.2, 2.1, 0.5).Turn(1.1, 0, 0.6)
	st.Cam.Eye.At(0.9, -9.2, 4.6)
	st.Cam.Target.At(-1.7, 0, 0.9) // look left of the objects so they sit on the right of the page
	st.Cam.Fovy = 38

	dr := st.Frame(0).(*ink.Drawing)

	// the wordmark: two brush strokes, thin at the ends and heavy in the middle
	brush := func(width float64, p ...geom.Vector) ink.Fill {
		return ink.Fill{Loops: []geom.Path{ink.Ribbon(ink.Curve(geom.Path(p), 10), func(t float64) float64 {
			return width * (0.4 + 0.6*math.Pow(math.Sin(math.Pi*t), 0.55))
		})}}
	}
	v := func(x, y float64) geom.Vector { return geom.Vector{X: x, Y: y} }
	marks := ink.Layer{Fills: []ink.Fill{
		brush(40, v(92, 150), v(95, 270), v(100, 380), v(104, 456)),                             // l
		brush(36, v(190, 150), v(192, 240), v(194, 326)),                                        // n: stem
		brush(36, v(194, 296), v(222, 330), v(282, 338), v(338, 304), v(360, 232), v(361, 150)), // n: arch
		{Loops: fx.PixelText("INK ANIMATION FROM CODE", 52, 78, 3.3)},                           // tagline
	}}
	dr.Layers = append(dr.Layers, marks)

	os.MkdirAll("assets", 0o755)
	if err := os.WriteFile("assets/logo.svg", []byte(dr.SVG()), 0o644); err != nil {
		panic(err)
	}
	if err := gg.SavePNG("assets/logo.png", dr.Image(w, h)); err != nil {
		panic(err)
	}
}

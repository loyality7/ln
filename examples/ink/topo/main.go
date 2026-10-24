// Command topo draws an island as a topographic print: a contour line at every height,
// coloured from sea to peak, heavier where the land is in shade.
//
//	go run ./examples/ink/topo        # out/topo.png
package main

import (
	"math"

	"github.com/fogleman/gg"
	"github.com/loyality7/ln/components/nature"
	"github.com/loyality7/ln/ln/buffer"
	"github.com/loyality7/ln/ln/ink"
	"github.com/loyality7/ln/ln/solid"
	"github.com/loyality7/ln/ln/stage"
)

func main() {
	const W, H = 1600, 1000
	height := func(x, y float64) float64 { // an island: noise lifted in the middle, flat sea outside
		r := math.Hypot(x, y) / 3.8
		return math.Max(0, 3.2*(nature.FBM(x*0.32+5, y*0.32, 6)-0.42+0.5*(1-r*r)))
	}
	st := stage.New(W, H, 1)
	st.Root.Surface(solid.Heightfield(9, 9, 170, 170, height))
	st.Cam.Eye.At(0, -10.5, 7.2)
	st.Cam.Target.At(0, 0.2, 0.3)
	st.Cam.Fovy = 34

	dr := ink.NewDrawing(W, H)
	dr.Paper = ink.RGB(0.97, 0.94, 0.86)

	// contour lines: where the surface under each pixel is exactly this high
	shot := st.Shot(0)
	g := buffer.Gather(shot.Scene, shot.Cam, W, H, shot.Rig)
	z := make([]float64, W*H)
	for k, p := range g.P {
		z[k] = p.Z
	}
	var levels []float64
	for h := 0.04; h < 2.9; h += 0.025 {
		levels = append(levels, h)
	}
	palette := []ink.Color{ink.RGB(0.1, 0.45, 0.55), ink.RGB(0.25, 0.6, 0.3), ink.RGB(0.75, 0.65, 0.25), ink.RGB(0.7, 0.3, 0.15), ink.RGB(0.35, 0.2, 0.3)}
	var layer ink.Layer
	for i, paths := range ink.IsoLines(z, g.OK, W, H, levels, 1, 1) {
		t := levels[i] / 2.9 * float64(len(palette)-1)
		c := palette[int(t)].Mix(palette[min(int(t)+1, len(palette)-1)], t-math.Floor(t))
		for _, p := range paths {
			w := make([]float64, len(p))
			for j, q := range p { // pen pressure follows the light: heavier in shade
				w[j] = 0.6 + 1.8*g.Dark[min(H-1, int(q.Y))*W+min(W-1, int(q.X))]
			}
			layer.Lines = append(layer.Lines, ink.Line{Pts: p, W: w, Color: c})
		}
	}
	dr.Layers = append(dr.Layers, layer)
	gg.SavePNG("out/topo.png", dr.Image(0, 0))
}

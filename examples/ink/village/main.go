package main

import (
	"flag"
	"fmt"
	"math"
	"os"

	"github.com/fogleman/gg"
	"github.com/loyality7/ln/components/world"
	"github.com/loyality7/ln/ln/anim"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/ink"
	"github.com/loyality7/ln/ln/scene"
)

var (
	w, h = 1280.0, 720.0
	up   = geom.V(0, 0, 1)
)

// ground: rolling hills outside a flat clearing around the houses
func ground(x, y float64) float64 {
	hills := 1.1*math.Sin(x*0.22)*math.Cos(y*0.18) + 0.6*math.Sin(x*0.11+y*0.13)
	return hills * world.Smoothstep(7, 16, math.Hypot(x, y))
}

var terrain = world.Terrain(ground, 36, 110)

// frame builds the whole village for time t in [0,1) and inks it.
func frame(t float64) anim.Frame {
	tau := 2 * math.Pi
	// camera dolly along a spline, always looking at the cottage area
	eye := anim.Spline([]geom.Vector{geom.V(18, -20, 9), geom.V(11, -16, 5.5), geom.V(4, -13, 3.6), geom.V(-4, -13, 4), geom.V(-11, -15, 6)})(anim.Segment(t, 0, 1, anim.InOutSine))
	center := anim.LerpVec(geom.V(0, 0, 2.2), geom.V(-2.5, 1, 2.6), t)

	s := &scene.Scene{}
	s.Add(terrain)
	for _, p := range world.House(geom.V(0, 0, 0), 0.0, 5.2, 4, 2.4) {
		s.Add(p)
	}
	for _, p := range world.House(geom.V(-9, 6, ground(-9, 6)), 0.6, 3.6, 3, 2.0) {
		s.Add(p)
	}
	for _, p := range world.Fence(geom.V(-4.5, -4.5, 0), geom.V(4.5, -4.5, 0), 10, 0.9) {
		s.Add(p)
	}
	trees := []struct{ x, y, h float64 }{
		{-5.5, -2, 4.6}, {6.5, -1, 5.2}, {7.5, 5, 4.2}, {-6, 4.5, 4.8}, {11, -5, 5}, {-12, -3, 5.6}, {3, 10, 5.2}, {-2, 12, 5.6},
	}
	for i, tr := range trees {
		sway := 0.12 * math.Sin(t*tau*2+float64(i))
		for _, p := range world.Tree(eye, up, geom.V(tr.x, tr.y, ground(tr.x, tr.y)), tr.h, int64(i+3), sway) {
			s.Add(p)
		}
	}
	for i, c := range []geom.Vector{geom.V(-6, 14, 13), geom.V(8, 18, 15), geom.V(-16, 8, 14)} {
		drift := geom.V(t*8-4, 0, 0)
		for _, p := range world.Cloud(eye, up, c.Add(drift), 3.2, int64(i)) {
			s.Add(p)
		}
	}

	// sun arcs across the sky so the shadows sweep over the village
	az := anim.Lerp(0.55*math.Pi, 1.25*math.Pi, t)
	el := 0.75 + 0.25*math.Sin(t*math.Pi)
	st := ink.DefaultStyle()
	st.Light = geom.V(math.Cos(az)*math.Cos(el), math.Sin(az)*math.Cos(el), math.Sin(el))
	st.WobbleSeed = math.Floor(t*6*8) * 1.7 // a 6 second film // lines "boil" 8 times a second, like hand-drawn cels
	st.Wobble = 0.9
	dr := ink.Render(s, eye, center, up, w, h, 40, st)
	return dr.Reveal(anim.Segment(t, 0, 0.18, anim.InOutCubic))
}

func main() {
	still := flag.Float64("still", -1, "render one frame at this t (0..1) to out/village_still.png")
	frames := flag.Int("frames", 144, "number of video frames")
	flag.Float64Var(&w, "w", 1280, "width in pixels")
	flag.Float64Var(&h, "h", 720, "height in pixels")
	flag.Parse()
	os.MkdirAll("out", 0755)
	if *still >= 0 {
		gg.SavePNG("out/village_still.png", frame(*still).Image(w, h))
		return
	}
	if err := anim.FrameAnimation(frame).WriteVideo("out/village.mp4", *frames, 24, w, h); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

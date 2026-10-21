// Command blackhole animates the funnel surface z = -k/(x^2+y^2): it breathes
// deeper and shallower, ripples run down its wall, and the camera circles it.
// The same drawing recipe as examples/functions/function, once per frame.
//
//	go run ./examples/functions/blackhole              # out/blackhole.mp4
//	go run ./examples/functions/blackhole -fmt gif -w 480 -frames 72
package main

import (
	"flag"
	"fmt"
	"math"
	"os"

	"github.com/loyality7/ln/ln/anim"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/scene"
	"github.com/loyality7/ln/ln/shapes"
)

func main() {
	size := flag.Float64("w", 720, "picture size in pixels (square)")
	frames := flag.Int("frames", 96, "frames in the loop")
	fps := flag.Float64("fps", 24, "frames per second")
	format := flag.String("fmt", "mp4", "mp4 | gif | webm")
	still := flag.Float64("still", -1, "write one picture at this point of the loop (0..1) instead")
	out := flag.String("out", "out/blackhole", "output path without extension")
	flag.Parse()

	box := geom.Box{Min: geom.V(-2, -2, -4), Max: geom.V(2, 2, 2)}
	frame := func(t float64) geom.Paths {
		phase := 2 * math.Pi * t // one full loop, so the film repeats seamlessly
		depth := 1 + 0.35*math.Sin(phase)
		funnel := func(x, y float64) float64 {
			r2 := x*x + y*y
			return -depth/r2 + 0.05*math.Sin(5*math.Sqrt(r2)-phase*2) // ripples running down the wall
		}
		s := scene.Scene{}
		s.Add(shapes.NewFunction(funnel, box, shapes.Below))
		// the camera circles the funnel, keeping it off-centre like the original picture
		a := phase
		eye := geom.V(3*math.Cos(a), 3*math.Sin(a), 3)
		centre := geom.V(1.1*math.Cos(a), 1.1*math.Sin(a), 0)
		return s.Render(eye, centre, geom.V(0, 0, 1), *size, *size, 50, 0.1, 100, 0.01)
	}

	os.MkdirAll("out", 0o755)
	if *still >= 0 {
		frame(*still).WriteToPNG(*out+".png", *size, *size)
		fmt.Println("wrote", *out+".png")
		return
	}
	path := *out + "." + *format
	fmt.Println("rendering", *frames, "frames to", path)
	film := anim.Animation(frame)
	if err := film.WriteVideo(path, *frames, *fps, *size, *size); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

package main

import (
	"flag"
	"fmt"
	"math"
	"os"

	"github.com/loyality7/ln/components/env"
	"github.com/loyality7/ln/ln/anim"
	"github.com/loyality7/ln/ln/ink"
	"github.com/loyality7/ln/ln/solid"
	"github.com/loyality7/ln/ln/stage"
)

// A cottage, trees and a lamp post under a moving sun. The scene is described
// once; the time of day and the look (ink, manga, ASCII) are separate choices.
func main() {
	hour := flag.Float64("hour", -1, "render one still at this hour (0-24); omit for the full day video")
	lookName := flag.String("look", "ink", "ink | toon | ascii")
	out := flag.String("out", "out/daycycle", "output path without extension")
	flag.Parse()

	st := stage.New(1280, 720, 15)
	sky := env.Noon()
	st.Atmosphere = sky
	switch *lookName {
	case "toon":
		st.Look = stage.NewToonLook()
	case "ascii":
		st.Look = stage.NewASCIILook(150)
	}
	root := st.Root

	// ground and cottage
	root.Add(solid.Box(70, 70, 0.4)).At(0, 0, -0.2).NoOutline()
	house := root.Group("house").At(0, 1, 0)
	house.Add(solid.Box(4, 3, 2.2)).At(0, 0, 1.1)
	house.Add(solid.Lathe([]solid.Pt{{X: 3.3, Y: 0}, {X: 0, Y: 1.7}}, 4)).At(0, 0, 2.2).Turn(0, 0, math.Pi/4)
	house.Add(solid.Box(0.6, 0.12, 1.3)).At(0, -1.5, 0.65)
	house.Add(solid.Box(0.45, 0.45, 1.1)).At(1.1, 0.5, 3.0)
	for _, x := range []float64{-1.25, 1.25} { // windows that glow after dark
		house.Add(solid.Box(0.75, 0.1, 0.75)).At(x, -1.52, 1.4).Ink(ink.Material{Class: ink.White, Line: true, LineW: 1}).Emit(0.55, 3.2)
	}

	// trees
	for i, p := range [][3]float64{{-5, -1, 1.1}, {6, 2, 1.3}, {-7, 5, 1.0}, {4.5, 6, 1.2}, {-3, 8, 1.4}} {
		s := p[2]
		root.Add(solid.Cylinder(0.16*s, 1.2*s)).At(p[0], p[1], 0)
		for k := 0; k < 3; k++ {
			a := float64(i*3+k) * 2.1
			root.Add(solid.Ellipsoid(0.95*s, 0.95*s, 0.85*s)).At(p[0]+0.45*s*math.Cos(a), p[1]+0.45*s*math.Sin(a), (1.6+0.45*float64(k))*s)
		}
	}

	// lamp post: a bulb that lights the ground at night
	lamp := root.Group("lamp").At(3, -3, 0)
	lamp.Add(solid.Cylinder(0.07, 2.5))
	lamp.Add(solid.Sphere(0.2)).At(0, 0, 2.65).Ink(ink.Material{Class: ink.White, Line: true, LineW: 1}).Emit(1.6, 4.5)

	st.Cam.Eye.At(9, -12, 3.4)
	st.Cam.Target.At(0.3, 0, 1.9)
	st.Cam.Fovy = 38

	if *hour >= 0 { // a single moment: set the clock and draw
		sky.Hour = *hour
		path := fmt.Sprintf("%s_%s_%04.1f.png", *out, *lookName, *hour)
		if err := st.Still(path, 0); err != nil {
			fmt.Println(err)
		}
		return
	}

	// one day: dawn to the next dawn
	sky.Hour = 5
	st.TL.Float(&sky.Hour, 29, 8, anim.Linear)
	if err := st.Video(*out + "_" + *lookName + ".mp4"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

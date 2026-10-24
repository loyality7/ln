// Command blackhole is a camera tour of a black hole in ln's ink style: white line art
// on black, the disk's far side bent over the shadow by gravitational lensing. The
// camera starts far away and edge-on, zooms in, climbs above the disk until the
// lensed ring opens up, swoops down to skim it, then pulls back out.
//
//	go run ./examples/space/blackhole                    # the tour, out/space/blackhole.mp4
//	go run ./examples/space/blackhole -still 7.5         # one picture, 7.5 s into the tour
//	go run ./examples/space/blackhole -fmt gif -w 288 -h 512 -fps 12
package main

import (
	"flag"
	"fmt"
	"math"
	"os"

	"github.com/fogleman/gg"
	"github.com/loyality7/ln/components/blackhole"
	"github.com/loyality7/ln/ln/anim"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/glow"
)

// shot is where the camera is at one moment of the tour.
type shot struct {
	t        float64 // seconds
	distance float64 // from the hole: smaller zooms in
	incline  float64 // height above the disk plane: 0 edge-on, 1.57 straight down
	azimuth  float64 // position around the hole
}

var tour = []shot{
	{0.0, 500, 0.04, 0.0},   // far away, edge-on: a small bright sliver
	{2.5, 240, 0.08, 0.7},   // zoom in
	{5.0, 95, 0.40, 1.8},    // closer, rising above the disk
	{7.5, 60, 0.95, 3.0},    // high above and close: the lensed ring opens wide
	{10.0, 48, 0.05, 4.4},   // swoop down to skim the disk
	{12.0, 150, 0.22, 5.4},  // pull back
	{14.0, 420, 0.10, 6.28}, // far again, back where it began
}

// camera is the smooth path through the shots: a curve through (log distance,
// incline, azimuth), so zooms feel even and the motion never jerks at a shot.
func camera(t float64) (distance, incline, azimuth float64) {
	pts := make([]geom.Vector, len(tour))
	for i, s := range tour {
		pts[i] = geom.V(math.Log(s.distance), s.incline, s.azimuth)
	}
	// time -> position along the curve, easing through each shot
	t = math.Max(tour[0].t, math.Min(tour[len(tour)-1].t, t))
	u := 0.0
	for i := 1; i < len(tour); i++ {
		if t <= tour[i].t {
			local := anim.InOutSine((t - tour[i-1].t) / (tour[i].t - tour[i-1].t))
			u = (float64(i-1) + local) / float64(len(tour)-1)
			break
		}
	}
	c := anim.Spline(pts)(u)
	return math.Exp(c.X), math.Max(0.02, c.Y), c.Z
}

func main() {
	w := flag.Int("w", 576, "width in pixels")
	h := flag.Int("h", 1024, "height in pixels")
	fps := flag.Float64("fps", 24, "frames per second")
	format := flag.String("fmt", "mp4", "mp4 | gif | webm")
	still := flag.Float64("still", -1, "write one picture at this time in seconds instead of a video")
	out := flag.String("out", "out/space/blackhole", "output path without extension")
	smooth := flag.Bool("glow", false, "smooth glowing light with bloom, like a photograph, instead of lines")
	colour := flag.Bool("colour", false, "glowing colour: hot gold inside, red outside, bluer on the side moving toward us")
	flag.Parse()

	duration := tour[len(tour)-1].t
	frame := func(t float64) *blackhole.Params {
		p := blackhole.Default(*w, *h)
		p.Distance, p.Incline, p.Azimuth = camera(t)
		p.Time = t
		p.Colour = *colour
		if *smooth { // the camera dives in during the tour, and starlight streaks with it
			p.Streak = 1.2 * math.Max(0, math.Min(1, (t-5)/2)) * math.Max(0, math.Min(1, (11-t)/2))
		}
		return &p
	}
	picture := func(p blackhole.Params) anim.Frame {
		if *smooth {
			return glow.Frame{Canvas: blackhole.Emit(p), Look: glow.Soft()}
		}
		return blackhole.Draw(p)
	}
	os.MkdirAll("out/space", 0o755)
	if *still >= 0 {
		if err := gg.SavePNG(*out+".png", picture(*frame(*still)).Image(float64(*w), float64(*h))); err != nil {
			panic(err)
		}
		fmt.Println("wrote", *out+".png")
		return
	}
	frames := int(duration * *fps)
	film := anim.FrameAnimation(func(u float64) anim.Frame {
		return picture(*frame(u * float64(frames) / *fps))
	})
	path := *out + "." + *format
	fmt.Println("rendering", frames, "frames to", path)
	if err := film.WriteVideo(path, frames, *fps, float64(*w), float64(*h)); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

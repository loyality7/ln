// Package studio is the simple front door to the engine: write a scene in a few
// lines and get a picture or a video. It handles the command line, output files,
// looks, time of day, terrain, water and scattering, so a scene is only the part
// that is yours.
//
//	func main() {
//		studio.Run(func(s *studio.Scene) {
//			s.Look("manga")
//			s.TimeOf("dusk")
//			s.Camera(studio.V(0, -12, 2), studio.V(0, 0, 1), 40)
//			s.Root.Sphere(1).At(0, 0, 1)
//		})
//	}
//
// Run it with -still 0 for a picture, or with no flags for a video when the scene
// has motion. -look, -sky, -w, -h and -out restyle any scene from the command line.
package studio

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strconv"

	"github.com/loyality7/ln/components/env"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/ink"
	"github.com/loyality7/ln/ln/stage"
)

// Short names so scene code needs one import.
type (
	Vec  = geom.Vector
	Node = stage.Node
	Rand = rand.Rand
)

// V builds a point or direction.
func V(x, y, z float64) Vec { return geom.V(x, y, z) }

// Rect is a region on the ground.
type Rect struct{ X0, X1, Y0, Y1 float64 }

// Scene is a stage with friendly defaults and helpers.
type Scene struct {
	*stage.Stage
	Sky  *env.Environment
	Rand *rand.Rand // seeded from -seed, for repeatable randomness

	ground func(x, y float64) float64
	opts   options
}

type options struct {
	w, h, fps      int
	still          float64
	out, look, sky string
	format         string
	seed           int64
}

// skyHours are the named times of day.
var skyHours = map[string]float64{
	"dawn": 6, "sunrise": 6, "morning": 9, "noon": 12, "afternoon": 15,
	"golden": 17.2, "sunset": 18, "dusk": 18.7, "night": 22, "midnight": 0,
}

// Run parses the command line, builds the scene with build, and renders it:
// a picture with -still T, otherwise a video if the scene has motion.
func Run(build func(*Scene)) {
	var o options
	flag.IntVar(&o.w, "w", 1280, "width in pixels")
	flag.IntVar(&o.h, "h", 720, "height in pixels")
	flag.IntVar(&o.fps, "fps", 24, "frames per second")
	flag.Float64Var(&o.still, "still", -1, "render one picture at this second instead of a video")
	flag.StringVar(&o.out, "out", "out/scene", "output path without extension")
	flag.StringVar(&o.format, "fmt", "mp4", "video format: mp4 | gif | webm")
	flag.StringVar(&o.look, "look", "", "ink | manga | sketch | technical | ascii (overrides the scene)")
	flag.StringVar(&o.sky, "sky", "", "dawn | morning | noon | afternoon | golden | dusk | night | midnight, or an hour 0-24")
	flag.Int64Var(&o.seed, "seed", 1, "random seed")
	flag.Parse()

	s := newScene(o)
	build(s)
	if o.look != "" {
		s.Look(o.look)
	}
	if o.sky != "" {
		s.TimeOf(o.sky)
	}
	if err := s.render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newScene(o options) *Scene {
	s := &Scene{opts: o, Rand: rand.New(rand.NewSource(o.seed))}
	s.Stage = stage.New(float64(o.w), float64(o.h), float64(o.fps))
	s.Sky = env.Noon()
	s.Atmosphere = s.Sky
	s.Look("ink")
	return s
}

func (s *Scene) render() error {
	if s.opts.still >= 0 || s.TL.Duration() == 0 {
		t := max(s.opts.still, 0)
		path := s.opts.out + ".png"
		fmt.Println("writing", path)
		return s.Still(path, t)
	}
	path := s.opts.out + "." + s.opts.format
	fmt.Println("writing", path, "-", s.Frames(), "frames")
	return s.Video(path)
}

// Look picks how the scene is drawn: "ink", "manga", "sketch", "technical", "colour" or "ascii".
// "colour" paints each object its Node.Paint colour in three tones from the light, under ink outlines.
func (s *Scene) Look(name string) {
	if name == "ascii" {
		s.Stage.Look = stage.NewASCIILook(130)
		return
	}
	if name == "colour" {
		l := stage.NewColourLook()
		l.Style.Sky = [3]ink.Color{ink.RGB(0.99, 0.78, 0.45), ink.RGB(0.9, 0.38, 0.28), ink.RGB(0.3, 0.14, 0.36)}
		s.Stage.Look = l
		return
	}
	style, ok := ink.Preset(name)
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown look %q; choose from %v and ascii\n", name, ink.PresetNames())
		os.Exit(2)
	}
	if name == "manga" {
		s.Stage.Look = &stage.ToonLook{Style: style, BoilFPS: 8}
		return
	}
	s.Stage.Look = &stage.InkLook{Style: style, BoilFPS: 12}
}

// SkyColours sets the sky of the "colour" look, from its lightest to its darkest part.
func (s *Scene) SkyColours(light, mid, dark ink.Color) {
	if l, ok := s.Stage.Look.(*stage.ColourLook); ok {
		l.Style.Sky = [3]ink.Color{light, mid, dark}
	}
}

// Time sets the hour of day, 0-24.
func (s *Scene) Time(hour float64) { s.Sky.Hour = hour }

// TimeOf sets a named time of day (dawn, noon, dusk, night, ...) or an hour like "17.5".
func (s *Scene) TimeOf(name string) {
	if h, ok := skyHours[name]; ok {
		s.Sky.Hour = h
		return
	}
	h, err := strconv.ParseFloat(name, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unknown time of day %q\n", name)
		os.Exit(2)
	}
	s.Sky.Hour = h
}

// Day animates the hour of day from one value to another over dur seconds.
func (s *Scene) Day(from, to, dur float64) {
	s.Sky.Hour = from
	s.TL.Float(&s.Sky.Hour, to, dur, nil)
}

// Camera places the camera: where it stands, what it looks at, and its field of view in degrees.
func (s *Scene) Camera(eye, target Vec, fov float64) {
	s.Cam.Eye.At(eye.X, eye.Y, eye.Z)
	s.Cam.Target.At(target.X, target.Y, target.Z)
	s.Cam.Fovy = fov
}

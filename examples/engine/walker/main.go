package main

import (
	"flag"
	"fmt"
	"math"
	"os"

	"github.com/loyality7/ln/components/env"
	"github.com/loyality7/ln/ln/anim"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/solid"
	"github.com/loyality7/ln/ln/stage"
)

// limb hangs a box from a pivot, so rotating the pivot swings the limb.
func limb(parent *stage.Node, x, z, length, thick float64) *stage.Node {
	pivot := parent.Group("limb").At(x, 0, z)
	pivot.Add(solid.Box(thick, thick, length)).At(0, 0, -length/2)
	return pivot
}

func pine(root *stage.Node, x, y, s float64) {
	root.Add(solid.Cylinder(0.18*s, 1.0*s)).At(x, y, 0)
	for i := 0; i < 3; i++ {
		k := float64(i)
		root.Add(solid.Cone((1.25-0.3*k)*s, 1.5*s)).At(x, y, (0.8+0.8*k)*s)
	}
}

func cottage(root *stage.Node, x, y, yaw float64) {
	h := root.Group("cottage").At(x, y, 0).Turn(0, 0, yaw)
	h.Add(solid.Box(3, 2.4, 1.8)).At(0, 0, 0.9)
	h.Add(solid.Lathe([]solid.Pt{{X: 2.5, Y: 0}, {X: 0, Y: 1.3}}, 4)).At(0, 0, 1.8).Turn(0, 0, math.Pi/4)
	h.Add(solid.Box(0.5, 0.1, 1.0)).At(0, -1.22, 0.5)
}

func main() {
	still := flag.Float64("still", -1, "render one frame at this time (seconds)")
	flag.Parse()

	st := stage.New(1280, 720, 24)
	sky := env.At(9.5) // the sun sweeps from morning to late afternoon
	st.Atmosphere = sky
	root := st.Root

	// world
	root.Add(solid.Box(80, 80, 0.4)).At(0, 0, -0.2).NoOutline()
	for i := 0; i < 8; i++ {
		root.Add(solid.Box(0.3, 1.4, 0.02)).At(0, -9+float64(i)*3, 0.01)
	}
	for _, p := range [][3]float64{{-4, -5, 1.1}, {5, -2, 1.3}, {-6, 3, 1.0}, {6, 6, 1.2}, {-3, 9, 1.3}, {9, -7, 1.0}, {-9, -1, 1.4}} {
		pine(root, p[0], p[1], p[2])
	}
	cottage(root, -7, -9, 0.4)
	cottage(root, 8, 2, math.Pi)
	cottage(root, -8, 7, -0.6)

	// robot: every part is a node, so the rig is just a hierarchy
	robot := root.Group("robot").At(0, -7, 0)
	body := robot.Group("body").At(0, 0, 1.7)
	body.Add(solid.Box(0.9, 0.5, 1.0)).At(0, 0, 0.55)
	head := body.Group("head").At(0, 0, 1.35)
	head.Add(solid.Sphere(0.38))
	head.Add(solid.Box(0.5, 0.12, 0.14)).At(0, 0.34, 0.04)
	head.Add(solid.Cylinder(0.04, 0.35)).At(0, 0, 0.36)
	legL, legR := limb(body, -0.22, 0, 1.6, 0.26), limb(body, 0.22, 0, 1.6, 0.26)
	armL, armR := limb(body, -0.62, 0.95, 0.9, 0.2), limb(body, 0.62, 0.95, 0.9, 0.2)
	for _, l := range []*stage.Node{legL, legR} {
		l.Add(solid.Box(0.3, 0.55, 0.14)).At(0, 0.1, -1.62)
	}

	// camera rides along with the robot
	st.Cam.Target.Reparent(robot).At(0, 0, 1.9)
	st.Cam.Eye.At(7, -13, 4)

	tl := st.TL
	tl.Par(func() {
		tl.Move(robot, geom.V(0, 5, 0), 6, anim.Linear) // the walk
		tl.Do(6, func(p float64) {                      // the walk cycle: a pure function of time
			s := p * 6
			swing := math.Sin(2*math.Pi*1.1*s) * 0.65
			legL.Rot.X, legR.Rot.X = swing, -swing
			armL.Rot.X, armR.Rot.X = -swing*0.8, swing*0.8
			body.Pos.Z = 1.7 + 0.05*math.Abs(math.Cos(2*math.Pi*1.1*s))
			body.Rot.Z = 0.06 * math.Sin(2*math.Pi*1.1*s)
		})
		tl.Seq(func() { // camera: swoop in, then circle the walker
			tl.Move(st.Cam.Eye, geom.V(5.36, -4.67, 2.6), 3, anim.InOutSine)
			tl.Do(3, func(p float64) {
				a := -0.6 + p*2.4
				st.Cam.Eye.Pos = robot.Pos.Add(geom.V(math.Cos(a)*6.5-0.0, math.Sin(a)*6.5, 2.6+p*1.2))
			})
		})
		tl.Seq(func() { // sun swings round
			tl.Float(&sky.Hour, 16.5, 6, anim.InOutSine)
		})
	})

	os.MkdirAll("out", 0755)
	if *still >= 0 {
		if err := st.Still("out/walker_still.png", *still); err != nil {
			fmt.Println(err)
		}
		return
	}
	if err := st.Video("out/walker.mp4"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

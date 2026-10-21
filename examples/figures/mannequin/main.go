package main

import (
	"fmt"
	"os"

	"github.com/loyality7/ln/components/mannequin"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/light"
	"github.com/loyality7/ln/ln/solid"
	"github.com/loyality7/ln/ln/stage"
)

// A turnaround of the mannequin: front, three-quarter, side, back.
func main() {
	st := stage.New(480, 720, 24)
	st.Atmosphere = stage.Studio{Lights: light.Rig{
		Lights:  []light.Light{light.Directional{Dir: geom.V(-0.5, -0.7, 0.8), Power: 1}},
		Ambient: 0.35, Shadows: true,
	}}
	st.Look = stage.NewToonLook()
	st.Root.Add(solid.Box(8, 8, 0.1)).At(0, 0, -0.05).NoOutline()
	mannequin.New(st.Root, 1.75).Root.At(0, 0, 0)
	st.Cam.Target.At(0, 0, 0.9)
	st.Cam.Fovy = 34
	views := []geom.Vector{geom.V(0, -4.6, 1.0), geom.V(3.2, -3.2, 1.0), geom.V(4.6, 0, 1.0), geom.V(0, 4.6, 1.0)}
	for i, eye := range views {
		st.Cam.Eye.At(eye.X, eye.Y, eye.Z)
		if err := st.Still(fmt.Sprintf("out/mannequin_%d.png", i), 0); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}

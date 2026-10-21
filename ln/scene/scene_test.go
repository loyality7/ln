package scene_test

import (
	"testing"

	"github.com/loyality7/ln/ln/anim"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/scene"
	"github.com/loyality7/ln/ln/shapes"
)

func cubeScene() *scene.Scene {
	s := &scene.Scene{}
	s.Add(shapes.NewCube(geom.Vector{X: -1, Y: -1, Z: -1}, geom.Vector{X: 1, Y: 1, Z: 1}))
	return s
}

func render(s *scene.Scene, eye geom.Vector) geom.Paths {
	return s.Render(eye, geom.Vector{}, geom.Vector{Z: 1}, 100, 100, 50, 0.1, 100, 0.1)
}

func TestRenderCube(t *testing.T) {
	if len(render(cubeScene(), geom.Vector{X: 4, Y: 3, Z: 2})) == 0 {
		t.Fatal("no paths rendered")
	}
}

func TestParallelFilterDeterministic(t *testing.T) {
	eye := geom.Vector{X: 4, Y: 3, Z: 2}
	if render(cubeScene(), eye).String() != render(cubeScene(), eye).String() {
		t.Fatal("render not deterministic")
	}
}

func TestAnimationFrames(t *testing.T) {
	s := cubeScene()
	a := anim.Animation(func(tt float64) geom.Paths {
		return render(s, geom.Vector{X: 4, Y: tt, Z: 2})
	})
	if got := len(a.Frames(4)); got != 4 {
		t.Fatalf("frames=%d", got)
	}
}

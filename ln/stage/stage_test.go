package stage

import (
	"math"
	"testing"

	"github.com/loyality7/ln/ln/anim"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/solid"
)

func near(a, b geom.Vector) bool { return a.Distance(b) < 1e-9 }

func TestChildFollowsParent(t *testing.T) {
	root := NewNode("r").At(10, 0, 0).Turn(0, 0, math.Pi/2)
	child := root.Group("c").At(2, 0, 0)
	// parent turned 90deg about z, so child's +x offset points along +y
	if got := child.WorldPos(); !near(got, geom.Vector{X: 10, Y: 2}) {
		t.Fatalf("world pos %v", got)
	}
}

func TestTimelineSequencing(t *testing.T) {
	st := New(64, 64, 10)
	n := NewNode("n")
	tl := st.TL
	tl.Move(n, geom.Vector{X: 10}, 2, anim.Linear)
	tl.Wait(1)
	tl.Move(n, geom.Vector{X: 0}, 2, anim.Linear)
	if tl.Duration() != 5 {
		t.Fatalf("duration %v", tl.Duration())
	}
	for _, c := range []struct{ t, x float64 }{{0, 0}, {1, 5}, {2, 10}, {2.5, 10}, {4, 5}, {5, 0}} {
		tl.apply(c.t)
		if math.Abs(n.Pos.X-c.x) > 1e-9 {
			t.Errorf("t=%v x=%v want %v", c.t, n.Pos.X, c.x)
		}
	}
}

func TestParRunsTogether(t *testing.T) {
	st := New(64, 64, 10)
	a, b := NewNode("a"), NewNode("b")
	tl := st.TL
	tl.Par(func() {
		tl.Move(a, geom.Vector{X: 4}, 2, anim.Linear)
		tl.Move(b, geom.Vector{X: 6}, 3, anim.Linear)
	})
	tl.Move(a, geom.Vector{X: 0}, 1, anim.Linear)
	if tl.Duration() != 4 {
		t.Fatalf("duration %v", tl.Duration())
	}
	tl.apply(1)
	if a.Pos.X != 2 || b.Pos.X != 2 {
		t.Fatalf("a=%v b=%v", a.Pos.X, b.Pos.X)
	}
	tl.apply(3.5) // a: back from 4 toward 0 starting at t=3
	if math.Abs(a.Pos.X-2) > 1e-9 {
		t.Fatalf("a=%v", a.Pos.X)
	}
}

func TestSeqLaneInsidePar(t *testing.T) {
	st := New(64, 64, 10)
	a := NewNode("a")
	tl := st.TL
	tl.Par(func() {
		tl.Seq(func() {
			tl.Move(a, geom.Vector{X: 1}, 1, anim.Linear)
			tl.Move(a, geom.Vector{X: 3}, 1, anim.Linear)
		})
		tl.Wait(1.5)
	})
	if tl.Duration() != 2 {
		t.Fatalf("duration %v", tl.Duration())
	}
}

func TestShowHideAndEffect(t *testing.T) {
	st := New(64, 64, 10)
	n := NewNode("n")
	tl := st.TL
	tl.Wait(1)
	tl.Hide(n)
	tl.Wait(1)
	tl.Show(n)
	tl.Do(2, func(p float64) { n.Pos.Y = p * 4 })
	tl.apply(0.5)
	if !n.shown() {
		t.Fatal("should be shown at 0.5")
	}
	tl.apply(1.5)
	if n.shown() {
		t.Fatal("should be hidden at 1.5")
	}
	tl.apply(3)
	if !n.shown() || math.Abs(n.Pos.Y-2) > 1e-9 {
		t.Fatalf("shown=%v y=%v", n.shown(), n.Pos.Y)
	}
}

func TestRenderStillAndMovedMesh(t *testing.T) {
	st := New(120, 80, 10)
	ball := st.Root.Add(solid.Sphere(1)).At(0, 0, 1)
	st.Cam.Eye.At(0, -8, 3)
	st.Cam.Target.At(0, 0, 1)
	st.TL.Move(ball, geom.Vector{X: 3, Z: 1}, 1, anim.Linear)
	if img := st.Frame(0.5).Image(120, 80); img.Bounds().Dx() != 120 {
		t.Fatal("bad image")
	}
	if st.Frames() != 10 {
		t.Fatalf("frames %d", st.Frames())
	}
}

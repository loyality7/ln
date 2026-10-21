package anim

import (
	"math"
	"testing"

	"github.com/loyality7/ln/ln/geom"
)

func TestEaseEndpoints(t *testing.T) {
	for name, e := range map[string]Ease{"InOutCubic": InOutCubic, "OutBack": OutBack, "OutElastic": OutElastic, "OutBounce": OutBounce, "InOutSine": InOutSine} {
		if math.Abs(e(0)) > 1e-9 || math.Abs(e(1)-1) > 1e-9 {
			t.Errorf("%s: e(0)=%v e(1)=%v", name, e(0), e(1))
		}
	}
}

func TestTrack(t *testing.T) {
	tr := Track{{1, 10, nil}, {0, 0, nil}, {2, 20, nil}}
	for _, c := range []struct{ t, want float64 }{{-1, 0}, {0.5, 5}, {1.5, 15}, {9, 20}} {
		if got := tr.At(c.t); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("At(%v)=%v want %v", c.t, got, c.want)
		}
	}
}

func TestSplinePassesThroughPoints(t *testing.T) {
	pts := []geom.Vector{{X: 0}, {X: 1, Y: 2}, {X: 3, Y: 1}, {X: 4}}
	s := Spline(pts)
	if s(0) != pts[0] || s(1) != pts[3] || s(1.0/3).Distance(pts[1]) > 1e-9 {
		t.Fatal("spline misses control points")
	}
}

func TestReveal(t *testing.T) {
	paths := geom.Paths{{{X: 0}, {X: 10}}, {{Y: 5}, {Y: 15}}}
	got := Reveal(paths, 0.75) // 15 of 20 units: all of path 0, half of path 1
	if len(got) != 2 || got[1][1].Y != 10 {
		t.Fatalf("got %v", got)
	}
	if len(Reveal(paths, 0)) != 0 || len(Reveal(paths, 1)) != 2 {
		t.Fatal("bad bounds")
	}
}

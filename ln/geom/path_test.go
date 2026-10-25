package geom

import (
	"math"
	"testing"
)

func TestPath2DTransformsAndCurves(t *testing.T) {
	p0 := Vector{X: 0, Y: 0}
	p1 := Vector{X: 10, Y: 20}
	p2 := Vector{X: 20, Y: 20}
	p3 := Vector{X: 30, Y: 0}

	curve := CubicBezier(p0, p1, p2, p3, 10)
	if len(curve) != 11 {
		t.Fatalf("expected 11 points, got %d", len(curve))
	}
	if curve[0] != p0 || curve[len(curve)-1] != p3 {
		t.Fatalf("curve endpoints mismatch: %v, %v", curve[0], curve[len(curve)-1])
	}

	circle := Circle(10, 10, 5, 16)
	if len(circle) != 16 {
		t.Fatalf("expected 16 points for circle, got %d", len(circle))
	}

	shifted := circle.Translate(5, -5)
	if math.Abs(shifted[0].X-(circle[0].X+5)) > 1e-6 {
		t.Fatalf("unexpected translation result: %v", shifted[0])
	}

	box := Box{Min: Vector{X: 0, Y: 0}, Max: Vector{X: 10, Y: 10}}
	rect := box.Rect()
	if len(rect) != 5 {
		t.Fatalf("expected 5 points for closed rect, got %d", len(rect))
	}

	c0, c1, ok := box.ClipSegment2D(Vector{X: -5, Y: 5}, Vector{X: 15, Y: 5})
	if !ok || c0.X != 0 || c1.X != 10 {
		t.Fatalf("expected clipped segment (0,5)-(10,5), got %v-%v (ok=%v)", c0, c1, ok)
	}
}

package studio

import (
	"math"
	"testing"
)

func testScene() *Scene { return newScene(options{w: 160, h: 90, fps: 10, seed: 3}) }

func TestTimeOfAcceptsNamesAndHours(t *testing.T) {
	s := testScene()
	s.TimeOf("dusk")
	if s.Sky.Hour != 18.7 {
		t.Fatalf("dusk = %v", s.Sky.Hour)
	}
	s.TimeOf("7.25")
	if s.Sky.Hour != 7.25 {
		t.Fatalf("hour = %v", s.Sky.Hour)
	}
}

func TestScatterStandsOnTerrainAndIsRepeatable(t *testing.T) {
	run := func() []Vec {
		s := testScene()
		s.Terrain(func(x, y float64) float64 { return 0.5 * x }, Rect{-10, 10, -10, 10})
		var pts []Vec
		s.Scatter(25, Rect{-5, 5, -5, 5}, func(p Vec, _ *Rand) { pts = append(pts, p) })
		return pts
	}
	a, b := run(), run()
	if len(a) != 25 || len(b) != 25 {
		t.Fatalf("counts %d %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatal("same seed must give the same scatter")
		}
		if math.Abs(a[i].Z-0.5*a[i].X) > 1e-9 {
			t.Fatalf("point %v is not on the terrain", a[i])
		}
	}
}

func TestDayAnimatesTheHour(t *testing.T) {
	s := testScene()
	s.Day(6, 18, 4)
	s.TL.Do(0, func(float64) {})
	s.Shot(2) // evaluates the timeline
	if math.Abs(s.Sky.Hour-12) > 1e-9 {
		t.Fatalf("hour at t=2 is %v, want 12", s.Sky.Hour)
	}
}

func TestEveryLookRenders(t *testing.T) {
	for _, name := range []string{"ink", "manga", "sketch", "technical", "ascii"} {
		s := testScene()
		s.Look(name)
		s.Camera(V(0, -6, 2), V(0, 0, 1), 40)
		s.Root.Sphere(1).At(0, 0, 1)
		if img := s.Frame(0).Image(160, 90); img.Bounds().Dx() == 0 {
			t.Errorf("%s produced no image", name)
		}
	}
}

package physics

import (
	"math"
	"testing"

	"github.com/loyality7/ln/ln/geom"
)

func near(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

func TestStep2SolvesHarmonicMotion(t *testing.T) {
	y, v := 0.0, 1.0 // y'' = -y from rest at 0 with speed 1 is y = sin(x)
	h := 0.01
	for i := 0; i < int(math.Pi/2/h); i++ {
		y, v = Step2(func(y, _ float64) float64 { return -y }, y, v, h)
	}
	if !near(y, 1, 1e-4) || !near(v, 0, 1e-2) {
		t.Fatalf("y=%v v=%v after a quarter turn", y, v)
	}
}

func TestStep3Projectile(t *testing.T) {
	a := func(p, v geom.Vector) geom.Vector { return geom.V(0, 0, -9.8) }
	p, v := geom.V(0, 0, 0), geom.V(3, 0, 10)
	for i := 0; i < 100; i++ {
		p, v = Step3(a, p, v, 0.01)
	}
	// after 1 s: x = 3, z = 10 - 4.9
	if !near(p.X, 3, 1e-9) || !near(p.Z, 5.1, 1e-9) || !near(v.Z, 0.2, 1e-9) {
		t.Fatalf("p=%v v=%v", p, v)
	}
}

func TestFlatSpaceGivesStraightLines(t *testing.T) {
	f := Field{Law: Flat(), Capture: 0.5, Escape: 500}
	out, exit := f.Trace(geom.V(-500, 20, 0), geom.V(1, 0, 0), func(Segment) bool { return true })
	if out != Escaped || exit.Distance(geom.V(1, 0, 0)) > 1e-3 {
		t.Fatalf("outcome %v exit %v", out, exit)
	}
	if out, _ := f.Trace(geom.V(-100, 0, 0), geom.V(1, 0, 0), func(Segment) bool { return true }); out != Captured {
		t.Fatalf("a ray aimed at the centre must be captured, got %v", out)
	}
}

func TestLightBendsByFourMOverB(t *testing.T) {
	g := Gravity{M: 1}
	f := Field{Law: g.Light(), Capture: g.SchwarzschildRadius(), Escape: 5000}
	const b = 200.0
	out, exit := f.Trace(geom.V(-5000, b, 0), geom.V(1, 0, 0), func(Segment) bool { return true })
	if out != Escaped {
		t.Fatalf("outcome %v", out)
	}
	angle := math.Acos(math.Min(1, exit.Dot(geom.V(1, 0, 0))))
	if want := g.Deflection(b); !near(angle, want, 0.05*want) {
		t.Fatalf("deflected by %.5f rad, expected about %.5f (4M/b)", angle, want)
	}
}

func TestCapturedInsideTheCriticalImpactParameter(t *testing.T) {
	g := Gravity{M: 1}
	f := Field{Law: g.Light(), Capture: g.SchwarzschildRadius(), Escape: 1000}
	crit := g.CriticalImpactParameter()
	for _, c := range []struct {
		b    float64
		want Outcome
	}{{crit * 0.93, Captured}, {crit * 1.08, Escaped}} {
		if out, _ := f.Trace(geom.V(-1000, c.b, 0), geom.V(1, 0, 0), func(Segment) bool { return true }); out != c.want {
			t.Errorf("b=%.2f: outcome %v, want %v", c.b, out, c.want)
		}
	}
}

func TestNewtonianCircularOrbitStaysCircular(t *testing.T) {
	f := Field{Law: Newtonian(0.1), Capture: 1, Escape: 1000, MaxTurn: 2 * math.Pi}
	out, _ := f.Trace(geom.V(10, 0, 0), geom.V(0, 1, 0), func(s Segment) bool {
		if !near(s.R1, 10, 1e-6) {
			t.Fatalf("radius drifted to %v", s.R1)
		}
		return true
	})
	if out != Stopped {
		t.Fatalf("a closed orbit never escapes or falls in: %v", out)
	}
}

func TestSegmentCrossPlane(t *testing.T) {
	s := Segment{From: geom.V(0, 0, 1), To: geom.V(2, 0, -3)}
	p, frac, ok := s.CrossPlane(geom.V(0, 0, 0), geom.V(0, 0, 1))
	if !ok || !near(frac, 0.25, 1e-12) || !near(p.X, 0.5, 1e-12) {
		t.Fatalf("p=%v frac=%v ok=%v", p, frac, ok)
	}
	if _, _, ok := (Segment{From: geom.V(0, 0, 1), To: geom.V(0, 0, 2)}).CrossPlane(geom.V(0, 0, 0), geom.V(0, 0, 1)); ok {
		t.Fatal("a segment on one side of the plane does not cross it")
	}
}

func TestOrbitalMechanics(t *testing.T) {
	g := Gravity{M: 1}
	if !near(g.CircularSpeed(100), 0.1, 1e-12) || !near(g.AngularVelocity(100), 0.001, 1e-12) {
		t.Fatal("Kepler circular orbit")
	}
	if !near(g.EscapeSpeed(100), math.Sqrt2*g.CircularSpeed(100), 1e-12) {
		t.Fatal("escape speed is sqrt(2) times circular speed")
	}
	if g.AngularVelocity(10) <= g.AngularVelocity(20) {
		t.Fatal("inner orbits must turn faster")
	}
	if !near(g.TimeDilation(2), 0, 1e-12) || g.TimeDilation(1e6) < 0.999 {
		t.Fatal("clocks stop at the Schwarzschild radius and run true far away")
	}
}

func TestDopplerAndBeaming(t *testing.T) {
	toward := Doppler(geom.V(0.5, 0, 0), geom.V(1, 0, 0))
	away := Doppler(geom.V(-0.5, 0, 0), geom.V(1, 0, 0))
	across := Doppler(geom.V(0, 0.5, 0), geom.V(1, 0, 0))
	if !near(toward, math.Sqrt(0.75)/0.5, 1e-12) || toward <= 1 || away >= 1 || !near(across, math.Sqrt(0.75), 1e-12) {
		t.Fatalf("toward=%v away=%v across=%v", toward, away, across)
	}
	if Beaming(toward, 3) <= Beaming(away, 3) || !near(Gamma(0.6), 1.25, 1e-12) {
		t.Fatal("approaching sources are brighter; gamma(0.6) = 1.25")
	}
}

func TestThermalRadiation(t *testing.T) {
	if ThinDiskTemperature(24, 6, 1) >= ThinDiskTemperature(12, 6, 1) || !near(ThinDiskTemperature(6, 6, 7), 7, 1e-12) {
		t.Fatal("a thin disk is hottest at its inner edge")
	}
	if Radiance(2) != 16 {
		t.Fatal("radiance grows as T^4")
	}
	r, _, b := Blackbody(2500)
	r2, _, b2 := Blackbody(15000)
	if r <= b || b2 <= r2 {
		t.Fatalf("cool glows red (r=%v b=%v), hot glows blue (r=%v b=%v)", r, b, r2, b2)
	}
	r, g, b := Blackbody(6600)
	if math.Abs(r-g) > 0.08 || math.Abs(g-b) > 0.08 {
		t.Fatalf("near 6600 K is close to white: %v %v %v", r, g, b)
	}
}

package env

import (
	"math"
	"testing"

	"github.com/loyality7/ln/ln/geom"
)

func elevation(d geom.Vector) float64 { return math.Asin(d.Z) * 180 / math.Pi }

func TestSunPath(t *testing.T) {
	e := New() // equinox, latitude 35
	e.Hour = 12
	if got := elevation(e.SunDir()); math.Abs(got-55) > 0.7 {
		t.Errorf("noon elevation %.1f want ~55", got)
	}
	if e.SunDir().Y >= 0 {
		t.Errorf("noon sun should be to the south at latitude 35, got %v", e.SunDir())
	}
	e.Hour = 6
	if got := elevation(e.SunDir()); math.Abs(got) > 1 || e.SunDir().X <= 0.9 {
		t.Errorf("sunrise elevation %.1f dir %v: want on the eastern horizon", got, e.SunDir())
	}
	e.Hour = 18
	if e.SunDir().X >= -0.9 {
		t.Errorf("sunset should be in the west: %v", e.SunDir())
	}
	e.Hour = 0
	if e.SunDir().Z >= 0 {
		t.Error("sun must be below the horizon at midnight")
	}
}

func TestDaylightAndRig(t *testing.T) {
	if Noon().Daylight() < 0.99 || Midnight().Daylight() > 0.01 {
		t.Fatal("daylight should be 1 at noon and 0 at midnight")
	}
	if n, d := len(Midnight().Rig().Lights), len(Noon().Rig().Lights); d != 1 || n != 1 {
		t.Fatalf("noon wants the sun only, midnight the moon only; got %d and %d lights", d, n)
	}
	if Noon().Rig().Ambient <= Midnight().Rig().Ambient {
		t.Fatal("ambient light should fall at night")
	}
}

func TestSkyDarkness(t *testing.T) {
	up := geom.Vector{Z: 1}
	if Noon().Darkness(up) > 0.2 || Midnight().Darkness(up) < 0.8 {
		t.Fatal("day sky should be pale and night sky black")
	}
	if g := Sunrise().Darkness(up); g > 0.5 {
		t.Fatalf("sunrise sky should already be light, got %.2f", g)
	}
}

func TestHourWrapsAroundMidnight(t *testing.T) {
	a, b := At(2).SunDir(), At(26).SunDir()
	if a.Distance(b) > 1e-9 {
		t.Fatal("hour 26 is hour 2 of the next day")
	}
}

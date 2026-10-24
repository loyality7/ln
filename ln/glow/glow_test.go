package glow

import "testing"

func TestLightBloomsAndNeverClips(t *testing.T) {
	c := New(80, 80)
	c.Spot(40, 40, 1.5, 40, 30, 8) // a very bright orange spot
	img := c.Image(Look{Exposure: 1, Bloom: []float64{0.05}, Strength: 0.8})
	r, g, b, _ := img.At(40, 40).RGBA()
	if r>>8 < 240 || g>>8 < 200 || b>>8 > r>>8 {
		t.Fatalf("core %d %d %d should be near white but warm", r>>8, g>>8, b>>8)
	}
	// a few pixels away the glow is a fainter orange; far away it is black
	rn, gn, bn, _ := img.At(40, 45).RGBA()
	if rn>>8 == 0 || rn <= gn || gn < bn {
		t.Fatalf("bloom %d %d %d should be a faint orange", rn>>8, gn>>8, bn>>8)
	}
	if rf, _, _, _ := img.At(2, 2).RGBA(); rf>>8 != 0 {
		t.Fatal("far from the light the canvas must stay black")
	}
}

func TestStreakIsBrighterAtItsHead(t *testing.T) {
	c := New(100, 40)
	c.Streak(10, 20, 90, 20, 1, 1, 1, 1)
	if c.Luma(88, 20) <= 3*c.Luma(30, 20) {
		t.Fatalf("head %v tail %v", c.Luma(88, 20), c.Luma(30, 20))
	}
}

// Package nature grows landscape pieces from rules and noise: terrain
// height functions, pines, rocks. Everything here is built from ln/solid.
package nature

import "math"

func hash(ix, iy int) float64 {
	n := uint32(ix)*374761393 + uint32(iy)*668265263
	n = (n ^ (n >> 13)) * 1274126177
	return float64((n^(n>>16))&0xffffff) / float64(0xffffff)
}

func fade(t float64) float64 { return t * t * t * (t*(t*6-15) + 10) }

// Noise is smooth value noise in [0,1].
func Noise(x, y float64) float64 {
	ix, iy := int(math.Floor(x)), int(math.Floor(y))
	fx, fy := fade(x-float64(ix)), fade(y-float64(iy))
	a, b := hash(ix, iy), hash(ix+1, iy)
	c, d := hash(ix, iy+1), hash(ix+1, iy+1)
	return (a+(b-a)*fx)*(1-fy) + (c+(d-c)*fx)*fy
}

// FBM layers octaves of noise into rolling hills, in [0,1].
func FBM(x, y float64, octaves int) float64 {
	var sum, amp, norm float64 = 0, 1, 0
	for i := 0; i < octaves; i++ {
		sum += amp * Noise(x, y)
		norm += amp
		x, y, amp = x*2.03+17, y*2.03+31, amp*0.5
	}
	return sum / norm
}

// Ridged is sharp-crested noise in [0,1], good for mountain ranges.
func Ridged(x, y float64, octaves int) float64 {
	var sum, amp, norm float64 = 0, 1, 0
	for i := 0; i < octaves; i++ {
		r := 1 - math.Abs(2*Noise(x, y)-1)
		sum += amp * r * r
		norm += amp
		x, y, amp = x*2.1+7, y*2.1+13, amp*0.5
	}
	return sum / norm
}

// Smoothstep eases 0..1 between e0 and e1.
func Smoothstep(e0, e1, x float64) float64 {
	t := math.Max(0, math.Min(1, (x-e0)/(e1-e0)))
	return t * t * (3 - 2*t)
}

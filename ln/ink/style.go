// Package ink renders a scene as manga-style pen-and-ink: outline strokes plus
// light-driven hatching that follows the surfaces, with depth-based line weight.
package ink

import (
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/light"
)

type Style struct {
	Light       geom.Vector   // direction from the scene toward the light
	Spacing     float64       // world distance between hatch lines (0 = auto)
	Thresholds  []float64     // darkness at which hatch family i switches on
	Dirs        []geom.Vector // hatch plane normals, one per family
	Shadows     bool          // cast shadow rays toward the light
	Scale       float64       // G-buffer resolution relative to output (0.5 = half)
	Outline     float64       // outline stroke width in px
	Hatch       float64       // hatch stroke width in px
	NearMul     float64       // width multiplier at NearDist from the eye
	FarMul      float64       // width multiplier at FarDist from the eye
	NearDist    float64       // 0 = auto from eye-center distance
	FarDist     float64
	Wobble      float64    // hand-drawn jitter amplitude in px (0 = ruler straight)
	WobbleSeed  float64    // change per frame for boiling lines
	Background  Background // sky; nil = plain paper
	Rig         *light.Rig // lighting; nil = one sun from Light with ambient
	MinGap      float64    // thinnest hatch spacing drawn, in buffer pixels (0 = default 3)
	WobbleScale float64    // how tight the hand-drawn wobble is: bigger = shorter waves (0 = 0.045)
	TaperMin    float64    // thinnest end of a tapered stroke as a fraction of full width; 1 = no taper (0 = 0.3)
	Smooth      int        // rounding passes on filled shapes; lines use one fewer (0 = 2)
	SkyBlackAt  float64    // sky darkness above which the sky is solid black (0 = 0.62)
	SkyBandFrom float64    // sky darkness above which pen-line bands start (0 = 0.18)
	SkyBandStep int        // pen-line band spacing in buffer rows (0 = 5)
	BlackAt     float64    // toon: shadow darkness at which paper turns solid black (0 = never)
	DarkBoost   float64    // toon: extra line weight on the shadow side
}

// ToonStyle is the clean manga look: full-resolution shading, solid blacks,
// brush-weight contours and only light hatching.
func ToonStyle() Style {
	st := DefaultStyle()
	st.Scale = 1
	st.Outline = 2.4
	st.Hatch = 0.8
	st.Wobble = 0.5
	st.BlackAt = 0.8
	st.DarkBoost = 1.1
	st.NearMul, st.FarMul = 1.2, 0.8
	st.Thresholds = []float64{0.55, 0.8}
	st.Dirs = st.Dirs[:2]
	return st
}

func DefaultStyle() Style {
	return Style{
		Light:      geom.Vector{X: -0.5, Y: -0.7, Z: 1.0},
		Thresholds: []float64{0.42, 0.68, 0.88},
		Dirs: []geom.Vector{
			{X: 0.8, Y: 0.3, Z: 0.5},
			{X: -0.3, Y: 0.85, Z: 0.45},
			{X: 0.5, Y: -0.5, Z: 0.7},
		},
		Shadows: true,
		Scale:   0.5,
		Outline: 2.6,
		Hatch:   1.1,
		NearMul: 1.5,
		FarMul:  0.55,
		Wobble:  1.3,
	}
}

// rig is the lighting to use: the explicit Rig, or a single sun from Light.
func (st Style) rig() light.Rig {
	if st.Rig != nil {
		return *st.Rig
	}
	r := light.Sun(st.Light)
	r.Shadows = st.Shadows
	return r
}

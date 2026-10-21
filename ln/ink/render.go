package ink

import (
	"math"

	"github.com/loyality7/ln/ln/buffer"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/scene"
)

// prepare fills in the automatic style values and returns the distance-weight function.
func prepare(st *Style, eye, center geom.Vector) func(float64) float64 {
	d := eye.Distance(center)
	if st.Spacing == 0 {
		st.Spacing = d / 90
	}
	if st.Scale == 0 {
		st.Scale = 0.5
	}
	if st.MinGap == 0 {
		st.MinGap = 3
	}
	if st.WobbleScale == 0 {
		st.WobbleScale = 0.045
	}
	if st.TaperMin == 0 {
		st.TaperMin = 0.3
	}
	if st.Smooth == 0 {
		st.Smooth = 2
	}
	if st.SkyBlackAt == 0 {
		st.SkyBlackAt = 0.62
	}
	if st.SkyBandFrom == 0 {
		st.SkyBandFrom = 0.18
	}
	if st.SkyBandStep == 0 {
		st.SkyBandStep = 5
	}
	if st.NearDist == 0 {
		st.NearDist = d * 0.6
	}
	if st.FarDist == 0 {
		st.FarDist = d * 2
	}
	st.Light = st.Light.Normalize()
	near, far, nm, fm := st.NearDist, st.FarDist, st.NearMul, st.FarMul
	return func(dist float64) float64 {
		t := math.Max(0, math.Min(1, (dist-near)/(far-near)))
		return nm + (fm-nm)*t
	}
}

// Render draws s from the given camera in the hatched pen-and-ink look.
func Render(s *scene.Scene, eye, center, up geom.Vector, width, height, fovy float64, st Style) *Drawing {
	s.Compile()
	mul := prepare(&st, eye, center)
	dr := &Drawing{W: width, H: height, mul: mul, taperMin: st.TaperMin}
	gw, gh := int(width*st.Scale), int(height*st.Scale)
	g := buffer.Gather(s, buffer.Camera{Eye: eye, Center: center, Up: up, Fovy: fovy}, gw, gh, st.rig())
	if st.Background != nil { // sky first: black fill, hatched gradient, then stars/sun/moon
		black := make([]float64, gw*gh)
		bands := skyMask(g, st.Background, black, width, height, st)
		ctx := &Ctx{W: width, H: height, Eye: eye, g: g}
		dr.addLayer(Layer{Fills: []Fill{maskFill(black, gw, gh, width/float64(gw), height/float64(gh), st.Smooth)}, Lines: bands})
		dr.addLayer(st.Background.Marks(ctx))
	}
	var layer Layer
	// hatching first so outlines draw on top
	for _, p := range hatch(g, st, width, height) {
		layer.Lines = append(layer.Lines, Line{Pts: p, Width: st.Hatch, Hatch: true})
	}
	d := eye.Distance(center)
	for _, p := range s.RenderDepth(eye, center, up, width, height, fovy, 0.1, 1000, d/400) {
		layer.Lines = append(layer.Lines, Line{Pts: p, Width: st.Outline})
	}
	dr.addLayer(layer)
	dr.wobbleAll(st.Wobble, st.WobbleSeed, st.WobbleScale)
	return dr
}

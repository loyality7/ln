package ink

import (
	"github.com/loyality7/ln/ln/buffer"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/scene"
)

// RenderColour draws the scene as a flat-colour illustration that still reads as 3D: each object is
// painted its Material.Color in three tones (lit, middle, shaded, from where the light falls), the sky in
// st.Sky, and the ink outlines of RenderToon go on top. Everything is vector shapes and lines, so the
// result can be written as SVG or as Go code like any other drawing.
func RenderColour(s *scene.Scene, eye, center, up geom.Vector, width, height, fovy float64, st Style, mats map[int]Material, overlays []Overlay) *Drawing {
	s.Compile()
	prepare(&st, eye, center) // fills the automatic style values (on this copy only)
	gw, gh := int(width*st.Scale), int(height*st.Scale)
	g := buffer.Gather(s, buffer.Camera{Eye: eye, Center: center, Up: up, Fovy: fovy}, gw, gh, st.rig())
	sx, sy := width/float64(gw), height/float64(gh)
	light, shade := st.Highlight, st.Shadow
	if light == 0 {
		light = 0.28
	}
	if shade == 0 {
		shade = 0.42
	}

	var layers []Layer
	if st.Sky[0].A > 0 && st.Background != nil {
		all, mid, dark := make([]float64, gw*gh), make([]float64, gw*gh), make([]float64, gw*gh)
		for k := range all {
			if g.OK[k] || g.IsVoid(k) {
				continue
			}
			d := st.Background.Darkness(g.Dir[k])
			all[k] = 1
			if d > 0.25 {
				mid[k] = 1
			}
			if d > 0.55 {
				dark[k] = 1
			}
		}
		for i, m := range [][]float64{all, mid, dark} {
			f := maskFill(m, gw, gh, sx, sy, st.Smooth)
			f.Color = st.Sky[i]
			layers = append(layers, Layer{Fills: []Fill{f}})
		}
	}

	// objects: base colour for the whole object, then the lit-or-middle part, then the lit part,
	// so the tones nest and nothing shows paper between them
	ids := map[int]bool{}
	for k, ok := range g.OK {
		if ok {
			id := g.ID[k]
			if id < 0 {
				id = -id
			}
			ids[id] = true
		}
	}
	for id := range ids {
		m, found := mats[id]
		if !found || m.Color.A == 0 {
			continue
		}
		all, mid, lit := make([]float64, gw*gh), make([]float64, gw*gh), make([]float64, gw*gh)
		for k, ok := range g.OK {
			if !ok {
				continue
			}
			kid := g.ID[k]
			if kid < 0 {
				kid = -kid
			}
			if kid != id {
				continue
			}
			all[k] = 1
			if g.Dark[k] < 0.62 {
				mid[k] = 1
			}
			if g.Dark[k] < 0.3 {
				lit[k] = 1
			}
		}
		for i, m2 := range [][]float64{all, mid, lit} {
			f := maskFill(m2, gw, gh, sx, sy, st.Smooth)
			switch i {
			case 0:
				f.Color = m.Color.Mix(Color{0.02, 0.02, 0.12, 1}, shade)
			case 1:
				f.Color = m.Color
			default:
				f.Color = m.Color.Mix(Color{1, 0.97, 0.88, 1}, light)
			}
			if len(f.Loops) > 0 {
				layers = append(layers, Layer{Fills: []Fill{f}})
			}
		}
	}

	// outlines and creases from the toon renderer, with its own shading, hatching and solid blacks switched off
	flat := make(map[int]Material, len(mats))
	for id, m := range mats {
		m.Tone, m.Class = false, Paper
		flat[id] = m
	}
	st.BlackAt = 0
	st.SkyBlackAt, st.SkyBandFrom = 2, 2 // no black sky or pen bands: the sky is painted above (stars and moon marks still draw)
	line := RenderToon(s, eye, center, up, width, height, fovy, st, flat, overlays)
	line.Layers = append(layers, line.Layers...)
	return line
}

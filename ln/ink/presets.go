package ink

import "sort"

var presets = map[string]func() Style{
	// ink: hatched pen drawing with cast shadows
	"ink": DefaultStyle,
	// manga: clean contours, solid blacks, light hatching
	"manga": ToonStyle,
	// sketch: loose, wobbly, heavier hatching
	"sketch": func() Style {
		st := DefaultStyle()
		st.Wobble, st.WobbleScale, st.Outline, st.Hatch, st.TaperMin = 2.2, 0.03, 2.0, 1.0, 0.15
		return st
	},
	// technical: even, precise lines with no wobble or taper
	"technical": func() Style {
		st := DefaultStyle()
		st.Wobble, st.TaperMin, st.Outline, st.Hatch, st.NearMul, st.FarMul = 0, 1, 1.4, 0.7, 1, 1
		return st
	},
}

// Preset returns a named style: "ink", "manga", "sketch" or "technical".
func Preset(name string) (Style, bool) {
	f, ok := presets[name]
	if !ok {
		return Style{}, false
	}
	return f(), true
}

// PresetNames lists the available presets.
func PresetNames() []string {
	var n []string
	for k := range presets {
		n = append(n, k)
	}
	sort.Strings(n)
	return n
}

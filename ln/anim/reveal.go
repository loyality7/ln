package anim

import "github.com/loyality7/ln/ln/geom"

// Reveal returns the first fraction p of the total line length, as if the
// pen were still drawing. Paths are drawn in order.
func Reveal(paths geom.Paths, p float64) geom.Paths {
	var total float64
	for _, path := range paths {
		for i := 1; i < len(path); i++ {
			total += path[i].Distance(path[i-1])
		}
	}
	budget := Clamp(p) * total
	var out geom.Paths
	for _, path := range paths {
		if budget <= 0 {
			break
		}
		var cut geom.Path
		for i, v := range path {
			if i == 0 {
				cut = geom.Path{v}
				continue
			}
			d := v.Distance(path[i-1])
			if d >= budget {
				if d > 0 {
					cut = append(cut, LerpVec(path[i-1], v, budget/d))
				}
				budget = 0
				break
			}
			budget -= d
			cut = append(cut, v)
		}
		if len(cut) > 1 {
			out = append(out, cut)
		}
	}
	return out
}

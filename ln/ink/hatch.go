package ink

import (
	"math"

	"github.com/loyality7/ln/ln/buffer"
	"github.com/loyality7/ln/ln/geom"
)

type seg struct{ a, b geom.Vector }

// hatch emits, per family, the iso-lines of dot(P,dir)/spacing over the visible
// surface (marching squares). Those are exactly the curves where parallel planes
// slice the surface, so lines bend around forms. A cell is drawn only where its
// darkness passes the family threshold, which makes shadows denser.
func hatch(g *buffer.Buffers, st Style, W, H float64) []geom.Path {
	var out []geom.Path
	sx, sy := W/float64(g.W), H/float64(g.H)
	f := make([]float64, g.W*g.H)
	for fam, dir := range st.Dirs {
		d := dir.Normalize()
		for k := range f {
			if g.OK[k] {
				f[k] = g.P[k].Dot(d) / st.Spacing
			}
		}
		var segs []seg
		for j := 0; j+1 < g.H; j++ {
			for i := 0; i+1 < g.W; i++ {
				c := [4]int{j*g.W + i, j*g.W + i + 1, (j+1)*g.W + i + 1, (j+1)*g.W + i}
				if !(g.OK[c[0]] && g.OK[c[1]] && g.OK[c[2]] && g.OK[c[3]]) {
					continue
				}
				mean := (g.Dark[c[0]] + g.Dark[c[1]] + g.Dark[c[2]] + g.Dark[c[3]]) / 4
				if mean < st.Thresholds[fam] {
					continue
				}
				lo, hi := f[c[0]], f[c[0]]
				tlo, thi := g.T[c[0]], g.T[c[0]]
				for _, k := range c[1:] {
					lo, hi = math.Min(lo, f[k]), math.Max(hi, f[k])
					tlo, thi = math.Min(tlo, g.T[k]), math.Max(thi, g.T[k])
				}
				if hi-lo > 1/st.MinGap {
					continue // lines would be packed tighter than the pen can draw
				}
				if thi-tlo > 0.08*tlo {
					continue // depth jump: different surfaces
				}
				if g.N[c[0]].Dot(g.N[c[1]]) < 0.92 || g.N[c[0]].Dot(g.N[c[2]]) < 0.92 || g.N[c[0]].Dot(g.N[c[3]]) < 0.92 {
					continue // crease or silhouette: normals disagree
				}
				for n := math.Floor(lo) + 1; n <= hi; n++ {
					var pts []geom.Vector
					for e := 0; e < 4; e++ {
						a, b := c[e], c[(e+1)%4]
						if (f[a] < n) == (f[b] < n) {
							continue
						}
						t := (n - f[a]) / (f[b] - f[a])
						ax, ay := float64(a%g.W), float64(a/g.W)
						bx, by := float64(b%g.W), float64(b/g.W)
						pts = append(pts, geom.Vector{
							X: (ax + (bx-ax)*t + 0.5) * sx,
							Y: (ay + (by-ay)*t + 0.5) * sy,
							Z: g.T[a] + (g.T[b]-g.T[a])*t,
						})
					}
					for q := 0; q+1 < len(pts); q += 2 {
						segs = append(segs, seg{pts[q], pts[q+1]})
					}
				}
			}
		}
		out = append(out, stitch(segs)...)
	}
	return out
}

type key [2]int64

func keyOf(v geom.Vector) key { return key{int64(math.Round(v.X * 100)), int64(math.Round(v.Y * 100))} }

// stitch joins segments that share endpoints into polylines.
func stitch(segs []seg) []geom.Path {
	at := map[key][]int{}
	for i, s := range segs {
		at[keyOf(s.a)] = append(at[keyOf(s.a)], i)
		at[keyOf(s.b)] = append(at[keyOf(s.b)], i)
	}
	used := make([]bool, len(segs))
	// next unused segment touching v, returning its far end
	next := func(v geom.Vector) (geom.Vector, bool) {
		for _, i := range at[keyOf(v)] {
			if used[i] {
				continue
			}
			used[i] = true
			if keyOf(segs[i].a) == keyOf(v) {
				return segs[i].b, true
			}
			return segs[i].a, true
		}
		return geom.Vector{}, false
	}
	var out []geom.Path
	for i, s := range segs {
		if used[i] {
			continue
		}
		used[i] = true
		path := geom.Path{s.a, s.b}
		for {
			v, ok := next(path[len(path)-1])
			if !ok {
				break
			}
			path = append(path, v)
		}
		for {
			v, ok := next(path[0])
			if !ok {
				break
			}
			path = append(geom.Path{v}, path...)
		}
		out = append(out, path)
	}
	return out
}

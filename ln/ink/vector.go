package ink

import (
	"math"
	"sort"

	"github.com/loyality7/ln/ln/geom"
)

// blur is two passes of a separable box blur of radius r; it rounds off pixel
// stair-steps before a mask is turned into a vector outline.
func blur(f []float64, w, h, r int) []float64 {
	if r <= 0 {
		return f
	}
	tmp := make([]float64, len(f))
	out := make([]float64, len(f))
	copy(out, f)
	for pass := 0; pass < 2; pass++ {
		for j := 0; j < h; j++ { // horizontal
			for i := 0; i < w; i++ {
				var s float64
				for d := -r; d <= r; d++ {
					x := min(w-1, max(0, i+d))
					s += out[j*w+x]
				}
				tmp[j*w+i] = s / float64(2*r+1)
			}
		}
		for j := 0; j < h; j++ { // vertical
			for i := 0; i < w; i++ {
				var s float64
				for d := -r; d <= r; d++ {
					y := min(h-1, max(0, j+d))
					s += tmp[y*w+i]
				}
				out[j*w+i] = s / float64(2*r+1)
			}
		}
	}
	return out
}

// trace returns the closed outlines of the region where f > level (marching
// squares). With replicate false the field is padded with zeros so every
// outline closes (needed for fills); with true the edge pixels are repeated so
// outlines run off the image instead of tracing its border. Points are scaled
// by sx, sy after offsetting by ox, oy.
func trace(f []float64, w, h int, level, sx, sy, ox, oy float64, replicate bool) []geom.Path {
	at := func(i, j int) float64 {
		if i < 0 || j < 0 || i >= w || j >= h {
			if !replicate {
				return 0
			}
			i, j = min(w-1, max(0, i)), min(h-1, max(0, j))
		}
		return f[j*w+i]
	}
	var segs []seg
	for j := -1; j < h; j++ {
		for i := -1; i < w; i++ {
			c := [4][2]int{{i, j}, {i + 1, j}, {i + 1, j + 1}, {i, j + 1}}
			var v [4]float64
			var in [4]bool
			n := 0
			for k := range c {
				v[k] = at(c[k][0], c[k][1])
				in[k] = v[k] > level
				if in[k] {
					n++
				}
			}
			if n == 0 || n == 4 {
				continue
			}
			var pt [4]geom.Vector
			var cross [4]bool
			for e := 0; e < 4; e++ {
				a, b := e, (e+1)%4
				if in[a] == in[b] {
					continue
				}
				cross[e] = true
				t := (level - v[a]) / (v[b] - v[a])
				x := float64(c[a][0]) + float64(c[b][0]-c[a][0])*t
				y := float64(c[a][1]) + float64(c[b][1]-c[a][1])*t
				pt[e] = geom.Vector{X: (x + ox + 0.5) * sx, Y: (y + oy + 0.5) * sy}
			}
			var es []int
			for e := 0; e < 4; e++ {
				if cross[e] {
					es = append(es, e)
				}
			}
			if len(es) == 2 {
				segs = append(segs, seg{pt[es[0]], pt[es[1]]})
				continue
			}
			// saddle: decide with the cell centre which corners stay joined
			centre := (v[0]+v[1]+v[2]+v[3])/4 > level
			if in[0] == centre { // cut off corners 1 and 3
				segs = append(segs, seg{pt[0], pt[1]}, seg{pt[2], pt[3]})
			} else { // cut off corners 0 and 2
				segs = append(segs, seg{pt[3], pt[0]}, seg{pt[1], pt[2]})
			}
		}
	}
	return stitch(segs)
}

// smoothLoop rounds corners with Chaikin subdivision, keeping per-vertex
// attributes (width) interpolated alongside.
func smoothLoop(p geom.Path, w []float64, iters int) (geom.Path, []float64) {
	closed := len(p) > 2 && keyOf(p[0]) == keyOf(p[len(p)-1])
	if closed {
		p = p[:len(p)-1]
		if w != nil {
			w = w[:len(w)-1]
		}
	}
	for it := 0; it < iters; it++ {
		n := len(p)
		var np geom.Path
		var nw []float64
		last := n
		if !closed {
			last = n - 1
			np = append(np, p[0])
			if w != nil {
				nw = append(nw, w[0])
			}
		}
		for i := 0; i < last; i++ {
			a, b := p[i], p[(i+1)%n]
			np = append(np, mix(a, b, 0.25), mix(a, b, 0.75))
			if w != nil {
				wa, wb := w[i], w[(i+1)%n]
				nw = append(nw, wa*0.75+wb*0.25, wa*0.25+wb*0.75)
			}
		}
		if !closed {
			np = append(np, p[n-1])
			if w != nil {
				nw = append(nw, w[n-1])
			}
		}
		p, w = np, nw
	}
	if closed {
		p = append(p, p[0])
		if w != nil {
			w = append(w, w[0])
		}
	}
	return p, w
}

func mix(a, b geom.Vector, t float64) geom.Vector { return a.Add(b.Sub(a).MulScalar(t)) }

// simplify drops points closer than eps to the line through their neighbours (Douglas-Peucker).
func simplify(p geom.Path, eps float64) geom.Path {
	if len(p) < 3 {
		return p
	}
	var rec func(lo, hi int, keep []bool)
	rec = func(lo, hi int, keep []bool) {
		a, b := p[lo], p[hi]
		best, bi := 0.0, -1
		for i := lo + 1; i < hi; i++ {
			if d := p[i].SegmentDistance(a, b); d > best {
				best, bi = d, i
			}
		}
		if bi >= 0 && best > eps {
			keep[bi] = true
			rec(lo, bi, keep)
			rec(bi, hi, keep)
		}
	}
	keep := make([]bool, len(p))
	keep[0], keep[len(p)-1] = true, true
	rec(0, len(p)-1, keep)
	var out geom.Path
	for i, k := range keep {
		if k {
			out = append(out, p[i])
		}
	}
	return out
}

var _ = math.Pi

// Curve smooths a polyline into a Catmull-Rom curve through its points,
// perSpan points between each pair.
func Curve(p geom.Path, perSpan int) geom.Path {
	at := func(i int) geom.Vector { return p[max(0, min(len(p)-1, i))] }
	var out geom.Path
	for i := 0; i+1 < len(p); i++ {
		for k := 0; k < perSpan; k++ {
			t := float64(k) / float64(perSpan)
			a, b, c, d := at(i-1), at(i), at(i+1), at(i+2)
			f := func(a, b, c, d float64) float64 {
				return 0.5 * (2*b + (-a+c)*t + (2*a-5*b+4*c-d)*t*t + (-a+3*b-3*c+d)*t*t*t)
			}
			out = append(out, geom.Vector{X: f(a.X, b.X, c.X, d.X), Y: f(a.Y, b.Y, c.Y, d.Y)})
		}
	}
	return append(out, p[len(p)-1])
}

// Ribbon turns a centre line into a filled brush stroke whose width follows
// width(t), t running 0..1 along the line: thick in the middle, a point at the
// end, whatever the pen did.
func Ribbon(center geom.Path, width func(t float64) float64) geom.Path {
	n := len(center)
	left, right := make(geom.Path, n), make(geom.Path, n)
	for i := range center {
		a, b := center[max(0, i-1)], center[min(n-1, i+1)]
		tx, ty := b.X-a.X, b.Y-a.Y
		l := math.Hypot(tx, ty)
		if l == 0 {
			l = 1
		}
		w := width(float64(i)/float64(n-1)) / 2
		left[i] = geom.Vector{X: center[i].X - ty/l*w, Y: center[i].Y + tx/l*w}
		right[i] = geom.Vector{X: center[i].X + ty/l*w, Y: center[i].Y - tx/l*w}
	}
	for i := n - 1; i >= 0; i-- {
		left = append(left, right[i])
	}
	return left
}

// IsoLines draws contour lines of a scalar field: for each level (ascending), the
// curves where the field equals it. f is w*h values, row 0 at the bottom; valid marks
// where the field exists. This is how ln draws a surface as ink: rings of constant
// radius, spokes of constant angle. Points are scaled by sx, sy.
func IsoLines(f []float64, valid []bool, w, h int, levels []float64, sx, sy float64) [][]geom.Path {
	segs := make([][]seg, len(levels))
	for j := 0; j+1 < h; j++ {
		for i := 0; i+1 < w; i++ {
			c := [4]int{j*w + i, j*w + i + 1, (j+1)*w + i + 1, (j+1)*w + i}
			if !(valid[c[0]] && valid[c[1]] && valid[c[2]] && valid[c[3]]) {
				continue
			}
			lo, hi := f[c[0]], f[c[0]]
			for _, k := range c[1:] {
				lo, hi = math.Min(lo, f[k]), math.Max(hi, f[k])
			}
			if hi-lo > 4 { // a jump between two separate surfaces, not a slope
				continue
			}
			for li := sort.SearchFloat64s(levels, lo); li < len(levels) && levels[li] <= hi; li++ {
				n := levels[li]
				var pts []geom.Vector
				for e := 0; e < 4; e++ {
					a, b := c[e], c[(e+1)%4]
					if (f[a] < n) == (f[b] < n) {
						continue
					}
					t := (n - f[a]) / (f[b] - f[a])
					ax, ay := float64(a%w), float64(a/w)
					bx, by := float64(b%w), float64(b/w)
					pts = append(pts, geom.Vector{X: (ax + (bx-ax)*t + 0.5) * sx, Y: (ay + (by-ay)*t + 0.5) * sy})
				}
				for q := 0; q+1 < len(pts); q += 2 {
					segs[li] = append(segs[li], seg{pts[q], pts[q+1]})
				}
			}
		}
	}
	out := make([][]geom.Path, len(levels))
	for li := range levels {
		out[li] = stitch(segs[li])
	}
	return out
}

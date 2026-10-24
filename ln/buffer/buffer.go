// Package buffer samples the visible world into attribute buffers: for every
// pixel, what surface is there, how far, which way it faces, which object it is
// and how dark it is. Every rendering look (ink, manga, ASCII) is a stylist that
// reads these buffers, so any shape works with any look.
package buffer

import (
	"math"
	"runtime"
	"sync"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/light"
	"github.com/loyality7/ln/ln/physics"
	"github.com/loyality7/ln/ln/scene"
	"github.com/loyality7/ln/ln/shapes"
)

type Camera struct {
	Eye, Center, Up geom.Vector
	Fovy            float64 // vertical field of view, degrees
	Near, Far       float64
	// Layers is how many surface crossings to remember per pixel (0 = just the first).
	// Only useful with a bent scene (Scene.Bend), where one ray can meet the same
	// surface twice: once directly and once bent around the mass.
	Layers int
}

// Surface is one place a camera ray met a shape.
type Surface struct {
	P, N    geom.Vector
	T       float64     // distance travelled along the ray to get there
	ID      int         // object tag
	Heading geom.Vector // unit direction the ray was travelling on arrival
}

// Matrix is the view-projection matrix for the given aspect ratio.
func (c Camera) Matrix(aspect float64) geom.Matrix {
	near, far := c.Near, c.Far
	if near == 0 {
		near = 0.1
	}
	if far == 0 {
		far = 1000
	}
	return geom.LookAt(c.Eye, c.Center, c.Up).Perspective(c.Fovy, aspect, near, far)
}

// Buffers holds one entry per pixel, row-major, row 0 at the bottom of the picture.
type Buffers struct {
	W, H     int
	Cam      Camera
	ViewProj geom.Matrix
	OK       []bool        // a surface was hit
	P        []geom.Vector // world position of the hit
	T        []float64     // distance from the eye
	N        []geom.Vector // unit normal facing the eye
	ID       []int         // object tag (0 = untagged); negative means seen in a mirror
	Refl     []bool        // the pixel is a mirror (it shows a reflection or the reflected sky)
	Dark     []float64     // 0 lit .. 1 black, from the light rig
	Dir      []geom.Vector // view ray, for sky and background shading (for a bent ray that left the scene: the direction it left along)

	// Only filled for bent scenes (Scene.Bend):
	Void    []bool        // the ray fell into the field's capture radius (a horizon)
	Exit    []geom.Vector // the direction an escaped ray left along (zero if it fell in)
	Crosses [][]Surface   // with Camera.Layers > 0: every crossing, nearest first
}

// IsVoid reports whether pixel k's ray was swallowed by the field.
func (g *Buffers) IsVoid(k int) bool { return g.Void != nil && g.Void[k] }

func Rows(h int, f func(j int)) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU())
	for j := 0; j < h; j++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(j int) {
			defer wg.Done()
			f(j)
			<-sem
		}(j)
	}
	wg.Wait()
}

// Gather ray-casts the scene once per pixel.
func Gather(s *scene.Scene, cam Camera, w, h int, rig light.Rig) *Buffers {
	s.Compile()
	n := w * h
	g := &Buffers{W: w, H: h, Cam: cam, ViewProj: cam.Matrix(float64(w) / float64(h)),
		OK: make([]bool, n), P: make([]geom.Vector, n), T: make([]float64, n), N: make([]geom.Vector, n),
		ID: make([]int, n), Refl: make([]bool, n), Dark: make([]float64, n), Dir: make([]geom.Vector, n)}
	inv := g.ViewProj.Inverse()
	known := make([]bool, n) // normal came from the shape, not an estimate
	bent := s.Bend != nil
	if bent {
		g.Void, g.Exit = make([]bool, n), make([]geom.Vector, n)
		if cam.Layers > 0 {
			g.Crosses = make([][]Surface, n)
		}
	}
	Rows(h, func(j int) {
		for i := 0; i < w; i++ {
			k := j*w + i
			ndc := geom.Vector{X: (float64(i)+0.5)/float64(w)*2 - 1, Y: (float64(j)+0.5)/float64(h)*2 - 1, Z: 1}
			dir := inv.MulPositionW(ndc).Sub(cam.Eye).Normalize()
			g.Dir[k] = dir
			var hit shapes.Hit
			var firstAt, firstHeading geom.Vector
			if bent {
				hit = shapes.NoHit
				seen := 0
				outcome, exit := s.March(geom.Ray{Origin: cam.Eye, Direction: dir}, func(h shapes.Hit, at, heading geom.Vector) bool {
					if seen == 0 {
						hit, firstAt, firstHeading = h, at, heading
					}
					seen++
					if cam.Layers > 0 {
						sf := Surface{P: at, N: h.Normal, T: h.T, Heading: heading}
						if tg, ok := h.Shape.(interface{ Tag() int }); ok {
							sf.ID = tg.Tag()
						}
						g.Crosses[k] = append(g.Crosses[k], sf)
					}
					return seen < max(1, cam.Layers)
				})
				switch outcome {
				case physics.Captured:
					g.Void[k] = true
				case physics.Escaped:
					g.Exit[k] = exit
					g.Dir[k] = exit
				}
			} else {
				hit = s.Intersect(geom.Ray{Origin: cam.Eye, Direction: dir})
			}
			if !hit.Ok() {
				continue
			}
			dist, pos, mirrored := hit.T, cam.Eye.Add(dir.MulScalar(hit.T)), false
			if bent { // the ray curved on the way: use where it really arrived
				pos, dir = firstAt, firstHeading
			}
			if m, ok := hit.Shape.(shapes.Reflector); ok && m.Reflectivity() > 0 {
				n := hit.Normal
				if n.Dot(dir) > 0 {
					n = n.MulScalar(-1)
				}
				dir = dir.Sub(n.MulScalar(2 * dir.Dot(n))).Normalize() // follow the bounce
				g.Refl[k] = true
				bounce := s.Intersect(geom.Ray{Origin: pos.Add(n.MulScalar(1e-3)), Direction: dir})
				if !bounce.Ok() {
					g.Dir[k] = dir // the mirror shows sky
					continue
				}
				pos, dist, hit, mirrored = pos.Add(dir.MulScalar(bounce.T)), dist+bounce.T, bounce, true
			}
			g.OK[k], g.T[k], g.P[k] = true, dist, pos
			if tg, ok := hit.Shape.(interface{ Tag() int }); ok {
				g.ID[k] = tg.Tag()
				if mirrored {
					g.ID[k] = -g.ID[k]
				}
			}
			if hit.Normal != (geom.Vector{}) {
				nm := hit.Normal
				if nm.Dot(dir) > 0 {
					nm = nm.MulScalar(-1)
				}
				g.N[k], known[k] = nm, true
			}
		}
	})
	Rows(h, func(j int) {
		for i := 0; i < w; i++ {
			k := j*w + i
			if !g.OK[k] {
				continue
			}
			if !known[k] {
				g.N[k] = g.estimateNormal(i, j)
			}
			g.Dark[k] = rig.Dark(s, g.P[k], g.N[k])
		}
	})
	return g
}

// estimateNormal derives a normal from neighbouring hit points, for shapes that
// do not report one. Falls back to facing the camera at depth edges.
func (g *Buffers) estimateNormal(i, j int) geom.Vector {
	k := j*g.W + i
	n := g.Dir[k].MulScalar(-1)
	if i > 0 && j > 0 && i < g.W-1 && j < g.H-1 {
		l, r, d, u := k-1, k+1, k-g.W, k+g.W
		if g.OK[l] && g.OK[r] && g.OK[d] && g.OK[u] && g.smooth(k, l, r, d, u) {
			if c := g.P[r].Sub(g.P[l]).Cross(g.P[u].Sub(g.P[d])); c.Length() > 0 {
				n = c.Normalize()
				if n.Dot(g.Dir[k]) > 0 {
					n = n.MulScalar(-1)
				}
			}
		}
	}
	return n
}

func (g *Buffers) smooth(k, l, r, d, u int) bool {
	lim := 0.08 * g.T[k]
	for _, o := range [...]int{l, r, d, u} {
		if math.Abs(g.T[o]-g.T[k]) > lim {
			return false
		}
	}
	return true
}

// Smooth reports whether pixel k has no depth jump to its four neighbours.
func (g *Buffers) Smooth(k int) bool {
	i, j := k%g.W, k/g.W
	if i == 0 || j == 0 || i == g.W-1 || j == g.H-1 {
		return false
	}
	return g.OK[k-1] && g.OK[k+1] && g.OK[k-g.W] && g.OK[k+g.W] && g.smooth(k, k-1, k+1, k-g.W, k+g.W)
}

// Project maps a world point to page pixels of a pageW x pageH picture; Z holds
// the distance from the eye.
func (g *Buffers) Project(p geom.Vector, pageW, pageH float64) geom.Vector {
	n := g.ViewProj.MulPositionW(p)
	return geom.Vector{X: (n.X + 1) / 2 * pageW, Y: (n.Y + 1) / 2 * pageH, Z: p.Distance(g.Cam.Eye)}
}

// Visible reports whether world point p is on the surface the camera sees,
// rather than hidden behind other geometry. Points off the buffer count as hidden.
func (g *Buffers) Visible(p geom.Vector) bool {
	q := g.Project(p, float64(g.W), float64(g.H))
	i, j := int(q.X), int(q.Y)
	if i < 0 || j < 0 || i >= g.W || j >= g.H {
		return false
	}
	best := math.Inf(1)
	for dj := -1; dj <= 1; dj++ {
		for di := -1; di <= 1; di++ {
			x, y := i+di, j+dj
			if x < 0 || y < 0 || x >= g.W || y >= g.H {
				continue
			}
			if k := y*g.W + x; g.OK[k] && g.T[k] < best {
				best = g.T[k]
			}
		}
	}
	return math.IsInf(best, 1) || q.Z <= best+0.008*q.Z
}

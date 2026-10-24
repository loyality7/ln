package ink

import (
	"math"

	"github.com/loyality7/ln/ln/buffer"
	"github.com/loyality7/ln/ln/geom"
)

// Background paints everything the camera sees beyond the scene: the sky.
// Darkness is its tone looking along dir (0 white paper .. 1 solid black);
// Marks draws discrete things on it (stars, sun and moon discs).
type Background interface {
	Darkness(dir geom.Vector) float64
	Marks(c *Ctx) Layer
}

// OnSky projects a world direction to page pixels and reports whether that spot
// shows sky (not a scene object) inside the picture.
func (c *Ctx) OnSky(dir geom.Vector) (geom.Vector, bool) {
	p := c.g.Project(c.Eye.Add(dir.Normalize().MulScalar(1000)), c.W, c.H)
	if p.X < 0 || p.Y < 0 || p.X >= c.W || p.Y >= c.H {
		return p, false
	}
	// directions behind the camera project mirrored; reject them
	if dir.Dot(c.g.Dir[(c.g.H/2)*c.g.W+c.g.W/2]) <= 0 {
		return p, false
	}
	k := int(p.Y/c.H*float64(c.g.H))*c.g.W + int(p.X/c.W*float64(c.g.W))
	return p, !c.g.OK[k] && !c.g.Refl[k]
}

// Focal is the camera's focal length in page pixels, to size things by angle.
func (c *Ctx) Focal() float64 { return c.H / 2 / math.Tan(c.g.Cam.Fovy*math.Pi/360) }

// skyMask marks sky pixels that should be solid black, and returns hatch bands
// for the mid-tone part of the sky (a gradient drawn as horizontal pen lines).
func skyMask(g *buffer.Buffers, bg Background, black []float64, W, H float64, st Style) []Line {
	var bands []Line
	sx, sy := W/float64(g.W), H/float64(g.H)
	step := st.SkyBandStep // pen-line spacing in grid rows
	for j := 0; j < g.H; j++ {
		for i := 0; i < g.W; i++ {
			k := j*g.W + i
			if !g.OK[k] && (g.IsVoid(k) || bg.Darkness(g.Dir[k]) >= st.SkyBlackAt) {
				black[k] = 1
			}
		}
		if j%step != 0 {
			continue
		}
		start := -1
		for i := 0; i <= g.W; i++ {
			mid := false
			var d float64
			if i < g.W {
				k := j*g.W + i
				if !g.OK[k] && !g.IsVoid(k) {
					d = bg.Darkness(g.Dir[k])
					mid = d >= st.SkyBandFrom && d < st.SkyBlackAt
				}
			}
			switch {
			case mid && start < 0:
				start = i
			case !mid && start >= 0:
				d := bg.Darkness(g.Dir[j*g.W+(start+i)/2])
				pts := geom.Path{{X: float64(start) * sx, Y: (float64(j) + 0.5) * sy}, {X: float64(i) * sx, Y: (float64(j) + 0.5) * sy}}
				bands = append(bands, Line{Pts: pts, W: []float64{0.5 + 1.6*d, 0.5 + 1.6*d}})
				start = -1
			}
		}
	}
	return bands
}

// maskFill turns a 0/1 mask into smooth filled vector outlines.
func maskFill(mask []float64, gw, gh int, sx, sy float64, smooth int) Fill {
	var fill Fill
	for _, loop := range trace(blur(mask, gw, gh, 1), gw, gh, 0.5, sx, sy, 0, 0, false) {
		if math.Abs(area2(loop)) < 12 {
			continue // speck
		}
		loop, _ = smoothLoop(loop, nil, smooth)
		fill.Loops = append(fill.Loops, simplify(loop, 0.25))
	}
	return fill
}

package ink

import (
	"math"
	"math/rand"

	"github.com/loyality7/ln/ln/geom"
)

// StrokeStyle configures pressure and taper dynamics for stroked lines.
type StrokeStyle struct {
	BaseWidth   float64 // standard line width in pixels
	PressureMin float64 // minimum relative width at stroke ends (e.g. 0.1 for sharp points)
	PressureMax float64 // maximum relative width at heavy stroke points
	EntryTaper  float64 // fraction of stroke spent ramping up (0.0 to 0.5)
	ExitTaper   float64 // fraction of stroke spent tapering off (0.0 to 0.5)
	CurveThick  float64 // extra line thickness on tight curves (nib flexing)
	Wobble      float64 // subtle hand micro-wobble amplitude
}

// FineStyle returns a crisp, delicate line style with light tapering.
func FineStyle(baseWidth float64) StrokeStyle {
	return StrokeStyle{
		BaseWidth:   baseWidth,
		PressureMin: 0.25,
		PressureMax: 1.25,
		EntryTaper:  0.08,
		ExitTaper:   0.15,
		CurveThick:  0.15,
		Wobble:      0.15,
	}
}

// DynamicStyle returns an expressive, variable-pressure stroke style with sharp end flicks.
func DynamicStyle(baseWidth float64) StrokeStyle {
	return StrokeStyle{
		BaseWidth:   baseWidth,
		PressureMin: 0.12,
		PressureMax: 1.60,
		EntryTaper:  0.12,
		ExitTaper:   0.25,
		CurveThick:  0.35,
		Wobble:      0.25,
	}
}

// BroadStyle returns a bold, swelling stroke style.
func BroadStyle(baseWidth float64) StrokeStyle {
	return StrokeStyle{
		BaseWidth:   baseWidth,
		PressureMin: 0.05,
		PressureMax: 2.20,
		EntryTaper:  0.18,
		ExitTaper:   0.38,
		CurveThick:  0.55,
		Wobble:      0.35,
	}
}

// SmoothStroke converts control points into a smooth spline with curvature-aware, pressure-varying line width.
func SmoothStroke(ctrl geom.Path, style StrokeStyle, samplesPerSegment int) (geom.Path, []float64) {
	if len(ctrl) < 2 {
		return ctrl, nil
	}
	if samplesPerSegment < 4 {
		samplesPerSegment = 12
	}

	spline := Curve(ctrl, samplesPerSegment)
	n := len(spline)
	if n < 2 {
		return spline, nil
	}

	lengths := make([]float64, n)
	totalLen := 0.0
	for i := 1; i < n; i++ {
		totalLen += spline[i].Distance(spline[i-1])
		lengths[i] = totalLen
	}
	if totalLen <= 0 {
		return spline, nil
	}

	w := make([]float64, n)
	for i := 0; i < n; i++ {
		t := lengths[i] / totalLen

		pressure := 1.0
		if t < style.EntryTaper && style.EntryTaper > 0 {
			u := t / style.EntryTaper
			pressure = style.PressureMin + (1.0-style.PressureMin)*math.Sin(u*math.Pi*0.5)
		} else if t > 1.0-style.ExitTaper && style.ExitTaper > 0 {
			u := (1.0 - t) / style.ExitTaper
			pressure = style.PressureMin + (style.PressureMax-style.PressureMin)*math.Sin(u*math.Pi*0.5)
		} else {
			denom := 1.0 - style.EntryTaper - style.ExitTaper
			if denom > 0 {
				mid := (t - style.EntryTaper) / denom
				pressure = 1.0 + 0.25*math.Sin(mid*math.Pi)
			}
		}

		if i > 0 && i < n-1 {
			v0 := spline[i].Sub(spline[i-1]).Normalize()
			v1 := spline[i+1].Sub(spline[i]).Normalize()
			dot := math.Max(-1, math.Min(1, v0.Dot(v1)))
			bend := 1.0 - dot
			pressure += bend * style.CurveThick
		}

		w[i] = style.BaseWidth * pressure
	}

	if style.Wobble > 0 {
		wobblePath(spline, style.Wobble, 1.7, 0.05)
	}

	return spline, w
}

// Pen provides a fluent, high-level vector inking interface on top of Drawing.
type Pen struct {
	dr     *Drawing
	cur    geom.Vector
	path   geom.Path
	loops  []geom.Path
	width  float64
	color  Color
	white  bool
	wobble float64
	wScale float64
	seed   float64
}

// Pen returns a fluent inking tool connected to this drawing.
func (dr *Drawing) Pen() *Pen {
	return &Pen{
		dr:     dr,
		width:  2.0,
		wScale: 0.045,
	}
}

func (p *Pen) activeLayer() *Layer {
	if len(p.dr.Layers) == 0 {
		p.dr.Layers = append(p.dr.Layers, Layer{})
	}
	return &p.dr.Layers[len(p.dr.Layers)-1]
}

// NewLayer pushes a new layer onto the drawing and directs subsequent inking to it.
func (p *Pen) NewLayer() *Pen {
	p.dr.Layers = append(p.dr.Layers, Layer{})
	return p
}

// SetWidth sets the stroke line width in pixels.
func (p *Pen) SetWidth(w float64) *Pen {
	p.width = w
	return p
}

// SetColor sets the drawing color for fills and lines.
func (p *Pen) SetColor(c Color) *Pen {
	p.color = c
	p.white = false
	return p
}

// SetWhite sets paper-white ink mode (cutouts, highlights).
func (p *Pen) SetWhite(white bool) *Pen {
	p.white = white
	return p
}

// SetWobble sets hand-drawn jitter amplitude and wave scale (0 = straight).
func (p *Pen) SetWobble(amp, scale float64) *Pen {
	p.wobble = amp
	if scale > 0 {
		p.wScale = scale
	}
	return p
}

// Seed sets the wobble randomness seed.
func (p *Pen) Seed(seed float64) *Pen {
	p.seed = seed
	return p
}

// Clear clears the current working path and loops.
func (p *Pen) Clear() *Pen {
	p.path = nil
	p.loops = nil
	return p
}

// MoveTo sets the current point without drawing.
func (p *Pen) MoveTo(x, y float64) *Pen {
	p.cur = geom.Vector{X: x, Y: y}
	p.path = geom.Path{p.cur}
	return p
}

// LineTo appends a straight line segment to (x, y).
func (p *Pen) LineTo(x, y float64) *Pen {
	p.cur = geom.Vector{X: x, Y: y}
	p.path = append(p.path, p.cur)
	return p
}

// QuadTo appends a quadratic Bezier curve to (x, y) with control point (cx, cy).
func (p *Pen) QuadTo(cx, cy, x, y float64, steps int) *Pen {
	if steps < 1 {
		steps = 16
	}
	p0 := p.cur
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		u := 1 - t
		px := u*u*p0.X + 2*u*t*cx + t*t*x
		py := u*u*p0.Y + 2*u*t*cy + t*t*y
		p.path = append(p.path, geom.Vector{X: px, Y: py})
	}
	p.cur = geom.Vector{X: x, Y: y}
	return p
}

// CubicTo appends a cubic Bezier curve to (x, y) with control points (c1x, c1y) and (c2x, c2y).
func (p *Pen) CubicTo(c1x, c1y, c2x, c2y, x, y float64, steps int) *Pen {
	if steps < 1 {
		steps = 24
	}
	curve := geom.CubicBezier(p.cur, geom.Vector{X: c1x, Y: c1y}, geom.Vector{X: c2x, Y: c2y}, geom.Vector{X: x, Y: y}, steps)
	if len(curve) > 1 {
		p.path = append(p.path, curve[1:]...)
	}
	p.cur = geom.Vector{X: x, Y: y}
	return p
}

// Spline appends a smooth Catmull-Rom curve passing through the given points.
func (p *Pen) Spline(pts ...geom.Vector) *Pen {
	if len(pts) == 0 {
		return p
	}
	all := append(geom.Path{p.cur}, pts...)
	smooth := Curve(all, 12)
	if len(smooth) > 1 {
		p.path = append(p.path, smooth[1:]...)
		p.cur = smooth[len(smooth)-1]
	}
	return p
}

// Close closes the current path by appending the start point.
func (p *Pen) Close() *Pen {
	if len(p.path) > 1 && p.path[0] != p.path[len(p.path)-1] {
		p.path = append(p.path, p.path[0])
		p.cur = p.path[0]
	}
	return p
}

// Rect sets the path to a rectangle with opposite corners (x0, y0) and (x1, y1).
func (p *Pen) Rect(x0, y0, x1, y1 float64) *Pen {
	box := geom.Box{Min: geom.Vector{X: math.Min(x0, x1), Y: math.Min(y0, y1)}, Max: geom.Vector{X: math.Max(x0, x1), Y: math.Max(y0, y1)}}
	p.path = box.Rect()
	if len(p.path) > 0 {
		p.cur = p.path[len(p.path)-1]
	}
	return p
}

// Circle sets the path to a circle of radius r around (cx, cy).
func (p *Pen) Circle(cx, cy, r float64) *Pen {
	return p.Ellipse(cx, cy, r, r)
}

// Ellipse sets the path to an ellipse with radii rx, ry around (cx, cy).
func (p *Pen) Ellipse(cx, cy, rx, ry float64) *Pen {
	p.path = geom.Ellipse(cx, cy, rx, ry, 36)
	if len(p.path) > 0 {
		p.cur = p.path[len(p.path)-1]
	}
	return p
}

// Polygon sets the path to an arbitrary polygon of points.
func (p *Pen) Polygon(pts ...geom.Vector) *Pen {
	p.path = append(geom.Path(nil), pts...)
	if len(p.path) > 0 {
		p.cur = p.path[len(p.path)-1]
	}
	return p
}

// Stroke strokes the current path with the pen's current width and color.
func (p *Pen) Stroke() *Pen {
	if len(p.path) < 2 {
		return p
	}
	linePath := append(geom.Path(nil), p.path...)
	if p.wobble > 0 {
		wobblePath(linePath, p.wobble, p.seed, p.wScale)
	}
	layer := p.activeLayer()
	layer.Lines = append(layer.Lines, Line{
		Pts:   linePath,
		Width: p.width,
		Color: p.color,
		White: p.white,
	})
	return p
}

// StrokeDynamic strokes the path using variable-pressure stroke dynamics (taper, curve flex).
func (p *Pen) StrokeDynamic(style StrokeStyle) *Pen {
	if len(p.path) < 2 {
		return p
	}
	pts, w := SmoothStroke(p.path, style, 12)
	layer := p.activeLayer()
	layer.Lines = append(layer.Lines, Line{
		Pts:   pts,
		Width: style.BaseWidth,
		W:     w,
		Color: p.color,
		White: p.white,
	})
	return p
}

// StrokeTaper strokes the current path with variable pressure tapering at both ends.
func (p *Pen) StrokeTaper(minTaper float64) *Pen {
	if len(p.path) < 2 {
		return p
	}
	linePath := append(geom.Path(nil), p.path...)
	if p.wobble > 0 {
		wobblePath(linePath, p.wobble, p.seed, p.wScale)
	}
	n := len(linePath)
	w := make([]float64, n)
	for i := range w {
		w[i] = p.width * taper(i+1, n, minTaper)
	}
	layer := p.activeLayer()
	layer.Lines = append(layer.Lines, Line{
		Pts:   linePath,
		Width: p.width,
		W:     w,
		Color: p.color,
		White: p.white,
	})
	return p
}

// Ribbon turns the current path into a filled pressure stroke whose thickness
// follows widthFunc(t), t = 0..1 along the path.
func (p *Pen) Ribbon(widthFunc func(t float64) float64) *Pen {
	if len(p.path) < 2 {
		return p
	}
	poly := Ribbon(p.path, widthFunc)
	layer := p.activeLayer()
	layer.Fills = append(layer.Fills, Fill{
		Loops: []geom.Path{poly},
		Color: p.color,
		White: p.white,
	})
	return p
}

// Fill fills the current path with solid ink.
func (p *Pen) Fill() *Pen {
	if len(p.path) < 3 && len(p.loops) == 0 {
		return p
	}
	loops := p.loops
	if len(p.path) >= 3 {
		loop := append(geom.Path(nil), p.path...)
		if loop[0] != loop[len(loop)-1] {
			loop = append(loop, loop[0])
		}
		if p.wobble > 0 {
			wobblePath(loop, p.wobble, p.seed, p.wScale)
		}
		loops = append(loops, loop)
	}
	layer := p.activeLayer()
	layer.Fills = append(layer.Fills, Fill{
		Loops: loops,
		Color: p.color,
		White: p.white,
	})
	p.loops = nil
	return p
}

// FillWhite fills the current path with paper-white ink (cutout / highlight).
func (p *Pen) FillWhite() *Pen {
	oldWhite := p.white
	p.white = true
	p.Fill()
	p.white = oldWhite
	return p
}

// Hatch draws parallel hatching lines across the bounding box of the current path.
func (p *Pen) Hatch(spacing, angleDeg, weight float64) *Pen {
	if len(p.path) < 2 {
		return p
	}
	box := p.path.BoundingBox()
	lines := HatchBox(box, spacing, angleDeg, weight)
	layer := p.activeLayer()
	for _, l := range lines {
		l.Color = p.color
		l.White = p.white
		layer.Lines = append(layer.Lines, l)
	}
	return p
}

// CrossHatch draws two sets of perpendicular hatching lines across the path's bounding box.
func (p *Pen) CrossHatch(spacing, angleDeg, weight float64) *Pen {
	p.Hatch(spacing, angleDeg, weight)
	p.Hatch(spacing, angleDeg+90, weight)
	return p
}

// Feather draws fine tapered feathering strokes along the edge from p0 to p1.
func (p *Pen) Feather(p0, p1 geom.Vector, count int, length float64, angleDeg float64) *Pen {
	if count < 2 {
		count = 6
	}
	rad := angleDeg * math.Pi / 180
	dx, dy := math.Cos(rad)*length, math.Sin(rad)*length
	layer := p.activeLayer()

	for i := 0; i < count; i++ {
		t := float64(i) / float64(count-1)
		orig := geom.Vector{X: p0.X + (p1.X-p0.X)*t, Y: p0.Y + (p1.Y-p0.Y)*t}
		dest := geom.Vector{X: orig.X + dx, Y: orig.Y + dy}
		pts := geom.Path{orig, dest}
		layer.Lines = append(layer.Lines, Line{
			Pts:   pts,
			Width: p.width,
			W:     []float64{p.width, p.width * 0.2},
			Color: p.color,
			White: p.white,
		})
	}
	return p
}

// RadialLines draws radiating lines toward a focal center between radii r0 and r1.
func (p *Pen) RadialLines(center geom.Vector, r0, r1 float64, count int, seed int64) *Pen {
	r := rand.New(rand.NewSource(seed))
	layer := p.activeLayer()
	for i := 0; i < count; i++ {
		angle := float64(i)*2*math.Pi/float64(count) + (r.Float64()-0.5)*0.08
		inner := r0 * (0.85 + 0.3*r.Float64())
		outer := r1 * (0.95 + 0.1*r.Float64())

		p0 := geom.Vector{X: center.X + outer*math.Cos(angle), Y: center.Y + outer*math.Sin(angle)}
		p1 := geom.Vector{X: center.X + inner*math.Cos(angle), Y: center.Y + inner*math.Sin(angle)}

		w0 := p.width * (0.8 + 1.2*r.Float64())
		w1 := w0 * 0.15

		layer.Lines = append(layer.Lines, Line{
			Pts:   geom.Path{p0, p1},
			Width: w0,
			W:     []float64{w0, w1},
			Color: p.color,
			White: p.white,
		})
	}
	return p
}

// HalftoneGrid generates a dot matrix across a box for print tones and shading.
// density is 0.0 (white) to 1.0 (black), pitch is dot spacing, angleDeg is rotation angle.
func HalftoneGrid(box geom.Box, density float64, pitch float64, angleDeg float64) []Fill {
	if density <= 0.02 || pitch <= 1.0 {
		return nil
	}
	density = math.Min(0.95, density)

	maxR := pitch * 0.5
	r := math.Sqrt(density/math.Pi) * pitch
	if r > maxR {
		r = maxR
	}

	rad := angleDeg * math.Pi / 180
	cos, sin := math.Cos(rad), math.Sin(rad)

	diag := math.Hypot(box.Max.X-box.Min.X, box.Max.Y-box.Min.Y)
	cx, cy := (box.Min.X+box.Max.X)*0.5, (box.Min.Y+box.Max.Y)*0.5

	steps := int(math.Ceil(diag / pitch))
	var fills []Fill

	for j := -steps; j <= steps; j++ {
		for i := -steps; i <= steps; i++ {
			gx := float64(i) * pitch
			gy := float64(j) * pitch

			px := cx + gx*cos - gy*sin
			py := cy + gx*sin + gy*cos

			if px >= box.Min.X && px <= box.Max.X && py >= box.Min.Y && py <= box.Max.Y {
				circle := geom.Circle(px, py, r, 10)
				fills = append(fills, Fill{Loops: []geom.Path{circle}})
			}
		}
	}
	return fills
}

// WovenHatch generates woven alternating clusters of parallel tapered pen lines.
func WovenHatch(box geom.Box, cellSize float64, linesPerCell int, weight float64) []Line {
	if cellSize <= 0 || linesPerCell < 2 {
		return nil
	}
	var out []Line
	w := box.Max.X - box.Min.X
	h := box.Max.Y - box.Min.Y

	cols := int(math.Ceil(w / cellSize))
	rows := int(math.Ceil(h / cellSize))

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			x0 := box.Min.X + float64(c)*cellSize
			y0 := box.Min.Y + float64(r)*cellSize
			x1 := math.Min(box.Max.X, x0+cellSize)
			y1 := math.Min(box.Max.Y, y0+cellSize)

			horizontal := (r+c)%2 == 0
			if horizontal {
				step := (y1 - y0) / float64(linesPerCell)
				for k := 0; k < linesPerCell; k++ {
					y := y0 + (float64(k)+0.5)*step
					out = append(out, Line{
						Pts:   geom.Path{{X: x0, Y: y}, {X: x1, Y: y}},
						Width: weight,
						W:     []float64{weight * 0.4, weight, weight * 0.3},
					})
				}
			} else {
				step := (x1 - x0) / float64(linesPerCell)
				for k := 0; k < linesPerCell; k++ {
					x := x0 + (float64(k)+0.5)*step
					out = append(out, Line{
						Pts:   geom.Path{{X: x, Y: y0}, {X: x, Y: y1}},
						Width: weight,
						W:     []float64{weight * 0.4, weight, weight * 0.3},
					})
				}
			}
		}
	}
	return out
}

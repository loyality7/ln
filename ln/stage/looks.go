package stage

import (
	"math"

	"github.com/loyality7/ln/ln/anim"
	"github.com/loyality7/ln/ln/ascii"
	"github.com/loyality7/ln/ln/ink"
)

func boil(style ink.Style, fps, t float64) ink.Style {
	if fps > 0 { // hand-drawn lines re-jitter a few times a second
		style.WobbleSeed = math.Floor(t*fps) * 1.7
	}
	return style
}

// InkLook is the hatched pen-and-ink drawing: outlines, form-following hatching,
// cast shadows, line weight that thins with distance.
type InkLook struct {
	Style   ink.Style
	BoilFPS float64 // how often the hand-drawn jitter changes (0 = frozen)
}

func (l *InkLook) UsesSilhouettes() bool { return true }

func (l *InkLook) Render(sh Shot) anim.Frame {
	st := boil(l.Style, l.BoilFPS, sh.Time)
	st.Rig, st.Background = &sh.Rig, sh.Sky
	return ink.Render(sh.Scene, sh.Cam.Eye, sh.Cam.Center, sh.Cam.Up, sh.W, sh.H, sh.Cam.Fovy, st)
}

// ToonLook is clean manga ink: solid blacks, brush-weight contours, light
// hatching, plus any overlays (marks, highlights).
type ToonLook struct {
	Style   ink.Style
	BoilFPS float64
}

func NewToonLook() *ToonLook { return &ToonLook{Style: ink.ToonStyle(), BoilFPS: 8} }

func (l *ToonLook) Render(sh Shot) anim.Frame {
	st := boil(l.Style, l.BoilFPS, sh.Time)
	st.Rig, st.Background = &sh.Rig, sh.Sky
	return ink.RenderToon(sh.Scene, sh.Cam.Eye, sh.Cam.Center, sh.Cam.Up, sh.W, sh.H, sh.Cam.Fovy, st, sh.Mats, sh.Overlays)
}

// ColourLook is a flat-colour illustration that still reads as 3D: every object is painted its
// colour (Node.Paint) in three tones from the light, with ink outlines over it, as vector shapes.
type ColourLook struct {
	Style   ink.Style
	BoilFPS float64
}

// NewColourLook starts from the clean toon style, so lines are brush-weight and hatching is light.
func NewColourLook() *ColourLook { return &ColourLook{Style: ink.ToonStyle(), BoilFPS: 8} }

func (l *ColourLook) Render(sh Shot) anim.Frame {
	st := boil(l.Style, l.BoilFPS, sh.Time)
	st.Rig, st.Background = &sh.Rig, sh.Sky
	return ink.RenderColour(sh.Scene, sh.Cam.Eye, sh.Cam.Center, sh.Cam.Up, sh.W, sh.H, sh.Cam.Fovy, st, sh.Mats, sh.Overlays)
}

// PlotterLook draws the scene the way a pen plotter would: only the lines of the shapes themselves (outlines, creases,
// silhouettes), with every line hidden behind something removed. The result is vector paths, so it can be saved as SVG.
type PlotterLook struct {
	Step float64 // how finely each line is tested for being hidden, in world units (0 = the eye distance / 300)
}

func (l *PlotterLook) UsesSilhouettes() bool { return true }

func (l *PlotterLook) Render(sh Shot) anim.Frame {
	step := l.Step
	if step == 0 {
		step = sh.Cam.Eye.Distance(sh.Cam.Center) / 300
	}
	return sh.Scene.Render(sh.Cam.Eye, sh.Cam.Center, sh.Cam.Up, sh.W, sh.H, sh.Cam.Fovy, 0.1, 1000, step)
}

// ASCIILook draws the scene as text characters.
type ASCIILook struct{ Options ascii.Options }

func NewASCIILook(cols int) *ASCIILook {
	return &ASCIILook{ascii.Options{Cols: cols, Edges: true}}
}

func (l *ASCIILook) Render(sh Shot) anim.Frame {
	o := l.Options
	o.Aspect = sh.W / sh.H
	o.Sky = sh.Sky
	return ascii.Draw(sh.Scene, sh.Cam, sh.Rig, o)
}

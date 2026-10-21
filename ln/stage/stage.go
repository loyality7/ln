package stage

import (
	"math"
	"os"
	"path/filepath"

	"github.com/fogleman/gg"
	"github.com/loyality7/ln/ln/anim"
	"github.com/loyality7/ln/ln/buffer"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/ink"
	"github.com/loyality7/ln/ln/light"
	"github.com/loyality7/ln/ln/model"
	"github.com/loyality7/ln/ln/scene"
	"github.com/loyality7/ln/ln/shapes"
)

// Camera is a pair of nodes, so it can be moved like anything else and can be
// parented: put Target under a character and the camera follows them.
type Camera struct {
	Eye, Target *Node
	Fovy        float64
}

// Shot is everything a look needs to draw one frame.
type Shot struct {
	Scene    *scene.Scene
	Cam      buffer.Camera
	Rig      light.Rig      // sun, moon and lamps for this moment
	Sky      ink.Background // the sky for this moment
	Mats     map[int]ink.Material
	Overlays []ink.Overlay
	W, H     float64
	Time     float64
}

// Look is how a shot becomes a picture: hatched ink, clean manga ink, ASCII
// text. Looks are interchangeable; the scene and the timeline never know which is used.
type Look interface {
	Render(Shot) anim.Frame
}

// silhouetteUser is implemented by looks that want shapes to supply their own
// silhouette lines (the hatched ink look); the others derive contours from buffers.
type silhouetteUser interface{ UsesSilhouettes() bool }

// Atmosphere is the world's light and sky at a moment: where the lights are
// and what the sky looks like. Anything that implements it can drive a scene;
// components/env is a time-of-day implementation and Studio is the plain default.
type Atmosphere interface {
	ink.Background
	Rig(extra ...light.Light) light.Rig
}

// Studio is even, fixed lighting on plain paper.
type Studio struct{ Lights light.Rig }

func (s Studio) Rig(extra ...light.Light) light.Rig {
	r := s.Lights
	r.Lights = append(append([]light.Light{}, r.Lights...), extra...)
	return r
}
func (Studio) Darkness(geom.Vector) float64 { return 0 }
func (Studio) Marks(*ink.Ctx) ink.Layer     { return ink.Layer{} }

type Stage struct {
	W, H, FPS  float64
	Root       *Node
	Cam        *Camera
	Atmosphere Atmosphere // lights and sky; swap it for a time-of-day model
	Look       Look
	Overlays   []ink.Overlay // vector art tied to the 3D scene (faces, highlights)
	TL         *Timeline
}

func New(w, h, fps float64) *Stage {
	cam := &Camera{Eye: NewNode("eye").At(10, -12, 6), Target: NewNode("target"), Fovy: 36}
	s := &Stage{W: w, H: h, FPS: fps, Root: NewNode("root"), Cam: cam, Atmosphere: Studio{light.Sun(geom.Vector{X: -0.5, Y: -0.6, Z: 1})}, Look: &InkLook{Style: ink.DefaultStyle(), BoilFPS: 12}}
	s.TL = newTimeline(cam)
	return s
}

// instance is a placed copy of a shared mesh: ray-casts against world-space
// triangles but draws the base mesh's outline through its transform.
type instance struct {
	*model.Mesh
	n          *Node
	world      geom.Matrix
	eye        geom.Vector // camera in the mesh's local space
	silhouette bool
}

func (i instance) Paths() geom.Paths {
	o := outlineFor(i.n.mesh, i.n.crease)
	if i.silhouette {
		return o.PathsFrom(i.eye).Transform(i.world)
	}
	return o.Creases().Transform(i.world)
}

func bake(m *model.Mesh, w geom.Matrix, id int) *model.Mesh {
	nm := w.Inverse().Transpose() // normals move by the inverse transpose
	tris := make([]*shapes.Triangle, len(m.Triangles))
	for i, t := range m.Triangles {
		c := shapes.NewTriangle(w.MulPosition(t.V1), w.MulPosition(t.V2), w.MulPosition(t.V3))
		c.ID = id
		if t.Smooth {
			c.Smooth = true
			for k := range t.N {
				c.N[k] = nm.MulDirection(t.N[k])
			}
		}
		tris[i] = c
	}
	return model.NewMesh(tris)
}

// Shot evaluates the timeline at sec and assembles everything for that moment.
func (s *Stage) Shot(sec float64) Shot {
	s.TL.apply(sec)
	sc := &scene.Scene{}
	eye := s.Cam.Eye.WorldPos()
	silhouettes := false
	if u, ok := s.Look.(silhouetteUser); ok {
		silhouettes = u.UsesSilhouettes()
	}
	mats := map[int]ink.Material{}
	var lamps []light.Light
	s.Root.walk(func(n *Node) {
		if !n.shown() {
			return
		}
		if n.emit != nil {
			lamps = append(lamps, light.Point{Pos: n.WorldPos(), Power: n.emit.power, Radius: n.emit.radius, Size: n.emit.size})
		}
		if n.shape != nil {
			sc.Add(shapes.NewTransformedShape(n.shape, n.World()))
			return
		}
		if n.mesh == nil {
			return
		}
		w := n.World()
		if math.Abs(w.Determinant()) < 1e-12 {
			return
		}
		id, mat := n.tag()
		if !n.cacheOK || n.cacheW != w || n.cacheID != id { // unmoved nodes keep their baked mesh (and its BVH)
			n.cacheM, n.cacheW, n.cacheOK, n.cacheID = bake(n.mesh, w, id), w, true, id
		}
		mats[id] = mat
		if n.crease < 0 {
			sc.Add(shapes.NoOutline{Shape: n.cacheM})
			return
		}
		sc.Add(instance{n.cacheM, n, w, w.Inverse().MulPosition(eye), silhouettes})
	})
	return Shot{
		Scene:    sc,
		Cam:      buffer.Camera{Eye: eye, Center: s.Cam.Target.WorldPos(), Up: geom.Vector{Z: 1}, Fovy: s.Cam.Fovy},
		Rig:      s.Atmosphere.Rig(lamps...),
		Sky:      s.Atmosphere,
		Mats:     mats,
		Overlays: s.Overlays,
		W:        s.W, H: s.H, Time: sec,
	}
}

// Probe runs fn with the scene posed at another moment, then restores the
// current one. Use it to read where things were a moment ago (afterimages,
// motion trails) without disturbing the frame being drawn.
func (s *Stage) Probe(sec float64, fn func()) {
	now := s.TL.Now
	s.TL.apply(sec)
	fn()
	s.TL.apply(now)
}

// Frame renders the picture at sec.
func (s *Stage) Frame(sec float64) anim.Frame { return s.Look.Render(s.Shot(sec)) }

// Still writes one frame as a PNG.
func (s *Stage) Still(path string, sec float64) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return gg.SavePNG(path, s.Frame(sec).Image(s.W, s.H))
}

// Frames returns how many video frames the timeline needs.
func (s *Stage) Frames() int { return int(math.Ceil(s.TL.Duration() * s.FPS)) }

// Video renders the whole timeline to .mp4, .webm or .gif (needs ffmpeg).
func (s *Stage) Video(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	n := s.Frames()
	return anim.FrameAnimation(func(t float64) anim.Frame {
		return s.Frame(t * float64(n) / s.FPS)
	}).WriteVideo(path, n, s.FPS, s.W, s.H)
}

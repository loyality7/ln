# ln/stage

The engine layer: a scene graph of nodes, a timeline that scripts motion in seconds, interchangeable looks, and stills/video from one object. Imports `anim`, `ascii`, `buffer`, `geom`, `ink`, `light`, `model`, `scene`, `shapes`, `solid`. Source: `ln/stage/*.go`.

## Stage (`stage.go`)

```go
type Stage struct {
	W, H, FPS  float64
	Root       *Node
	Cam        *Camera
	Atmosphere Atmosphere     // lights and sky
	Look       Look
	Overlays   []ink.Overlay  // vector art tied to the 3D scene
	TL         *Timeline
}
func New(w, h, fps float64) *Stage
```

`New` creates: camera eye at (10, -12, 6) looking at a target node at the origin, `Fovy` 36 degrees, `Atmosphere = Studio{light.Sun({-0.5,-0.6,1})}`, `Look = &InkLook{Style: ink.DefaultStyle(), BoilFPS: 12}`, and an empty timeline.

| Method | Meaning |
|---|---|
| `Shot(sec float64) Shot` | Applies the timeline at `sec`, then walks the node tree. Hidden or zero-determinant nodes are skipped; lamps (`Emit`) become `light.Point`s; analytic shapes are added through `NewTransformedShape`; meshes are baked into world space (a node that has not moved reuses its baked mesh and BVH) |
| `Frame(sec) anim.Frame` | `Look.Render(Shot(sec))` |
| `Still(path string, sec float64) error` | Creates the directory and writes one PNG |
| `Frames() int` | `ceil(TL.Duration() * FPS)` |
| `Video(path string) error` | Renders `Frames()` frames, calling `Frame(t * n / FPS)` for each, through `anim.FrameAnimation.WriteVideo` (`.mp4`, `.webm`, `.gif`; needs ffmpeg) |
| `Probe(sec float64, fn func())` | Poses the scene at `sec`, runs `fn`, restores the current time. Use it to read where things were a moment ago (afterimages, trails) |

```go
type Camera struct { Eye, Target *Node; Fovy float64 }   // two nodes, so they can be parented or animated
type Shot struct {
	Scene *scene.Scene; Cam buffer.Camera; Rig light.Rig; Sky ink.Background
	Mats map[int]ink.Material; Overlays []ink.Overlay; W, H, Time float64
}
type Look interface{ Render(Shot) anim.Frame }
type Atmosphere interface { ink.Background; Rig(extra ...light.Light) light.Rig }
type Studio struct{ Lights light.Rig }   // fixed lighting on plain paper
```

`Studio` implements `Atmosphere`: `Rig(extra...)` appends lamps to its lights, `Darkness` returns 0 and `Marks` returns an empty layer. `components/env` provides a time-of-day atmosphere.

## Looks (`looks.go`)

| Type | Constructor | What it does |
|---|---|---|
| `InkLook{Style ink.Style; BoilFPS float64}` | (literal) | Hatched pen-and-ink via `ink.Render`. `UsesSilhouettes()` is true, so shapes supply their own silhouette lines. `BoilFPS` sets how often hand-drawn jitter changes (0 freezes it) |
| `ToonLook{Style, BoilFPS}` | `NewToonLook()` (style `ink.ToonStyle()`, boil 8) | Manga look via `ink.RenderToon`, using the shot's materials and overlays |
| `ASCIILook{Options ascii.Options}` | `NewASCIILook(cols int)` (edges on) | Text rendering via `ascii.Draw` |

Every look receives `Shot.Rig` and `Shot.Sky`, so you can swap looks without touching the scene. The "boil" works by setting `Style.WobbleSeed = floor(t*fps) * 1.7`.

## Node (`node.go`, `sugar.go`)

```go
type Node struct {
	Name string; Pos, Rot, Scl geom.Vector; Parent *Node; Children []*Node
	// unexported: mesh, shape, tag, merge, material, emitter, crease, visibility, caches
}
func NewNode(name string) *Node   // Scl (1,1,1), crease 35 degrees, default ink material
```

`Rot` is Euler angles in radians applied X then Y then Z; `Local()` negates them because `geom.Rotate` turns clockwise. `World()` is `Parent.World().Mul(Local())`; `WorldPos()` is the origin transformed by it.

Building the graph (each returns the node it creates or acts on, so calls chain):

| Method | Meaning |
|---|---|
| `Group(name)` | Empty child (pivot or container) |
| `Add(m *model.Mesh)` | Child that draws the mesh; gets a unique object id |
| `Place(s shapes.Shape)` | Child holding an analytic shape (`shapes.Water`, a plane...) |
| `Reparent(p)` | Moves under `p`, keeping the local transform |
| `At(x,y,z)`, `Turn(x,y,z)`, `Size(x,y,z)`, `Uniform(s)`, `Named(name)` | Set position, rotation, scale, name |
| `NoOutline()`, `Crease(deg)` | No outline, or the crease angle for outlines |
| `Merge()` | Draws this node and all descendants as one object: one outline, no seams between parts |
| `Ink(m ink.Material)` | How the manga look draws it |
| `Emit(power, radius)` | Makes the node a point lamp (half power at `radius`) |
| `Hide()` | Hidden until shown by the timeline (`shown()` also needs `vis >= 0.5` up the chain) |

Shape shortcuts on a node (all `solid` meshes): `Box(sx,sy,sz)`, `Sphere(r)`, `Ellipsoid(rx,ry,rz)`, `Cylinder(r,h)`, `Cone(r,h)`, `Torus(big,small)`, `Capsule(r,length)`. `Surface(m)` adds a big shaded surface that is outlined at the sky but has no crease lines.

Outline edges are expensive to weld, so they are cached per mesh and crease angle.

## Timeline (`timeline.go`, `sugar.go`)

`type Ease = anim.Ease`. Actions run one after another from a cursor; `Par` runs actions together, `Seq` makes a lane, `At` places absolutely. Sampling is stateless, so any time can be rendered in any order. `Now` is the time last applied.

| Method | Meaning |
|---|---|
| `Duration()`, `Cursor()` | End of the last action; where the next one starts |
| `Wait(dur)` | Pause |
| `Par(fn)`, `Seq(fn)`, `At(t, fn)` | Concurrency, lanes, absolute time (see source comments) |
| `Move(n, pos, dur, e)`, `Turn(n, rot, dur, e)`, `Scale(n, scl, dur, e)` | Tween a property from where it last was to the target |
| `Follow(n, f func(u float64) geom.Vector, dur, e)` | Moves along `f(u)` using about 48 samples per second |
| `Show(n)`, `Hide(n)` | Visibility switch at the cursor |
| `Float(p *float64, to, dur, e)` | Animates any float you point at (FOV, hour of day...) |
| `Fov(deg, dur, e)` | Animates the camera's `Fovy` |
| `Do(dur, fn func(p float64))` | Calls `fn(p)` every frame while active, `p` 0 to 1 then stays 1. The hook for custom motion |
| `Orbit(eye *Node, center geom.Vector, radius, height, from, to, dur float64, e Ease)` | In `sugar.go`: swings a node around a point, angles in radians |

## Example

Compiled against the API above:

```go
st := stage.New(640, 360, 30)
cube := st.Root.Box(2, 2, 2)
st.TL.Move(cube, geom.V(3, 0, 0), 1, anim.InOutSine)
st.TL.Orbit(st.Cam.Eye, geom.V(0, 0, 0), 12, 5, 0, math.Pi, 2, nil)
st.Still("out/frame.png", 1.0)
st.Video("out/scene.mp4")
```

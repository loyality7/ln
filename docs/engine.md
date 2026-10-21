# Engine guide

The core is `ln/`: only machinery that makes and draws shapes. Specific things (trees,
houses, a sky model, a face) live in `components/` and are built on the core.

| Layer | Package | Job |
|---|---|---|
| Shape forge | `ln/solid` | Build any 3D shape from code |
| Lights | `ln/light` | Sun, point and spot lights, ambient, shadows |
| Buffers | `ln/buffer` | The visible world as per-pixel depth, normal, object id, darkness |
| Looks | `ln/ink`, `ln/ascii` | Turn buffers into hatched ink, manga ink or ASCII text |
| Stage | `ln/stage` | Scene graph, timeline in seconds, atmosphere, PNG / video |

A frame is: scene graph -> shapes -> buffers (lit by the atmosphere's lights) -> look -> picture.
Each arrow is an interface, so a new look or a new atmosphere plugs in without touching the rest.

## 1. Make shapes (`solid`)

```go
solid.Box(sx, sy, sz)          solid.Sphere(r)        solid.Ellipsoid(rx, ry, rz)
solid.Cylinder(r, h)           solid.Cone(r, h)       solid.Torus(R, r)
solid.Capsule(r, length)

solid.Lathe(profile, segments) // spin a curve around z: vases, bottles, wheels
solid.Smooth(points, perSpan)  // curve through control points, for Lathe profiles
solid.Extrude(polygon, h)      // any outline (convex or not) lifted into a solid
solid.Tube(path, radius, sides, closed)   // pipe along a 3D curve, knots, vines
solid.Surface(nu, nv, f)       // any f(u,v) -> point becomes a surface
solid.Heightfield(sx, sy, nx, ny, f)      // z = f(x,y)
solid.Merge(a, b, ...)         solid.Place(mesh, matrix)
```

All of them return a `*model.Mesh`. Outlines are automatic: borders, creases sharper
than 35 degrees, and the silhouette from the camera.

## 2. Build a scene and animate it (`stage`)

```go
st := stage.New(1280, 720, 24)            // width, height, fps
ball := st.Root.Add(solid.Sphere(1)).At(0, 0, 1)
arm  := body.Group("arm").At(0.6, 0, 1)   // a pivot; children turn with it
arm.Add(solid.Box(0.2, 0.2, 0.9)).At(0, 0, -0.45)

st.Cam.Eye.At(8, -10, 4)                  // camera is two nodes: Eye and Target
st.Cam.Target.Reparent(ball)              // parent it and the camera follows
st.Atmosphere = env.At(17.5)              // lights and sky; see Atmosphere below
```

Node: `Pos`, `Rot` (radians, X then Y then Z), `Scl`, `Parent`, `Children`.
Helpers: `At`, `Turn`, `Size`, `Uniform`, `NoOutline`, `Crease`, `Hide`, `Reparent`.

### Timeline

Actions run one after another from a cursor. Time is in seconds.

```go
tl := st.TL
tl.Move(ball, v(4, 0, 1), 2, anim.OutBounce)   // glide to a point
tl.Turn(arm, v(1.2, 0, 0), 0.5, anim.InOutSine)
tl.Scale(ball, v(1, 1, 0.6), 0.1, anim.OutQuad)
tl.Follow(ball, spline, 3, anim.Linear)        // along f(u), u in [0,1]
tl.Wait(1)
tl.Show(node); tl.Hide(node)
tl.Fov(25, 2, anim.InOutSine)

tl.Par(func() { ... })   // everything inside starts together
tl.Seq(func() { ... })   // one lane of sequential actions (use inside Par)
tl.At(5, func() { ... }) // place actions at an absolute time

tl.Do(dur, func(p float64) { ... })   // any custom motion, p runs 0..1
```

`Do` is the escape hatch. Set node properties as a pure function of `p` (walk cycles,
wobble, physics). Sampling is stateless, so any time can be rendered in any order.

Easing curves live in `anim`: `Linear`, `InQuad`, `OutQuad`, `InOutQuad`, `InCubic`,
`OutCubic`, `InOutCubic`, `InOutSine`, `OutBack`, `OutElastic`, `OutBounce`.

### Atmosphere, lights and looks

`st.Atmosphere` is anything with a light rig and a sky. The default is `stage.Studio`
(fixed lighting, plain paper). `components/env` models time of day:

```go
sky := env.At(9)                       // hour 0-24, latitude, season, moon phase, clouds
st.Atmosphere = sky
tl.Float(&sky.Hour, 29, 8, anim.Linear) // sunrise to the next dawn; shadows follow the sun
lamp.Emit(1.6, 4.5)                    // any node becomes a point light
```

`tl.Float(&x, to, dur, ease)` animates any number: the hour, the field of view, a light level.

`st.Look` chooses how a frame is drawn, without touching the scene:
`&stage.InkLook{...}` (hatched), `stage.NewToonLook()` (clean manga ink, solid blacks,
overlays) and `stage.NewASCIILook(cols)` (text). Per-object ink materials:
`node.Ink(ink.Material{Class: ink.Black})`.

### Characters and motion (`rig`, `fx`)

Nothing about a character is code. A skeleton is a list of joints, a pose is named channel
values, and a move is a table of poses or oscillators:

```go
skel  := rig.Build(s.Root, spec, 1.8, 0, 1)          // spec = joints and limbs as data
actor := rig.NewActor(s.TL, skel)

punch := rig.Keys(0.9,                                 // poses at moments of the move
	rig.K(0,   pose{"shoulderN": 72, "elbowN": 112}),
	rig.K(0.3, pose{"shoulderN": -28, "elbowN": 130}), // wind up
	rig.K(0.46, pose{"shoulderN": 90, "elbowN": 6})).  // strike
	Travel(0.35, 0, 0.5).                              // lunge forward
	Mark(rig.Mark{Kind: "impact", At: 0.5, X: 1.17, Z: 1.28})

walk := rig.Cycle(2.8, 2.7, rest,                      // a repeating move: oscillators
	rig.Osc{Ch: "hipN", Amp: 30}, rig.Osc{Ch: "hipF", Amp: -30}).Travel(2.6, 0, 1)

flip := someKeys.Spin("pelvis", 1, 0.17, 0.83).Arc(1.5, 0.17, 0.83) // a somersault, in the air

actor.Play(walk); actor.Play(punch)                     // one after another, blended
rig.Beat(s.TL, rig.Moves(a, punch), rig.Moves(b, block, hit)) // two actors at once
fx.Install(s.Stage, fx.DefaultFollow(), a, b)           // effects and follow camera from the marks
```

Channels: `"knee"` is a joint's angle (degrees in `Keys`/`Cycle`), `"pelvis.z"` offsets a joint,
`"root.x"`, `"root.z"`, `"root.yaw"` move the whole skeleton. Modifiers: `Arc` (jump), `Spin`
(flip, pirouette), `Travel`, `Mark`. Mark kinds: `dust`, `impact`, `speed`, `trail`.
`rig.Freeze(dur)` holds a pose (a fighter who stays down). `components/stickman` is a complete
example: one skeleton spec and about 25 moves, all tables.

### Output

```go
st.Still("out/frame.png", 2.5)   // one picture at t = 2.5 s
st.Video("out/film.mp4")         // .mp4 / .webm / .gif, needs ffmpeg on PATH
```

Tune a look through its own `Style` (for `InkLook`, `BoilFPS` is how often the hand-drawn jitter
changes; 0 freezes it).

## Examples

- `examples/forge/shapes`: every forge function in one picture.
- `examples/engine/walker`: a jointed robot walking through a village, with a camera
  that follows it and orbits, and a moving sun.
- `examples/engine/daycycle`: one scene across a whole day; `-hour 22 -look toon|ink|ascii`.
- `examples/ink/village`: a village with swaying trees and a moving sun.

Run any of them from the repo root: `go run ./examples/engine/walker -still 2`
(a picture at 2 s) or without `-still` for the video.

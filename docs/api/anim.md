# ln/anim

Time: animations as functions of normalised time, easing, keyframe tracks, splines, a pen-reveal effect, a camera helper, and ffmpeg export. Imports `ln/geom`, `ln/scene`. Source: `ln/anim/*.go`.

## Animation (`animation.go`)

```go
type Animation func(t float64) geom.Paths   // t in [0,1)
```

| Method | Meaning |
|---|---|
| `Frames(n int) []geom.Paths` | Calls the function with `t = i/n` for `i = 0..n-1`, so the last frame is not `t = 1` and a loop wraps cleanly |
| `WriteFrames(dir string, n int, width, height float64) error` | Creates `dir` and writes `frame_0000.svg`, `frame_0001.svg`, ... |
| `WriteSVG(path string, n int, fps, width, height float64) error` | One looping SVG that shows frame *i* with SMIL `<animate>`; no ffmpeg needed. Loop length is `n/fps` seconds |

## Video (`video.go`)

```go
type Frame interface{ Image(width, height float64) image.Image } // geom.Paths and *ink.Drawing satisfy it
type FrameAnimation func(t float64) Frame
```

`Animation.WriteVideo` and `FrameAnimation.WriteVideo` both have the signature `WriteVideo(path string, n int, fps, width, height float64) error`. They look up `ffmpeg` on `PATH` (error `ffmpeg not found on PATH` otherwise), pipe PNG frames to it, and pick the encoder from the extension:

| Extension | ffmpeg settings |
|---|---|
| `.gif` | palettegen/paletteuse, `-loop 0` |
| `.webm` | `libvpx-vp9`, `yuv420p` |
| anything else (use `.mp4`) | scales to even dimensions, `libx264`, `yuv420p` |

Frame *i* is requested at `t = i/n`. The caller decides what `t` means in seconds: `examples/space/blackhole` uses `u * frames / fps`.

## Easing and interpolation (`ease.go`)

`type Ease func(t float64) float64` maps progress 0..1 to eased progress (back and elastic overshoot).

Functions: `Linear`, `InQuad`, `OutQuad`, `InCubic`, `OutCubic`, `InOutSine`, `InOutQuad`, `InOutCubic`, `OutBack`, `OutElastic`, `OutBounce`. Helpers: `Clamp(t)` (0..1), `Lerp(a, b, t)`, `LerpVec(a, b geom.Vector, t)`.

`Segment(t, start, end float64, e Ease) float64` is progress of `t` through the window `[start, end]`, clamped to 0..1 and eased by `e` (`nil` means linear). If `end <= start` it is an instant step at `end`. Use it to schedule parts of one timeline: `grow := Segment(t, 0.1, 0.4, OutBack)`.

## Tracks and splines (`track.go`)

- `Key{T, V float64; Ease Ease}`: a value at time `T`; `Ease` shapes the move from the previous key **into** this one.
- `Track []Key`: keyed scalar. `At(t)` sorts a copy of the keys, holds the first value before the first key and the last value after the last, and otherwise interpolates with `Lerp(a.V, b.V, Segment(t, a.T, b.T, b.Ease))`. Keys may be given in any order.
- `VecTrack{X, Y, Z Track}` with `At(t) geom.Vector`.
- `Spline(pts []geom.Vector) func(u float64) geom.Vector`: Catmull-Rom through the points for `u` in 0..1 (clamped); end points repeat so the curve starts and ends at the first and last point. One point returns that point.

## Reveal (`reveal.go`)

`Reveal(paths geom.Paths, p float64) geom.Paths` returns the first fraction `p` (clamped) of the total line length, drawn in path order, cutting the last path part-way. This is the "pen is still drawing" effect (used by `examples/animation/spin`).

## Camera (`camera.go`)

```go
type Camera struct { Eye, Center, Up geom.Vector; Fovy float64 }
func Orbit(center geom.Vector, radius, height, angle, fovy float64) Camera
func (c Camera) Render(s *scene.Scene, width, height, step float64) geom.Paths
```

`Orbit` puts the eye at `center + (cos(angle)*radius, sin(angle)*radius, height)` with Z up; `angle` is in radians. `Render` calls `Scene.Render` with near 0.1 and far 1000.

## Example

```go
s := &scene.Scene{}
s.Add(shapes.NewCube(geom.V(-1, -1, -1), geom.V(1, 1, 1)))
a := anim.Animation(func(t float64) geom.Paths {
	return anim.Orbit(geom.V(0, 0, 0), 5, 2, t*2*math.Pi, 50).Render(s, 400, 400, 0.1)
})
err := a.WriteVideo("out/orbit.mp4", 60, 30, 400, 400)
```

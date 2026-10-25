# ln/buffer

Samples the visible world into per-pixel attribute buffers. Every look (ink, manga, ASCII) reads these buffers, so any shape works with every look. Imports `geom`, `light`, `scene`, `shapes`. Source: `ln/buffer/buffer.go`.

## Camera

```go
type Camera struct {
	Eye, Center, Up geom.Vector
	Fovy            float64 // vertical field of view, degrees
	Near, Far       float64 // 0 means 0.1 and 1000
}
func (c Camera) Matrix(aspect float64) geom.Matrix // LookAt(...).Perspective(Fovy, aspect, near, far)
```

## Buffers

```go
type Buffers struct {
	W, H     int
	Cam      Camera
	ViewProj geom.Matrix
	OK       []bool        // a surface was hit
	P        []geom.Vector // world position of the hit
	T        []float64     // distance from the eye
	N        []geom.Vector // unit normal facing the eye
	ID       []int         // object tag (0 = untagged); negative = seen in a mirror
	Refl     []bool        // the pixel is a mirror
	Dark     []float64     // 0 lit .. 1 black
	Dir      []geom.Vector // view ray (for sky and background shading)
}
```

All slices are row-major with **row 0 at the bottom** of the picture; index `k = j*W + i`.

`Gather(s *scene.Scene, cam Camera, w, h int, rig light.Rig) *Buffers` compiles the scene, then casts one ray per pixel (centre of the pixel) in parallel (`Rows`):

- On a miss the pixel stays `OK == false`; `Dir` is still set.
- If the hit shape implements `shapes.Reflector` with reflectivity above 0, the ray bounces about the (eye-facing) normal. `Refl` is set. If the bounced ray misses, `Dir` becomes the bounced direction so the mirror shows sky and `OK` stays false; otherwise the pixel records the surface seen in the mirror, with `T` the total path length and the object `ID` **negated**.
- `ID` comes from the shape's `Tag() int` method if it has one (`shapes.Triangle` does).
- The normal comes from the shape when it supplies one (flipped to face the ray). Otherwise a second pass estimates it from the neighbouring hit points, falling back to facing the camera at depth edges. The same pass fills `Dark` from `rig.Dark`.

## Methods

| Method | Meaning |
|---|---|
| `Smooth(k int) bool` | True when pixel `k` is interior and has no depth jump (more than 8% of its distance) to its four neighbours |
| `Project(p geom.Vector, pageW, pageH float64) geom.Vector` | World point to page pixels; `Z` is the distance from the eye |
| `Visible(p geom.Vector) bool` | Whether `p` is on the surface the camera sees (compared with the nearest neighbouring depth plus 0.8% tolerance). Off-buffer points count as hidden |
| `Rows(h int, f func(j int))` | Runs `f` for each row on up to `NumCPU` goroutines, waits for all; exported because other packages (`components/blackhole`) use it for their own per-pixel work |

## Example

```go
cam := buffer.Camera{Eye: geom.V(6, -8, 4), Center: geom.V(0, 0, 0), Up: geom.V(0, 0, 1), Fovy: 36}
b := buffer.Gather(sc, cam, 640, 360, light.Sun(geom.V(-0.5, -0.6, 1)))
darkAtCentre := b.Dark[180*640+320]
```

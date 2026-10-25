# ln/geom

Vectors, matrices, rays, boxes and 2D/3D polylines. Every other package builds on this one; `ln/geom` imports nothing else from the repo. Source: `ln/geom/*.go`.

Constants (`common.go`): `INF = 1e9`, `EPS = 1e-9`.

## Vector (`vector.go`)

`type Vector struct{ X, Y, Z float64 }`. All methods have value receivers and return new vectors.

| Identifier | Meaning |
|---|---|
| `V(x, y, z float64) Vector` | Shorthand for `Vector{X: x, Y: y, Z: z}` |
| `RandomUnitVector() Vector` | Random direction, drawn from the global `math/rand` |
| `RandomUnitVectorFrom(r *rand.Rand) Vector` | Same, drawn from `r`, so results repeat for a seed |
| `Length()`, `LengthSquared() float64` | Magnitude |
| `Distance(b)`, `DistanceSquared(b) float64` | Distance to `b` |
| `Dot(b) float64`, `Cross(b) Vector` | Dot and cross product |
| `Normalize() Vector` | Unit vector in the same direction |
| `Add(b)`, `Sub(b)`, `Mul(b)`, `Div(b) Vector` | Component-wise with another vector |
| `AddScalar`, `SubScalar`, `MulScalar`, `DivScalar(float64) Vector` | Component-wise with a number |
| `Min(b)`, `Max(b) Vector` | Component-wise minimum and maximum |
| `MinAxis() Vector` | Unit axis (`{1,0,0}`, `{0,1,0}` or `{0,0,1}`) along the component with the smallest absolute value; ties prefer X, then Y |
| `MinComponent() float64` | Smallest of X, Y, Z |
| `SegmentDistance(v, w Vector) float64` | Distance from the point to the segment v-w |

## Matrix (`matrix.go`)

`type Matrix struct` is a 4x4 `float64` matrix with unexported fields `x00`..`x33` (row-major naming). Build one with the constructors, chain with the method forms, apply with the `Mul*` methods.

| Identifier | Meaning |
|---|---|
| `Identity() Matrix` | The identity |
| `Translate(v)`, `Scale(v Vector) Matrix` | Translation, per-axis scale |
| `Rotate(v Vector, a float64) Matrix` | Rotation by `a` radians about axis `v` (normalised inside) |
| `Frustum(l, r, b, t, n, f)` | OpenGL-style perspective frustum |
| `Orthographic(l, r, b, t, n, f)` | Orthographic projection |
| `Perspective(fovy, aspect, near, far)` | `fovy` is in **degrees** (the code uses `fovy*math.Pi/360`) |
| `LookAt(eye, center, up Vector) Matrix` | View matrix; builds the camera basis and returns its inverse |
| `m.Translate/Scale/Rotate/Frustum/Orthographic/Perspective(...)` | Each is `New(...).Mul(m)`, so the new transform is applied after `m` (`matrix.go:88-110`) |
| `a.Mul(b) Matrix` | Matrix product |
| `MulPosition(v)` | Transforms a point (includes translation, no divide) |
| `MulPositionW(v)` | Transforms a point and divides by `w`; use for projection matrices |
| `MulDirection(v)` | Transforms a direction (no translation) and **normalises** the result |
| `MulRay(r)` | Transforms origin as a position and direction as a direction |
| `MulBox(b)` | Transforms a box and returns the axis-aligned box that contains it |
| `Transpose()`, `Determinant()`, `Inverse()` | As named |

## Ray and Box

- `type Ray struct{ Origin, Direction Vector }`; `Position(t) Vector` is `Origin + Direction*t`.
- `type Box struct{ Min, Max Vector }`.
  - `BoxForVectors([]Vector) Box` (empty input gives the zero Box)
  - `Anchor(a Vector) Vector` (point inside the box: `Min + Size*a`), `Center()`, `Size()`, `Contains(v)`, `Extend(b Box) Box` (union)
  - `(*Box).Intersect(r Ray) (t1, t2 float64)`: slab test. The ray hits when `t1 <= t2`; it divides by the direction components without guarding zero.
  - `(*Box).Partition(axis Axis, point float64) (left, right bool)`: which sides of a splitting plane the box touches.
- `type Axis uint8` with `AxisNone, AxisX, AxisY, AxisZ`; used by the BVH in `ln/accel`.

## Path and Paths (`path.go`)

`type Path []Vector` is a polyline; `type Paths []Path` is a set of them. Both have:

| Method | Meaning |
|---|---|
| `BoundingBox() Box` | `Path` version panics on an empty path (it reads `p[0]`) |
| `Transform(m Matrix)` | Applies `MulPosition` to every point |
| `Chop(step float64)` | Inserts points so no segment is longer than `step` |
| `Filter(f Filter)` | Runs each point through `f`, splitting a path where `f` says `false`; keeps pieces of two or more points. `Paths.Filter` runs paths in parallel and keeps input order, so `f` must be safe for concurrent use |
| `Simplify(threshold float64)` | Ramer-Douglas-Peucker |
| `Print()`, `String()` | `x,y;x,y` text (X and Y only) |

`type Filter interface{ Filter(v Vector) (Vector, bool) }`.

Output, on `Paths`: `Image(width, height float64) image.Image` (black 3 px lines on white, Y inverted), `WriteToPNG(path, w, h)`, `ToSVG(w, h) string`, `WriteToSVG(path, w, h) error`, `WriteToTXT(path) error`. `Path.ToSVG()` returns one `<polyline>`.

## Helpers (`util.go`)

`Radians(deg)`, `Degrees(rad)`, `Median(items []float64) float64` (**does not sort**: pass sorted input; empty gives 0), `ParseFloats([]string) []float64` (unparseable items become 0).

## Example

The same chain `ln/scene` and `ln/buffer` use to build a camera matrix (`scene.go:59`, `buffer.go:33`):

```go
m := geom.LookAt(eye, center, up).Perspective(fovy, width/height, near, far)
p := m.MulPositionW(geom.V(0, 0, 0)) // project a point
```

`a.Mul(b)` is the ordinary product `a * b` (`matrix.go:112`), so `b` acts on a point first.

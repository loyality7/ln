# ln/solid

The shape forge: build any 3D shape from code as a triangle mesh (`*model.Mesh`) that can be ray-cast, shaded and outlined. Imports `geom`, `model`, `shapes`. Source: `ln/solid/*.go`.

All builders return meshes with smooth vertex normals already applied (`Shade(mesh, 50)`), and closed solids are `Orient`ed so winding points outward. Nodes in `ln/stage` take these meshes through `Node.Add`, and have shortcuts for the primitives.

## Primitives (`build.go`)

| Function | Shape |
|---|---|
| `Box(sx, sy, sz)` | Centred on the origin |
| `Sphere(r)`, `Ellipsoid(rx, ry, rz)` | 48 by 24 grid; Z is the pole axis |
| `Torus(R, r)` | Ring radius `R`, tube radius `r`, in the XY plane |
| `Cylinder(r, h)` | Stands on z=0, flat ends (a 48-segment lathe) |
| `Cone(r, h)` | Base on z=0, tip at z=`h` |
| `Capsule(r, l)` | Cylinder of length `l` between the sphere centres, round ends; starts at z=0 (bottom of the lower hemisphere) |

## Building blocks

| Function | Meaning |
|---|---|
| `Surface(nu, nv int, f func(u, v float64) geom.Vector) *model.Mesh` | Samples `f` over the unit square on an `nu` by `nv` grid, giving an open sheet (a periodic `f` closes its own seam). Call `Orient` for closed solids |
| `Heightfield(sx, sy float64, nx, ny int, f func(x, y float64) float64)` | The surface `z = f(x, y)` over `[-sx/2, sx/2] x [-sy/2, sy/2]` |
| `Lathe(profile []Pt, segments int)` | Spins a profile (`X` = distance from the Z axis, `Y` = height) around Z. Points are used as given, so corners stay sharp. Oriented only if the first and last points have `X == 0` |
| `Smooth(pts []Pt, perSpan int) []Pt` | Catmull-Rom curve through the points, for curved lathe profiles |
| `Extrude(poly []Pt, h float64)` | Lifts any simple polygon (convex or concave, ear-clipped) from z=0 to z=`h`. Note it **reverses `poly` in place** if the polygon is clockwise |
| `Tube(path []geom.Vector, radius func(t float64) float64, sides int, closed bool)` | Sweeps a circle along a path using rotation-minimising frames; `radius(t)` varies along the path (`t` 0..1). `closed` joins the ends (twist is spread so the seam matches); otherwise the ends are capped |
| `Constant(r) func(float64) float64` | A constant radius function for `Tube` |
| `Merge(ms ...*model.Mesh) *model.Mesh` | One mesh from several (shares triangle pointers) |
| `Place(m *model.Mesh, matrix geom.Matrix) *model.Mesh` | Transforms in place and returns it |
| `Orient(m) *model.Mesh` | Flips winding (and shading normals) when the signed volume is negative |
| `type Pt struct{ X, Y float64 }` | 2D point for profiles and outlines |

## Shading and outlines

- `Shade(m *model.Mesh, creaseDegrees float64) *model.Mesh` (`shade.go`): at each welded vertex, averages (area-weighted) the normals of neighbouring faces within `creaseDegrees` of the triangle's own normal, so curved surfaces shade smoothly and real edges stay sharp. Sets `Smooth = true` and fills `N` on every triangle.
- `Outline(m *model.Mesh, creaseDegrees float64) *Outlined` (`outline.go`): welds vertices (1e-5 precision) to find which triangles meet at each edge. `Outlined` embeds `*model.Mesh` and has fields `Eye geom.Vector` and `Crease float64`.
  - `Creases() geom.Paths`: open borders and creases sharper than `Crease` degrees (no silhouette).
  - `PathsFrom(eye geom.Vector) geom.Paths`: creases plus the silhouette as seen from `eye`, so one `Outlined` can be shared by many placed copies (this is what `stage` does).
  - `Paths()`: `PathsFrom(o.Eye)`.

## Example

From the tests' usage pattern (`solid_test.go` checks volumes: a box of 2x3x4 has volume 24):

```go
vase := solid.Lathe(solid.Smooth([]solid.Pt{{0, 0}, {1, 0}, {1.4, 1.5}, {0.6, 3}, {0, 3}}, 8), 48)
knot := solid.Tube(path, solid.Constant(0.2), 16, true) // path is your []geom.Vector
```

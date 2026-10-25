# ln/shapes

Ray-castable shapes and the `Shape` interface everything else renders. Imports only `ln/geom`. Source: `ln/shapes/*.go`.

## The interface (`shape.go`, `hit.go`)

```go
type Shape interface {
	Compile()
	BoundingBox() geom.Box
	Contains(geom.Vector, float64) bool
	Intersect(geom.Ray) Hit
	Paths() geom.Paths
}
```

- `Compile()` is a hook called before rendering; every shape in this package implements it as a no-op.
- `Contains(v, f)` tests whether `v` is inside, with `f` as a tolerance. Not every shape implements a real test: `Cone`, `Triangle` and `Water` always return `false`.
- `Intersect(ray)` returns the first `Hit`.
- `Paths()` returns the shape's own line work (used by the line-art renderers).

```go
type Hit struct {
	Shape  Shape
	T      float64     // distance along the ray
	Normal geom.Vector // unit, outward for closed shapes; zero means "unknown"
}
var NoHit = Hit{T: geom.INF}
```

`Hit.Ok()` is `T < geom.INF`. `Hit.Min(b)` / `Max(b)` return the nearer / farther hit. A zero `Normal` tells renderers to estimate it from neighbouring pixels.

### Helpers and wrappers

| Identifier | Meaning |
|---|---|
| `EmptyShape` | Does nothing: zero box, `NoHit`, no paths |
| `TransformedShape{Shape, Matrix, Inverse}` / `NewTransformedShape(s, m) Shape` | Places a shape with a matrix. `Intersect` casts the ray in the shape's own space, then reports `T` as the distance to the world-space point and the normal through the inverse transpose, so non-uniform scale does not distort either (tested in `hit_test.go`) |
| `NoOutline{Shape}` | Wraps a shape and returns no `Paths`, so it appears only through shading |
| `BoxForShapes([]Shape)`, `BoxForTriangles([]*Triangle) geom.Box` | Union of bounding boxes (zero box when empty) |
| `Reflector` interface: `Reflectivity() float64` | Implemented by mirror shapes; renderers continue the ray after a bounce |

## Primitives

All constructors return pointers. Axes follow `ln/geom`: Z is up for the cone, cylinder and water.

| Shape | Constructor and fields | Notes (from the code) |
|---|---|---|
| `Sphere` | `NewSphere(center geom.Vector, radius float64)`; fields `Center, Radius, Box` | Real `Contains`. `Paths()` draws latitude lines every 10 degrees (from -80 to 80) and longitude lines every 10 degrees. `Paths2/3/4` are alternate stylised path generators (random lines or dots, using global `math/rand`). Hits closer than 1e-2 are ignored |
| `OutlineSphere` | `NewOutlineSphere(eye, up, center geom.Vector, radius)` | Embeds `Sphere`; `Paths()` is one circle: the silhouette as seen from `eye` |
| `Cube` | `NewCube(min, max geom.Vector)`; fields `Min, Max, Box` | Axis-aligned box. Hit normal is the outward normal of the nearest face. Ray starting inside returns the exit hit |
| `Cylinder` | `NewCylinder(radius, z0, z1 float64)` | Axis along Z between `z0` and `z1`, centred on X=Y=0. The ray test hits the curved side only (no end caps). `Paths()` is vertical lines every 10 degrees |
| `OutlineCylinder` | `NewOutlineCylinder(eye, up, radius, z0, z1)`; `NewTransformedOutlineCylinder(eye, up, v0, v1 geom.Vector, radius) Shape` | Silhouette lines seen from `eye`; the transformed form places the cylinder from `v0` to `v1` |
| `Cone` | `NewCone(radius, height float64)` | Base at Z=0, tip at Z=`height`. `Paths()` is lines from the base circle (every 30 degrees) to the tip |
| `OutlineCone` | `NewOutlineCone(eye, up, radius, height)`; `NewTransformedOutlineCone(eye, up, v0, v1 geom.Vector, radius) Shape` | Base circle plus two silhouette lines to the tip |
| `Triangle` | `NewTriangle(v1, v2, v3)`; fields `V1,V2,V3, Box, ID int, N [3]geom.Vector, Smooth bool` | Moller-Trumbore test; hits behind the origin (`d < EPS`) are ignored. `FaceNormal()` is from winding. When `Smooth` is set, hits interpolate the corner normals `N`. `ID` is an object tag (`Tag()` returns it, 0 = untagged). Call `UpdateBoundingBox()` after moving corners |
| `Function` | `NewFunction(f func(x, y float64) float64, box geom.Box, dir Direction) Shape`; `Direction` is `Above` or `Below` | A surface `z = f(x, y)`. `Contains` is `v.Z > f` (Above) or `v.Z < f` (Below). `Intersect` marches the ray in steps of 1/64 up to t=10 and returns the first sign change inside `Box`; the normal comes from finite differences. `Paths()` draws spokes (every 5 degrees, radius 0 to 8) clamped to the box's Z range; `Paths1()` draws a grid |
| `Water` | `NewWater(level, half float64) *Water`; fields `Level, Half, Height func(x,y) float64, Amp, Gloss` | Horizontal mirror covering X and Y in `[-Half, Half]`. `Height` only supplies slope (scaled by `Amp`) so reflections ripple without moving the surface. `Reflectivity()` returns `Gloss` (default 1). Implements `Reflector`. No paths |

`LatLngToXYZ(lat, lng, radius float64) geom.Vector` converts degrees to a point on a sphere (Z is the pole).

## Boolean shapes (`csg.go`)

`type Op int` with `Intersection` and `Difference` (Union is commented out in the source). `NewBooleanShape(op, shapes...)` folds left (A op B op C...), `NewIntersection(...)`, `NewDifference(...)`. `BooleanShape.Contains` uses a fixed tolerance of 1e-3 and ignores the `f` argument. `Intersect` casts both shapes, takes the nearer hit and, if that point is not inside the boolean result, re-casts from 0.01 further along. `BoundingBox` is the union of both operands (the source marks it `TODO: fix this`). `Paths()` chops the operands' paths to 0.01 and keeps only the parts that pass `Contains`.

## Example

```go
s := shapes.NewTransformedShape(shapes.NewSphere(geom.V(0, 0, 0), 1), geom.Scale(geom.V(1, 4, 1)))
h := s.Intersect(geom.Ray{Origin: geom.V(0, -10, 0), Direction: geom.V(0, 1, 0)})
// h.T == 6, h.Normal == (0, -1, 0)   (from hit_test.go)
```

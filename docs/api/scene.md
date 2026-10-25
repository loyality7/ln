# ln/scene

A list of shapes plus the classic line-art renderer: project each shape's `Paths()` through a camera and keep only the visible parts. Imports `ln/accel`, `ln/geom`, `ln/shapes`. Source: `ln/scene/scene.go`, `filter.go`.

## Scene

```go
type Scene struct {
	Shapes []shapes.Shape
	Tree   *accel.Tree
}
```

| Method | Meaning |
|---|---|
| `Add(shape shapes.Shape)` | Appends a shape |
| `Compile()` | Calls `Compile()` on every shape, then builds `Tree` over `Shapes` if `Tree` is nil. If you add shapes after compiling, set `Tree = nil` first |
| `Intersect(r geom.Ray) shapes.Hit` | Casts through `Tree`; needs `Compile()` to have run |
| `Visible(eye, point geom.Vector) bool` | Casts from `point` toward `eye`; true when nothing is hit nearer than `eye` |
| `Paths() geom.Paths` | Every shape's own `Paths()` concatenated, in world space, unprojected |
| `Render(eye, center, up geom.Vector, width, height, fovy, near, far, step float64) geom.Paths` | Builds `LookAt(eye, center, up).Perspective(fovy, width/height, near, far)` and renders; `fovy` is in degrees |
| `RenderWithMatrix(matrix geom.Matrix, eye geom.Vector, width, height, step float64) geom.Paths` | Same with a camera matrix you supply |
| `RenderDepth(...)` | Same arguments as `Render`; each output point's Z holds its distance from the eye (used for ink line weight) |

What the private `render` does, in order: `Compile()`; `Paths()`; if `step > 0`, `Chop(step)` so long lines are tested at many points; filter with a `ClipFilter`; if `step > 0`, `Simplify(1e-6)`; then map the clip cube to the picture: X and Y to `[0, width]` x `[0, height]` (Y is up, as in `geom.Paths.Image`), Z kept only for `RenderDepth`.

## ClipFilter (`filter.go`)

```go
type ClipFilter struct {
	Matrix geom.Matrix
	Eye    geom.Vector
	Scene  *Scene
	Depth  bool
}
var ClipBox = geom.Box{Min: {-1,-1,-1}, Max: {1,1,1}}
```

It implements `geom.Filter`. For each point it projects with `MulPositionW`, drops the point if `Scene.Visible(Eye, v)` is false or if the projected point is outside `ClipBox`, and when `Depth` is set replaces Z with `v.Distance(Eye)`.

## Example

From `scene_test.go`:

```go
s := &scene.Scene{}
s.Add(shapes.NewCube(geom.V(-1, -1, -1), geom.V(1, 1, 1)))
paths := s.Render(geom.V(4, 3, 2), geom.V(0, 0, 0), geom.V(0, 0, 1), 100, 100, 50, 0.1, 100, 0.1)
paths.WriteToSVG("cube.svg", 100, 100)
```

Tests also check that rendering is deterministic (the parallel `Filter` keeps input order).

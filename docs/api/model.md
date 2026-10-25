# ln/model

Triangle meshes: loading, fitting, transforming and slicing. Imports `ln/accel`, `ln/geom`, `ln/shapes`. Source: `ln/model/*.go`.

## Mesh (`mesh.go`)

```go
type Mesh struct {
	Box       geom.Box
	Triangles []*shapes.Triangle
	Tree      *accel.Tree
}
func NewMesh(triangles []*shapes.Triangle) *Mesh
```

`*Mesh` implements `shapes.Shape`:

- `Compile()` builds `Tree` over the triangles if it is nil. **Call it before `Intersect`**; `Intersect` dereferences `Tree` without checking.
- `BoundingBox()` returns `Box`. `Contains` always returns `false`.
- `Paths()` is every triangle's three edges.

Other methods:

| Method | Meaning |
|---|---|
| `UpdateBoundingBox()` | Recomputes `Box` from the triangles |
| `Transform(m geom.Matrix)` | Applies `m` to every vertex, updates boxes and sets `Tree = nil` (recompile before ray casting again) |
| `MoveTo(position, anchor geom.Vector)` | Translates so the box point at `anchor` (0..1 per axis, see `Box.Anchor`) lands on `position` |
| `FitInside(box geom.Box, anchor geom.Vector)` | Uniformly scales to the largest size that fits inside `box`, then places the leftover space using `anchor` |
| `UnitCube()` | Fits inside the unit box, then centres the mesh on the origin (`MoveTo` with anchor 0.5) |
| `Voxelize(size float64) []*shapes.Cube` | Slices at every `size` in Z, snaps the cut points to a grid and returns one cube per occupied cell |
| `SaveBinarySTL(path) error` | Same as `SaveBinarySTL(path, m)` |

Note: a mesh's triangles keep whatever `ID`, `Smooth` and normals they had; `Transform` moves vertices only and does not touch `N`.

## Loaders (`obj.go`, `stl.go`)

| Function | Behaviour |
|---|---|
| `LoadOBJ(path) (*Mesh, error)` | Reads `v` lines (needs three coordinates) and `f` lines. Faces with more than three vertices are fan-triangulated. Index text after the first `/` is ignored; negative indices count from the end. Normals and textures are not read |
| `LoadBinarySTL(path) (*Mesh, error)` | 80-byte header, `uint32` count, then 50-byte records (a skipped normal, three vertices as `float32`, a skipped `uint16`). Prints `Loading STL (Binary): <path>` |
| `LoadSTL(path) (*Mesh, error)` | ASCII STL: every line of the form `vertex x y z` is collected and grouped three at a time. Prints `Loading STL (ASCII): <path>` |
| `SaveBinarySTL(path, mesh) error` | Writes the binary format; the normal fields are written as zeros |

The exported structs `STLHeader` and `STLTriangle` describe the binary layout (blank fields skip data).

## Plane slicing (`plane.go`)

`IntersectPlane(p *shapes.Plane, m *Mesh) geom.Paths` returns, for each triangle the plane cuts, a two-point path of the cut segment. See `shapes.Plane` (`Point`, `Normal`, `IntersectSegment`, `IntersectTriangle`).

## Example

From `examples/meshes/suzanne/main.go`:

```go
mesh, err := model.LoadOBJ("assets/models/suzanne.obj")
mesh.UnitCube()
// add mesh to a scene; the scene compiles it
```

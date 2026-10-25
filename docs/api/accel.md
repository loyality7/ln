# ln/accel

A bounding-volume tree that speeds up ray casts against many shapes. Imports `ln/geom` and `ln/shapes`. Source: `ln/accel/tree.go`.

```go
type Tree struct { Box geom.Box; Root *Node }
func NewTree(list []shapes.Shape) *Tree
func (tree *Tree) Intersect(r geom.Ray) shapes.Hit
```

`NewTree` computes the box with `shapes.BoxForShapes`, builds a root `Node` holding the whole list and calls `Split(0)`. `Tree.Intersect` first clips the ray against `Box` (returning `shapes.NoHit` if it misses or `tmax <= 0`), then walks the tree.

## Node

```go
type Node struct {
	Axis   geom.Axis
	Point  float64
	Shapes []shapes.Shape // only filled on leaves
	Left, Right *Node
}
```

| Method | What the code does |
|---|---|
| `NewNode(list) *Node` | A leaf (`Axis == geom.AxisNone`) holding `list` |
| `Intersect(r, tmin, tmax) shapes.Hit` | A leaf tests every shape (`IntersectShapes`); an inner node computes where the ray crosses the split plane and visits the near child first, then the far child only if needed |
| `IntersectShapes(r) shapes.Hit` | Brute-force nearest hit over the node's shapes |
| `PartitionScore(axis, point) int` | The larger of the left count and the right count if split there (a shape that straddles the plane counts on both sides) |
| `Partition(size, axis, point) (left, right []shapes.Shape)` | Splits the shapes; straddlers go to both |
| `Split(depth int)` | Leaves with fewer than 8 shapes are left alone. Otherwise it takes the median of all box minima and maxima on each axis, scores each candidate and keeps the best one that puts fewer than 85% of the shapes on its larger side. If none qualifies it stays a leaf. Splits recurse; `Shapes` is cleared on inner nodes |

## Use

You rarely call it directly: `model.Mesh.Compile()` builds a `Tree` over its triangles, and `scene.Scene.Compile()` builds one over its shapes.

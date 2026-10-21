package accel

import (
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/shapes"
	"math"
	"sort"
)

type Tree struct {
	Box  geom.Box
	Root *Node
}

func NewTree(list []shapes.Shape) *Tree {
	box := shapes.BoxForShapes(list)
	node := NewNode(list)
	node.Split(0)
	return &Tree{box, node}
}

func (tree *Tree) Intersect(r geom.Ray) shapes.Hit {
	tmin, tmax := tree.Box.Intersect(r)
	if tmax < tmin || tmax <= 0 {
		return shapes.NoHit
	}
	return tree.Root.Intersect(r, tmin, tmax)
}

type Node struct {
	Axis   geom.Axis
	Point  float64
	Shapes []shapes.Shape
	Left   *Node
	Right  *Node
}

func NewNode(list []shapes.Shape) *Node {
	return &Node{geom.AxisNone, 0, list, nil, nil}
}

func (node *Node) Intersect(r geom.Ray, tmin, tmax float64) shapes.Hit {
	var tsplit float64
	var leftFirst bool
	switch node.Axis {
	case geom.AxisNone:
		return node.IntersectShapes(r)
	case geom.AxisX:
		tsplit = (node.Point - r.Origin.X) / r.Direction.X
		leftFirst = (r.Origin.X < node.Point) || (r.Origin.X == node.Point && r.Direction.X <= 0)
	case geom.AxisY:
		tsplit = (node.Point - r.Origin.Y) / r.Direction.Y
		leftFirst = (r.Origin.Y < node.Point) || (r.Origin.Y == node.Point && r.Direction.Y <= 0)
	case geom.AxisZ:
		tsplit = (node.Point - r.Origin.Z) / r.Direction.Z
		leftFirst = (r.Origin.Z < node.Point) || (r.Origin.Z == node.Point && r.Direction.Z <= 0)
	}
	var first, second *Node
	if leftFirst {
		first = node.Left
		second = node.Right
	} else {
		first = node.Right
		second = node.Left
	}
	if tsplit > tmax || tsplit <= 0 {
		return first.Intersect(r, tmin, tmax)
	} else if tsplit < tmin {
		return second.Intersect(r, tmin, tmax)
	} else {
		h1 := first.Intersect(r, tmin, tsplit)
		if h1.T <= tsplit {
			return h1
		}
		h2 := second.Intersect(r, tsplit, math.Min(tmax, h1.T))
		if h1.T <= h2.T {
			return h1
		} else {
			return h2
		}
	}
}

func (node *Node) IntersectShapes(r geom.Ray) shapes.Hit {
	hit := shapes.NoHit
	for _, shape := range node.Shapes {
		h := shape.Intersect(r)
		if h.T < hit.T {
			hit = h
		}
	}
	return hit
}

func (node *Node) PartitionScore(axis geom.Axis, point float64) int {
	left, right := 0, 0
	for _, shape := range node.Shapes {
		box := shape.BoundingBox()
		l, r := box.Partition(axis, point)
		if l {
			left++
		}
		if r {
			right++
		}
	}
	if left >= right {
		return left
	} else {
		return right
	}
}

func (node *Node) Partition(size int, axis geom.Axis, point float64) (left, right []shapes.Shape) {
	left = make([]shapes.Shape, 0, size)
	right = make([]shapes.Shape, 0, size)
	for _, shape := range node.Shapes {
		box := shape.BoundingBox()
		l, r := box.Partition(axis, point)
		if l {
			left = append(left, shape)
		}
		if r {
			right = append(right, shape)
		}
	}
	return
}

func (node *Node) Split(depth int) {
	if len(node.Shapes) < 8 {
		return
	}
	xs := make([]float64, 0, len(node.Shapes)*2)
	ys := make([]float64, 0, len(node.Shapes)*2)
	zs := make([]float64, 0, len(node.Shapes)*2)
	for _, shape := range node.Shapes {
		box := shape.BoundingBox()
		xs = append(xs, box.Min.X)
		xs = append(xs, box.Max.X)
		ys = append(ys, box.Min.Y)
		ys = append(ys, box.Max.Y)
		zs = append(zs, box.Min.Z)
		zs = append(zs, box.Max.Z)
	}
	sort.Float64s(xs)
	sort.Float64s(ys)
	sort.Float64s(zs)
	mx, my, mz := geom.Median(xs), geom.Median(ys), geom.Median(zs)
	best := int(float64(len(node.Shapes)) * 0.85)
	bestAxis := geom.AxisNone
	bestPoint := 0.0
	sx := node.PartitionScore(geom.AxisX, mx)
	if sx < best {
		best = sx
		bestAxis = geom.AxisX
		bestPoint = mx
	}
	sy := node.PartitionScore(geom.AxisY, my)
	if sy < best {
		best = sy
		bestAxis = geom.AxisY
		bestPoint = my
	}
	sz := node.PartitionScore(geom.AxisZ, mz)
	if sz < best {
		best = sz
		bestAxis = geom.AxisZ
		bestPoint = mz
	}
	if bestAxis == geom.AxisNone {
		return
	}
	l, r := node.Partition(best, bestAxis, bestPoint)
	node.Axis = bestAxis
	node.Point = bestPoint
	node.Left = NewNode(l)
	node.Right = NewNode(r)
	node.Left.Split(depth + 1)
	node.Right.Split(depth + 1)
	node.Shapes = nil // only needed at leaf nodes
}

package geom

import (
	"math"
)

type Box struct {
	Min, Max Vector
}

func BoxForVectors(vectors []Vector) Box {
	if len(vectors) == 0 {
		return Box{}
	}
	min := vectors[0]
	max := vectors[0]
	for _, v := range vectors {
		min = min.Min(v)
		max = max.Max(v)
	}
	return Box{min, max}
}

func (a Box) Anchor(anchor Vector) Vector {
	return a.Min.Add(a.Size().Mul(anchor))
}

func (a Box) Center() Vector {
	return a.Anchor(Vector{0.5, 0.5, 0.5})
}

func (a Box) Size() Vector {
	return a.Max.Sub(a.Min)
}

func (a Box) Contains(b Vector) bool {
	return a.Min.X <= b.X && a.Max.X >= b.X &&
		a.Min.Y <= b.Y && a.Max.Y >= b.Y &&
		a.Min.Z <= b.Z && a.Max.Z >= b.Z
}

func (a Box) Extend(b Box) Box {
	return Box{a.Min.Min(b.Min), a.Max.Max(b.Max)}
}

func (b *Box) Intersect(r Ray) (float64, float64) {
	x1 := (b.Min.X - r.Origin.X) / r.Direction.X
	y1 := (b.Min.Y - r.Origin.Y) / r.Direction.Y
	z1 := (b.Min.Z - r.Origin.Z) / r.Direction.Z
	x2 := (b.Max.X - r.Origin.X) / r.Direction.X
	y2 := (b.Max.Y - r.Origin.Y) / r.Direction.Y
	z2 := (b.Max.Z - r.Origin.Z) / r.Direction.Z
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	if z1 > z2 {
		z1, z2 = z2, z1
	}
	t1 := math.Max(math.Max(x1, y1), z1)
	t2 := math.Min(math.Min(x2, y2), z2)
	return t1, t2
}

func (b *Box) Partition(axis Axis, point float64) (left, right bool) {
	switch axis {
	case AxisX:
		left = b.Min.X <= point
		right = b.Max.X >= point
	case AxisY:
		left = b.Min.Y <= point
		right = b.Max.Y >= point
	case AxisZ:
		left = b.Min.Z <= point
		right = b.Max.Z >= point
	}
	return
}

// Rect returns a closed 2D Path of the box in the XY plane.
func (b Box) Rect() Path {
	return Path{
		{X: b.Min.X, Y: b.Min.Y},
		{X: b.Max.X, Y: b.Min.Y},
		{X: b.Max.X, Y: b.Max.Y},
		{X: b.Min.X, Y: b.Max.Y},
		{X: b.Min.X, Y: b.Min.Y},
	}
}

// ClipSegment2D clips a line segment (p0 to p1) against the box in the XY plane
// using the Liang-Barsky algorithm. It returns the clipped start and end points
// and true if any part of the segment lies within the box.
func (b Box) ClipSegment2D(p0, p1 Vector) (Vector, Vector, bool) {
	t0, t1 := 0.0, 1.0
	dx := p1.X - p0.X
	dy := p1.Y - p0.Y

	p := [4]float64{-dx, dx, -dy, dy}
	q := [4]float64{p0.X - b.Min.X, b.Max.X - p0.X, p0.Y - b.Min.Y, b.Max.Y - p0.Y}

	for i := 0; i < 4; i++ {
		if p[i] == 0 {
			if q[i] < 0 {
				return Vector{}, Vector{}, false
			}
		} else {
			t := q[i] / p[i]
			if p[i] < 0 {
				if t > t1 {
					return Vector{}, Vector{}, false
				}
				if t > t0 {
					t0 = t
				}
			} else {
				if t < t0 {
					return Vector{}, Vector{}, false
				}
				if t < t1 {
					t1 = t
				}
			}
		}
	}
	return Vector{X: p0.X + t0*dx, Y: p0.Y + t0*dy}, Vector{X: p0.X + t1*dx, Y: p0.Y + t1*dy}, true
}


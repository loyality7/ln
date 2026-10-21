package shapes

import "github.com/loyality7/ln/ln/geom"

func BoxForShapes(shapes []Shape) geom.Box {
	if len(shapes) == 0 {
		return geom.Box{}
	}
	box := shapes[0].BoundingBox()
	for _, shape := range shapes {
		box = box.Extend(shape.BoundingBox())
	}
	return box
}

func BoxForTriangles(shapes []*Triangle) geom.Box {
	if len(shapes) == 0 {
		return geom.Box{}
	}
	box := shapes[0].BoundingBox()
	for _, shape := range shapes {
		box = box.Extend(shape.BoundingBox())
	}
	return box
}

package main

import (
	"fmt"
	"math"

	"github.com/jonas-p/go-shp"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/scene"
	"github.com/loyality7/ln/ln/shapes"
)

func GetPaths(shape shp.Shape) geom.Paths {
	switch v := shape.(type) {
	case *shp.PolyLine:
		return getPaths(v)
	case *shp.Polygon:
		line := shp.PolyLine(*v)
		return getPaths(&line)
	}
	return nil
}

func getPaths(line *shp.PolyLine) geom.Paths {
	var result geom.Paths
	parts := append(line.Parts, line.NumPoints)
	for part := 0; part < len(parts)-1; part++ {
		var path geom.Path
		a := parts[part]
		b := parts[part+1]
		for i := a; i < b; i++ {
			pt := line.Points[i]
			path = append(path, shapes.LatLngToXYZ(pt.Y, pt.X, 1))
		}
		result = append(result, path)
	}
	return result
}

func LoadPaths() geom.Paths {
	var result geom.Paths
	file, err := shp.Open("examples/ne_10m_coastline/ne_10m_coastline.shp")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	for file.Next() {
		_, shape := file.Shape()
		paths := GetPaths(shape)
		result = append(result, paths...)
	}
	return result
}

type Earth struct {
	shapes.Shape
	Lines geom.Paths
}

func (e *Earth) Paths() geom.Paths {
	// return append(e.Lines, e.Shape.Paths()...)
	return e.Lines
}

func Render(lines geom.Paths, matrix geom.Matrix) geom.Paths {
	scene := scene.Scene{}
	sphere := shapes.NewSphere(geom.Vector{}, 1)
	earth := Earth{sphere, lines}
	shape := shapes.NewTransformedShape(&earth, matrix)
	scene.Add(shape)
	eye := shapes.LatLngToXYZ(35.7806, -78.6389, 1).Normalize().MulScalar(2.46)
	center := geom.Vector{}
	up := geom.Vector{0, 0, 1}
	return scene.Render(eye, center, up, 256, 256, 60, 0.1, 100, 0.01)
}

func Circle(r float64) geom.Path {
	var result geom.Path
	for i := 0; i <= 360; i++ {
		a := geom.Radians(float64(i))
		x := math.Cos(a) * r
		y := math.Sin(a) * r
		result = append(result, geom.Vector{x, y, 0})
	}
	return result
}

func main() {
	lines := LoadPaths()
	m := geom.Identity()
	paths := Render(lines, m)
	paths = append(paths, Circle(0.95))
	paths = append(paths, Circle(0.953))
	paths = append(paths, Circle(0.956))
	paths.WriteToPNG("earth.png", 256, 256)
	// paths.Print()
	for i := 0; i < 360; i += 2 {
		fmt.Println(i)
		m := geom.Rotate(geom.Vector{0, 0, 1}, geom.Radians(float64(i)))
		paths := Render(lines, m)
		paths = append(paths, Circle(0.95))
		paths = append(paths, Circle(0.953))
		paths = append(paths, Circle(0.956))
		filename := fmt.Sprintf("out%03d.png", i)
		paths.WriteToPNG(filename, 256, 256)
	}
}

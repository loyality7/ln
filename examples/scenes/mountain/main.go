package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/scene"
	"github.com/loyality7/ln/ln/shapes"
)

func load(path string) []geom.Vector {
	var result []geom.Vector
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	for _, record := range records {
		x, _ := strconv.ParseFloat(record[0], 64)
		y, _ := strconv.ParseFloat(record[1], 64)
		z, _ := strconv.ParseFloat(record[2], 64)
		result = append(result, geom.Vector{x, z, y})
	}
	return result
}

func main() {
	blocks := load("examples/mountain.csv")
	fmt.Println(len(blocks))
	scene := scene.Scene{}
	size := geom.Vector{0.5, 0.5, 0.5}
	for _, v := range blocks {
		scene.Add(shapes.NewCube(v.Sub(size), v.Add(size)))
	}
	eye := geom.Vector{90, -90, 70}
	center := geom.Vector{0, 0, -15}
	up := geom.Vector{0, 0, 1}
	width := 1920.0
	height := 1080.0
	paths := scene.Render(eye, center, up, width, height, 50, 0.1, 1000, 0.1)
	paths.WriteToPNG("out.png", width, height)
	// paths.Print()
}

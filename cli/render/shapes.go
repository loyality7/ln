package render

import (
	"fmt"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/model"
	"github.com/loyality7/ln/ln/shapes"
	"github.com/loyality7/ln/ln/solid"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
)

// csgPiece makes one piece of a csg shape.
func csgPiece(n string, p part) (shapes.Shape, error) {
	a := p.Args
	need := func(c int) error {
		if len(a) != c {
			return fmt.Errorf("csg part %q needs %d number(s) in \"args\", got %d", p.Shape, c, len(a))
		}
		return nil
	}
	var sh shapes.Shape
	switch p.Shape {
	case "cube":
		if err := need(3); err != nil {
			return nil, err
		}
		sh = shapes.NewCube(geom.V(-a[0]/2, -a[1]/2, -a[2]/2), geom.V(a[0]/2, a[1]/2, a[2]/2))
	case "sphere":
		if err := need(1); err != nil {
			return nil, err
		}
		sh = shapes.NewSphere(geom.Vector{}, a[0])
	case "cylinder":
		if err := need(3); err != nil {
			return nil, err
		}
		sh = shapes.NewCylinder(a[0], a[1], a[2])
	case "cone":
		if err := need(2); err != nil {
			return nil, err
		}
		sh = shapes.NewCone(a[0], a[1])
	default:
		return nil, fmt.Errorf("node %q: unknown csg part %q (cube, sphere, cylinder, cone)", n, p.Shape)
	}
	if p.At != nil {
		sh = shapes.NewTransformedShape(sh, geom.Translate(vec(p.At)))
	}
	return sh, nil
}

// loadHeightfield turns a grayscale picture into a landscape: white is high, black is low.
func loadHeightfield(file string, sx, sy, height float64) (*model.Mesh, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("%s is not a picture ln can read (png, jpeg or gif): %v", file, err)
	}
	b := img.Bounds()
	at := func(u, v float64) float64 { // bilinear, 0..1 across the picture
		x, y := u*float64(b.Dx()-1), (1-v)*float64(b.Dy()-1)
		x0, y0 := int(x), int(y)
		x1, y1 := min(x0+1, b.Dx()-1), min(y0+1, b.Dy()-1)
		fx, fy := x-float64(x0), y-float64(y0)
		g := func(i, j int) float64 {
			r, gg, bb, _ := img.At(b.Min.X+i, b.Min.Y+j).RGBA()
			return (0.2126*float64(r) + 0.7152*float64(gg) + 0.0722*float64(bb)) / 65535
		}
		return (g(x0, y0)*(1-fx)+g(x1, y0)*fx)*(1-fy) + (g(x0, y1)*(1-fx)+g(x1, y1)*fx)*fy
	}
	nx, ny := min(b.Dx(), 160), min(b.Dy(), 160)
	return solid.Heightfield(sx, sy, nx, ny, func(x, y float64) float64 { return height * at(x/sx+0.5, y/sy+0.5) }), nil
}

// loadMesh reads an .obj or .stl file and fits it into a unit cube.
func loadMesh(file string) (*model.Mesh, error) {
	var m *model.Mesh
	var err error
	switch strings.ToLower(filepath.Ext(file)) {
	case ".obj":
		m, err = model.LoadOBJ(file)
	case ".stl":
		head := make([]byte, 5)
		if f, e := os.Open(file); e == nil {
			f.Read(head)
			f.Close()
		}
		if string(head) == "solid" {
			m, err = model.LoadSTL(file)
		} else {
			m, err = model.LoadBinarySTL(file)
		}
	default:
		return nil, fmt.Errorf("mesh %q: use an .obj or .stl file", file)
	}
	if err != nil {
		return nil, err
	}
	if len(m.Triangles) == 0 {
		return nil, fmt.Errorf("mesh %q has no triangles", file)
	}
	m.UnitCube()
	return solid.Shade(m, 50), nil
}

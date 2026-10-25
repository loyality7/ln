// Package mesh is `ln mesh`: convert, fit and slice 3D model files.
package mesh

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/loyality7/ln/cli/ui"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/model"
	"github.com/loyality7/ln/ln/shapes"
)

// Run is `ln mesh`.
func Run(args []string) int {
	fs := ui.NewFlagSet("mesh")
	out := fs.String("o", "", "write the model here as binary .stl (after any fit)")
	unit := fs.Bool("unit", false, "scale and move the model to fit a 1 x 1 x 1 cube")
	slice := fs.Float64("slice", 0, "cut the model flat at this height (Z) and write the outline to -svg")
	svg := fs.String("svg", "", "where -slice writes its outline (default: <model>_slice.svg)")
	info := fs.Bool("info", false, "print the size and triangle count")
	ui.Usage(fs, "ln mesh <model.obj|model.stl> [options]",
		"Look at, convert, resize or slice a 3D model. Reads .obj and .stl.", nil,
		"ln mesh bunny.obj -info", "ln mesh bunny.obj -unit -o bunny.stl", "ln mesh bunny.stl -slice 0.5 -svg cut.svg")
	if proceed, code := ui.ParseFlags(fs, args); !proceed {
		return code
	}
	if fs.NArg() != 1 {
		return ui.Fail("give one model file, for example: ln mesh bunny.obj -info")
	}
	in := fs.Arg(0)
	var m *model.Mesh
	var err error
	switch strings.ToLower(filepath.Ext(in)) {
	case ".obj":
		m, err = model.LoadOBJ(in)
	case ".stl":
		if isBinarySTL(in) {
			m, err = model.LoadBinarySTL(in)
		} else {
			m, err = model.LoadSTL(in)
		}
	default:
		return ui.Fail("%s: only .obj and .stl files can be read", in)
	}
	if err != nil {
		return ui.Fail("%v", err)
	}
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	if *unit {
		m.UnitCube()
	}
	if *info || (!*unit && *out == "" && !set["slice"]) {
		b := m.BoundingBox()
		s := b.Size()
		fmt.Printf("%d triangles\nsize %.4g x %.4g x %.4g\nfrom (%.4g, %.4g, %.4g) to (%.4g, %.4g, %.4g)\n",
			len(m.Triangles), s.X, s.Y, s.Z, b.Min.X, b.Min.Y, b.Min.Z, b.Max.X, b.Max.Y, b.Max.Z)
	}
	if set["slice"] {
		b := m.BoundingBox()
		plane := &shapes.Plane{Point: geom.Vector{Z: *slice}, Normal: geom.Vector{Z: 1}}
		paths := model.IntersectPlane(plane, m)
		if len(paths) == 0 {
			return ui.Fail("the cut at Z=%g misses the model (it spans Z %.4g to %.4g)", *slice, b.Min.Z, b.Max.Z)
		}
		name := *svg
		if name == "" {
			name = strings.TrimSuffix(in, filepath.Ext(in)) + "_slice.svg"
		}
		sb := paths.BoundingBox()
		w, h := sb.Max.X-sb.Min.X, sb.Max.Y-sb.Min.Y
		shift := geom.Translate(geom.Vector{X: -sb.Min.X, Y: -sb.Min.Y})
		if err := os.WriteFile(name, []byte(paths.Transform(shift).ToSVG(w, h)), 0o644); err != nil {
			return ui.Fail("%v", err)
		}
		ui.OK("wrote %s", name)
	}
	if *out != "" {
		if err := m.SaveBinarySTL(*out); err != nil {
			return ui.Fail("%v", err)
		}
		ui.OK("wrote %s", *out)
	}
	return 0
}

// isBinarySTL tells a binary .stl from a text one by its size: a binary file is 84 bytes plus 50 per triangle.
// (A binary file whose header starts with "solid" would otherwise be taken for text.)
func isBinarySTL(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil || len(data) < 84 {
		return false
	}
	n := int(data[80]) | int(data[81])<<8 | int(data[82])<<16 | int(data[83])<<24
	return len(data) == 84+50*n
}

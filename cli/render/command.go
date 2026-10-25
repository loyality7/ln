package render

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/loyality7/ln/ln/ascii"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/ink"
	"github.com/loyality7/ln/ln/stage"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Run is `ln render`: a scene file drawn as a video or a picture.
func Run(args []string) int {
	var file string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		file, args = args[0], args[1:]
	}
	fs := flag.NewFlagSet("render", flag.ExitOnError)
	out := fs.String("o", "", "output: .mp4 .gif .webm for a video; .png, .svg or .txt (ascii look) for one picture (default: <scene>.mp4)")
	frame := fs.Float64("frame", -1, "draw one picture at this time in seconds, instead of a video")
	look := fs.String("look", "", "ink | manga | sketch | technical | ascii | colour (overrides the scene)")
	w := fs.Float64("w", 0, "width in pixels (overrides the scene)")
	h := fs.Float64("h", 0, "height in pixels (overrides the scene)")
	fps := fs.Float64("fps", 0, "frames per second (overrides the scene)")
	frames := fs.String("frames", "", "write every frame as a numbered .png into this folder, instead of a video")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: ln render <scene.json | program.go> [options]")
		fs.PrintDefaults()
	}
	fs.Parse(args)
	if file == "" {
		file = fs.Arg(0)
	}
	if file == "" {
		fs.Usage()
		return 2
	}
	if strings.EqualFold(filepath.Ext(file), ".go") {
		return runGo(file, os.Args[3:])
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ln: cannot read", file)
		return 1
	}
	var sc scene
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&sc); err != nil {
		fmt.Fprintf(os.Stderr, "ln: %s is not a valid scene: %v\n", file, err)
		return 1
	}
	if *w > 0 {
		sc.Width = *w
	}
	if *h > 0 {
		sc.Height = *h
	}
	if *fps > 0 {
		sc.FPS = *fps
	}
	st, err := build(sc, *look, filepath.Dir(file))
	if err != nil {
		fmt.Fprintln(os.Stderr, "ln:", err)
		return 1
	}
	if *frames != "" {
		if st.TL.Duration() == 0 {
			fmt.Fprintln(os.Stderr, "ln: the scene has no timeline, so there are no frames. Use -frame 0 for a picture.")
			return 1
		}
		if err := os.MkdirAll(*frames, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, "ln:", err)
			return 1
		}
		n := st.Frames()
		for i := 0; i < n; i++ {
			if err := st.Still(filepath.Join(*frames, fmt.Sprintf("frame_%05d.png", i)), float64(i)/st.FPS); err != nil {
				fmt.Fprintln(os.Stderr, "ln:", err)
				return 1
			}
		}
		fmt.Println("wrote", n, "frames to", *frames)
		return 0
	}
	ext := ".mp4"
	if *frame >= 0 {
		ext = ".png"
	}
	if *out == "" {
		*out = strings.TrimSuffix(filepath.Base(file), filepath.Ext(file)) + ext
	}
	picture := *frame >= 0
	switch strings.ToLower(filepath.Ext(*out)) {
	case ".png", ".svg", ".txt":
		picture = true
	}
	if picture {
		t := max(*frame, 0)
		fmt.Fprintf(os.Stderr, "drawing %s at %.2fs -> %s\n", file, t, *out)
		if err := writePicture(st, t, *out); err != nil {
			fmt.Fprintln(os.Stderr, "ln:", err)
			return 1
		}
	} else {
		if st.TL.Duration() == 0 {
			fmt.Fprintln(os.Stderr, "ln: the scene has no timeline, so there is nothing to play. Add steps to \"timeline\", or use -frame 0 for a picture.")
			return 1
		}
		fmt.Fprintf(os.Stderr, "drawing %s: %d frames, %.1fs -> %s\n", file, st.Frames(), st.TL.Duration(), *out)
		if err := st.Video(*out); err != nil {
			fmt.Fprintln(os.Stderr, "ln:", err)
			return 1
		}
	}
	fmt.Println("wrote", *out)
	return 0
}

// writePicture draws the scene at time t as a PNG, as SVG (the ink looks), or as text (the ascii look).
func writePicture(st *stage.Stage, t float64, out string) error {
	switch strings.ToLower(filepath.Ext(out)) {
	case ".svg":
		switch f := st.Frame(t).(type) {
		case *ink.Drawing:
			return os.WriteFile(out, []byte(f.SVG()), 0o644)
		case geom.Paths:
			return os.WriteFile(out, []byte(f.ToSVG(st.W, st.H)), 0o644)
		}
		return fmt.Errorf("svg output needs an ink or plotter look, not ascii")
	case ".txt":
		a, ok := st.Frame(t).(*ascii.Art)
		if !ok {
			return fmt.Errorf("text output needs the ascii look: add -look ascii")
		}
		return os.WriteFile(out, []byte(a.String()), 0o644)
	}
	return st.Still(out, t)
}

// runGo runs a Go program that draws with ln (for example one written by `ln art -code`). It needs the Go
// toolchain, and the file must sit in a module that requires ln.
func runGo(file string, rest []string) int {
	goBin, err := exec.LookPath("go")
	if err != nil {
		fmt.Fprintln(os.Stderr, "ln: running a .go file needs Go installed (https://go.dev/dl/)")
		return 1
	}
	var passed []string
	for i, a := range rest {
		if a == file && i == 0 {
			continue
		}
		passed = append(passed, a)
	}
	cmd := exec.Command(goBin, append([]string{"run", file}, passed...)...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return 1
	}
	return 0
}

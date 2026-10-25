// Package play is `ln play`: turn a frames file (made by `ln art -code frames.json`) into a video.
package play

import (
	"path/filepath"
	"strings"

	"github.com/fogleman/gg"
	"github.com/loyality7/ln/cli/setup"
	"github.com/loyality7/ln/cli/ui"
	"github.com/loyality7/ln/ln/film"
)

// Run is `ln play`.
func Run(args []string) int {
	fs := ui.NewFlagSet("play")
	out := fs.String("o", "", "output: .mp4 .gif .webm, or .png for one frame (default: <file>.mp4)")
	frame := fs.Int("frame", -1, "draw just this frame (a number from 0) as a picture")
	fps := fs.Float64("fps", 0, "frames per second (default: the speed the film was made at)")
	ui.Usage(fs, "ln play <frames.json> [options]",
		"Play a frames file as a video. The file holds every frame as shapes, so you need neither the original\nvideo nor Python, only ffmpeg. Make one with: ln art video.mp4 -code frames.json", nil,
		"ln play frames.json", "ln play frames.json -o clip.gif", "ln play frames.json -frame 40 -o one.png")
	if proceed, code := ui.ParseFlags(fs, args); !proceed {
		return code
	}
	if fs.NArg() != 1 {
		return ui.Fail("give one frames file, for example: ln play frames.json")
	}
	in := fs.Arg(0)
	f, err := film.Load(in)
	if err != nil {
		return ui.Fail("%v", err)
	}
	if *frame >= 0 {
		if *frame >= len(f.Frames) {
			return ui.Fail("the film has %d frames (0 to %d)", len(f.Frames), len(f.Frames)-1)
		}
		name := *out
		if name == "" {
			name = strings.TrimSuffix(in, filepath.Ext(in)) + ".png"
		}
		if err := gg.SavePNG(name, f.Frame(*frame).Image(0, 0)); err != nil {
			return ui.Fail("%v", err)
		}
		ui.OK("wrote %s", name)
		return 0
	}
	if err := setup.FFmpegOK(); err != nil {
		return ui.Fail("%v", err)
	}
	name := *out
	if name == "" {
		name = strings.TrimSuffix(in, filepath.Ext(in)) + ".mp4"
	}
	ui.Step("drawing %d frames", len(f.Frames))
	if err := f.Write(name, *fps); err != nil {
		return ui.Fail("%v", err)
	}
	ui.OK("wrote %s", name)
	return 0
}

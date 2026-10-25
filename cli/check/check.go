// Package check is `ln check`: measure a drawing against the video it was made from.
package check

import (
	"fmt"
	"os"

	"github.com/loyality7/ln/cli/setup"
	"github.com/loyality7/ln/cli/ui"
	"github.com/loyality7/ln/ln/vision"
)

// Run is `ln check`.
func Run(args []string) int {
	fs := ui.NewFlagSet("check")
	report := fs.String("report", "", "write the per-frame numbers here as JSON (default: <drawing>.report.json)")
	sheet := fs.String("sheet", "", "write a picture of the worst frames here (default: <drawing>.worst.png)")
	ui.Usage(fs, "ln check <original-video> <drawing-video> [options]",
		"Compare a video with the drawing made from it, frame by frame: how much the drawing flickers, what\nstructure it lost, and which subjects blink in and out.", nil,
		"ln check clip.mp4 clip_glow-colour.mp4")
	if proceed, code := ui.ParseFlags(fs, args); !proceed {
		return code
	}
	if fs.NArg() != 2 {
		return ui.Fail("give the original video and the drawing made from it")
	}
	src, dr := fs.Arg(0), fs.Arg(1)
	for _, f := range []string{src, dr} {
		if _, err := os.Stat(f); err != nil {
			return ui.Fail("%v", err)
		}
	}
	if m := setup.Missing(); len(m) > 0 {
		return ui.Fail("missing: %v (run `ln setup`)", m)
	}
	if *report == "" {
		*report = dr + ".report.json"
	}
	if *sheet == "" {
		*sheet = dr + ".worst.png"
	}
	text, err := vision.Check(src, dr, *report, *sheet)
	fmt.Println(text)
	if err != nil {
		return ui.Fail("%v", err)
	}
	return 0
}

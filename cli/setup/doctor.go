package setup

import (
	"fmt"
	"github.com/loyality7/ln/cli/ui"
	"os/exec"
	"strings"

	"github.com/loyality7/ln/ln/anim"
	"github.com/loyality7/ln/ln/vision"
)

// Version is the ln version, shown by doctor. The cli package sets it.
var Version = ""

const pipPackages = "opencv-python numpy scikit-image"

// Missing lists what `ln art` and `ln check` need that is not working, with the fix for each.
func Missing() []string {
	var out []string
	if bin, err := anim.FFmpeg(); err != nil {
		out = append(out, err.Error())
	} else if err := exec.Command(bin, "-version").Run(); err != nil {
		out = append(out, "ffmpeg at "+bin+" does not run")
	}
	py, err := vision.Python()
	if err != nil {
		return append(out, "python not found: install Python 3 from python.org (or set LN_PYTHON)")
	}
	if err := exec.Command(py, "-c", "import cv2, numpy, skimage").Run(); err != nil {
		out = append(out, "the Python packages are missing: pip install "+pipPackages)
	}
	return out
}

// FFmpegOK says whether ffmpeg works: what video output needs.
func FFmpegOK() error {
	bin, err := anim.FFmpeg()
	if err != nil {
		return err
	}
	if err := exec.Command(bin, "-version").Run(); err != nil {
		return fmt.Errorf("ffmpeg at %s does not run", bin)
	}
	return nil
}

// Doctor is `ln doctor`.
func Doctor(args []string) int {
	fs := ui.NewFlagSet("doctor")
	ui.Usage(fs, "ln doctor", "Check that everything ln needs is installed, and say what to do about anything that is not.", nil)
	if proceed, code := ui.ParseFlags(fs, args); !proceed {
		return code
	}
	good := true
	line := func(name string, err error, what string) {
		if err != nil {
			fmt.Printf("  %s  %-9s %s\n", ui.Red("MISSING"), name, err)
			good = false
			return
		}
		fmt.Printf("  %s       %-9s %s\n", ui.Green("ok"), name, ui.Dim(what))
	}
	fmt.Println(ui.Bold("ln " + Version))
	bin, err := anim.FFmpeg()
	line("ffmpeg", err, bin)
	py, err := vision.Python()
	if err != nil {
		line("python", fmt.Errorf("not found: install Python 3 from python.org (tick \"Add to PATH\")"), "")
	} else {
		v, _ := exec.Command(py, "--version").CombinedOutput()
		line("python", nil, py+" ("+strings.TrimSpace(string(v))+")")
		if e := exec.Command(py, "-c", "import cv2, numpy, skimage").Run(); e != nil {
			line("packages", fmt.Errorf("pip install %s", pipPackages), "")
		} else {
			line("packages", nil, "opencv-python, numpy, scikit-image")
		}
	}
	fmt.Println()
	if !good {
		fmt.Println("Run " + ui.Cyan("ln setup") + " to install what is missing.")
		fmt.Println(ui.Dim("(ln render, ln draw and ln mesh need only ffmpeg for video. ln art and ln check need Python as well.)"))
		return 1
	}
	fmt.Println("Everything is installed.")
	return 0
}

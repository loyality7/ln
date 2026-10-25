// Package cli is the `ln` command: it finds the command you typed and hands the rest of the line to it. Each command
// lives in its own package under cli/, and the terminal styling and option help are in cli/ui.
package cli

import (
	"fmt"
	"strings"

	"github.com/loyality7/ln/cli/art"
	"github.com/loyality7/ln/cli/check"
	"github.com/loyality7/ln/cli/draw"
	"github.com/loyality7/ln/cli/mesh"
	"github.com/loyality7/ln/cli/render"
	"github.com/loyality7/ln/cli/setup"
	"github.com/loyality7/ln/cli/ui"
)

// Version is set when a release is built: go build -ldflags "-X github.com/loyality7/ln/cli.Version=1.0.0".
var Version = "0.1.0"

// command is one thing `ln` can do.
type command struct {
	name    string
	group   string // the heading it is listed under
	summary string
	run     func(args []string) int
}

var commands []command

func init() {
	commands = []command{
		{"render", "Draw", "draw a 3D scene file as a video or a picture", render.Run},
		{"draw", "Draw", "draw a 2D drawing file: brush strokes, hatching, glowing light", draw.Run},
		{"art", "Draw", "redraw any video as line art", art.Run},
		{"mesh", "Draw", "convert, slice and voxelize 3D models", mesh.Run},
		{"check", "Draw", "measure a drawing against the video it came from", check.Run},
		{"new", "Learn", "write a starter file to edit (a scene or a drawing)", newCmd},
		{"looks", "Learn", "list the ways ln can draw", looksCmd},
		{"help", "Learn", "help for a command or a topic", helpCmd},
		{"install", "Setup", "put ln on your PATH and install what it needs", setup.Install},
		{"setup", "Setup", "install what ln needs (Python packages, ffmpeg)", setup.SetupCmd},
		{"doctor", "Setup", "check that everything ln needs is installed", setup.Doctor},
		{"version", "Setup", "print the version", func([]string) int { fmt.Println("ln", Version); return 0 }},
	}
}

// Main runs `ln` with the given arguments (not including the program name) and returns the exit code.
func Main(args []string) int {
	ui.Init()
	setup.Version = Version
	if len(args) == 0 {
		printUsage()
		return 0
	}
	switch args[0] {
	case "-h", "--help":
		printUsage()
		return 0
	case "-v", "--version":
		fmt.Println("ln", Version)
		return 0
	}
	c := find(args[0])
	if c == nil {
		ui.Fail("unknown command %q", args[0])
		if s := nearest(args[0]); s != "" {
			ui.Hint("did you mean %q?", s)
		}
		ui.Hint("run `ln` to see every command")
		return 2
	}
	return c.run(args[1:])
}

func find(name string) *command {
	for i := range commands {
		if commands[i].name == name {
			return &commands[i]
		}
	}
	return nil
}

// nearest is the command whose name is closest to what was typed, if any is close.
func nearest(name string) string {
	best, bestD := "", 3
	for _, c := range commands {
		if d := distance(name, c.name); d < bestD {
			best, bestD = c.name, d
		}
	}
	return best
}

func distance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}

func printUsage() {
	fmt.Printf("%s %s: draw scenes, drawings and video as line art\n\n", ui.Bold("ln"), Version)
	fmt.Println(ui.Bold("USAGE"))
	fmt.Println("  ln <command> [options]")
	fmt.Println()
	for _, g := range []string{"Draw", "Learn", "Setup"} {
		fmt.Println(ui.Bold(strings.ToUpper(g)))
		for _, c := range commands {
			if c.group == g {
				fmt.Printf("  %s%s%s\n", ui.Cyan(c.name), strings.Repeat(" ", 10-len(c.name)), c.summary)
			}
		}
		fmt.Println()
	}
	fmt.Println(ui.Bold("START HERE"))
	fmt.Println("  ln install                            once: put ln on your PATH and set it up")
	fmt.Println("  ln art clip.mp4                       redraw a video")
	fmt.Println("  ln new scene.json && ln render scene.json")
	fmt.Println()
	fmt.Println(ui.Dim("Run `ln help <command>` for the options of one command, or `ln help scene` for the scene file."))
}

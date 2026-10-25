package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/loyality7/ln/cli/draw"
	"github.com/loyality7/ln/cli/render"
	"github.com/loyality7/ln/cli/ui"
)

// helpCmd is `ln help [command | topic]`.
func helpCmd(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 0
	}
	if t, ok := topics[args[0]]; ok {
		fmt.Print(t)
		return 0
	}
	if c := find(args[0]); c != nil {
		return c.run([]string{"-h"})
	}
	ui.Fail("no help for %q", args[0])
	var names []string
	for _, c := range commands {
		names = append(names, c.name)
	}
	ui.Hint("commands: %s", strings.Join(names, ", "))
	ui.Hint("topics: scene, drawing, looks")
	return 2
}

// newCmd is `ln new`: it writes a starter file that works, to edit.
func newCmd(args []string) int {
	fs := ui.NewFlagSet("new")
	kind := fs.String("type", "scene", "scene | drawing")
	ui.Usage(fs, "ln new [file] [-type scene|drawing]",
		"Write a starter file that works, for you to edit. A scene is drawn with `ln render`, a drawing with `ln draw`.",
		nil,
		"ln new                         writes scene.json",
		"ln new -type drawing           writes drawing.json")
	if proceed, code := ui.ParseFlags(fs, args); !proceed {
		return code
	}
	name, body, next := "scene.json", render.Starter, "render"
	if *kind == "drawing" {
		name, body, next = "drawing.json", draw.Starter, "draw"
	} else if *kind != "scene" {
		return ui.Fail("type is scene or drawing, not %q", *kind)
	}
	if fs.NArg() > 0 {
		name = fs.Arg(0)
	}
	if _, err := os.Stat(name); err == nil {
		return ui.Fail("%s already exists; not overwriting it", name)
	}
	if err := os.WriteFile(name, []byte(body), 0o644); err != nil {
		return ui.Fail("%v", err)
	}
	ui.OK("wrote %s", ui.Bold(name))
	fmt.Fprintf(os.Stderr, "    ln %s %s -frame 0     one picture\n    ln %s %s              the whole thing\n    ln help %s            every field you can use\n", next, name, next, name, map[string]string{"render": "scene", "draw": "drawing"}[next])
	return 0
}

// looksCmd is `ln looks`.
func looksCmd(args []string) int {
	fs := ui.NewFlagSet("looks")
	ui.Usage(fs, "ln looks", "List the ways ln can draw.", nil)
	if proceed, code := ui.ParseFlags(fs, args); !proceed {
		return code
	}
	fmt.Print(topics["looks"])
	return 0
}

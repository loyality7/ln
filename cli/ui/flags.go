package ui

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// usage makes `-h` print a clean page for a command: what it does, its options under headings, and examples.
func Usage(fs *flag.FlagSet, synopsis, intro string, groups []Group, examples ...string) {
	fs.Usage = func() {
		out := os.Stdout
		fmt.Fprintf(out, "%s\n\n", intro)
		fmt.Fprintf(out, "%s\n  %s\n\n", Bold("USAGE"), synopsis)
		seen := map[string]bool{}
		for _, g := range groups {
			fmt.Fprintf(out, "%s\n", Bold(strings.ToUpper(g.Title)))
			for _, n := range g.Names {
				if f := fs.Lookup(n); f != nil {
					printFlag(out, f)
					seen[n] = true
				}
			}
			fmt.Fprintln(out)
		}
		var rest []string
		fs.VisitAll(func(f *flag.Flag) {
			if !seen[f.Name] {
				rest = append(rest, f.Name)
			}
		})
		if len(rest) > 0 {
			fmt.Fprintf(out, "%s\n", Bold("OTHER"))
			for _, n := range rest {
				printFlag(out, fs.Lookup(n))
			}
			fmt.Fprintln(out)
		}
		if len(examples) > 0 {
			fmt.Fprintf(out, "%s\n", Bold("EXAMPLES"))
			for _, e := range examples {
				fmt.Fprintf(out, "  %s\n", e)
			}
			fmt.Fprintln(out)
		}
	}
}

func printFlag(out *os.File, f *flag.Flag) {
	kind, text := flag.UnquoteUsage(f)
	head := "-" + f.Name
	if kind != "" {
		head += " <" + kind + ">"
	}
	def := ""
	if f.DefValue != "" && f.DefValue != "0" && f.DefValue != "false" && f.DefValue != "-1" {
		def = Dim(" (default " + f.DefValue + ")")
	}
	if len(head) > 22 {
		fmt.Fprintf(out, "  %s\n  %-22s %s%s\n", Cyan(head), "", text, def)
		return
	}
	fmt.Fprintf(out, "  %s%s %s%s\n", Cyan(head), strings.Repeat(" ", 22-len(head)), text, def)
}

// parseFlags parses a command's flags, and tells whether the command should go on. A help request is not an error.
func ParseFlags(fs *flag.FlagSet, args []string) (proceed bool, code int) {
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(flagsFirst(fs, args)); err != nil {
		if err == flag.ErrHelp {
			return false, 0
		}
		return false, 2
	}
	return true, 0
}

// Group is a heading and the flags listed under it, in a command's help.
type Group struct {
	Title string
	Names []string
}

// NewFlagSet makes a flag set that returns errors instead of exiting, so help is not an error.
func NewFlagSet(name string) *flag.FlagSet { return flag.NewFlagSet(name, flag.ContinueOnError) }

// flagsFirst moves the flags in front of the plain arguments, so `ln draw file.json -o out.png` works as well as
// `ln draw -o out.png file.json`.
func flagsFirst(fs *flag.FlagSet, args []string) []string {
	var flags, rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			rest = append(rest, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(a, "-") || a == "-" {
			rest = append(rest, a)
			continue
		}
		flags = append(flags, a)
		name := strings.TrimLeft(a, "-")
		if strings.Contains(name, "=") {
			continue
		}
		if f := fs.Lookup(name); f != nil {
			if b, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && b.IsBoolFlag() {
				continue
			}
		}
		if i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return append(flags, rest...)
}

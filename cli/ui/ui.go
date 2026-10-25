package ui

import (
	"fmt"
	"os"
	"strings"
)

// Terminal styling. Colour is used only on a real terminal, never when output is piped to a file, and never when
// NO_COLOR is set.

var colour bool

func Init() {
	colour = IsTerminal(os.Stderr) && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"
	if colour {
		colour = enableANSI()
	}
}

func IsTerminal(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func paint(code, s string) string {
	if !colour {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func Bold(s string) string   { return paint("1", s) }
func Dim(s string) string    { return paint("2", s) }
func Red(s string) string    { return paint("31", s) }
func Green(s string) string  { return paint("32", s) }
func Yellow(s string) string { return paint("33", s) }
func Cyan(s string) string   { return paint("36", s) }

// fail prints an error the same way everywhere and returns the exit code to use.
func Fail(format string, args ...any) int {
	fmt.Fprintln(os.Stderr, Red(Bold("error:"))+" "+fmt.Sprintf(format, args...))
	return 1
}

// hint prints a suggestion under an error.
func Hint(format string, args ...any) {
	fmt.Fprintln(os.Stderr, Dim("hint: ")+fmt.Sprintf(format, args...))
}

// step prints one line of progress.
func Step(format string, args ...any) {
	fmt.Fprintln(os.Stderr, Cyan("==>")+" "+fmt.Sprintf(format, args...))
}

// ok prints a finished line.
func OK(format string, args ...any) {
	fmt.Fprintln(os.Stderr, Green("ok")+"  "+fmt.Sprintf(format, args...))
}

// bar draws a progress bar of the given width.
func Bar(done, total, width int) string {
	if total <= 0 {
		return strings.Repeat(" ", width)
	}
	n := done * width / total
	if n > width {
		n = width
	}
	head := ">"
	if n >= width {
		head = "="
	}
	return strings.Repeat("=", max(n-1, 0)) + func() string {
		if n == 0 {
			return ""
		}
		return head
	}() + strings.Repeat(" ", width-n)
}

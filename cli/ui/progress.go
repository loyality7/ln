package ui

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"time"
)

var frameLine = regexp.MustCompile(`^frame (\d+)/(\d+)`)

// progress is a writer for the video reader's messages: lines like "frame 40/118" become a progress bar, and
// anything else (a Python error, say) is passed through untouched.
type progress struct {
	out   io.Writer
	label string
	buf   bytes.Buffer
	last  time.Time
	shown bool
	tty   bool
}

func NewProgress(label string) *progress {
	return &progress{out: os.Stderr, label: label, tty: IsTerminal(os.Stderr)}
}

func (p *progress) Write(b []byte) (int, error) {
	p.buf.Write(b)
	for {
		line, err := p.buf.ReadString('\n')
		if err != nil {
			p.buf.WriteString(line) // not a whole line yet
			break
		}
		line = line[:len(line)-1]
		if m := frameLine.FindStringSubmatch(line); m != nil {
			done, _ := strconv.Atoi(m[1])
			total, _ := strconv.Atoi(m[2])
			p.draw(done, total)
			continue
		}
		p.clear()
		fmt.Fprintln(p.out, line)
	}
	return len(b), nil
}

func (p *progress) draw(done, total int) {
	if !p.tty || time.Since(p.last) < 80*time.Millisecond {
		return
	}
	p.last = time.Now()
	pct := 0
	if total > 0 {
		pct = done * 100 / total
	}
	fmt.Fprintf(p.out, "\r%s %s [%s] %3d%%", Cyan("==>"), p.label, Bar(done, total, 28), pct)
	p.shown = true
}

func (p *progress) clear() {
	if p.shown {
		fmt.Fprint(p.out, "\r\x1b[2K")
		p.shown = false
	}
}

// Done clears the bar.
func (p *progress) Done() { p.clear() }

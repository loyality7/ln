package anim

import (
	"fmt"
	"github.com/loyality7/ln/ln/geom"
	"os"
	"path/filepath"
	"strings"
)

// Animation is a function from normalized time t in [0,1) to a rendered frame.
type Animation func(t float64) geom.Paths

// Frames renders n frames; t runs 0, 1/n, ... (n-1)/n so a loop wraps cleanly.
func (a Animation) Frames(n int) []geom.Paths {
	frames := make([]geom.Paths, n)
	for i := range frames {
		frames[i] = a(float64(i) / float64(n))
	}
	return frames
}

// WriteFrames writes frame_0000.svg ... into dir (feed to ffmpeg for video).
func (a Animation) WriteFrames(dir string, n int, width, height float64) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	for i, p := range a.Frames(n) {
		name := filepath.Join(dir, fmt.Sprintf("frame_%04d.svg", i))
		if err := p.WriteToSVG(name, width, height); err != nil {
			return err
		}
	}
	return nil
}

// WriteSVG writes one self-contained looping SVG (SMIL), no ffmpeg needed.
func (a Animation) WriteSVG(path string, n int, fps, width, height float64) error {
	var b strings.Builder
	fmt.Fprintf(&b, `<svg width="%f" height="%f" version="1.1" xmlns="http://www.w3.org/2000/svg">`+"\n", width, height)
	fmt.Fprintf(&b, `<g transform="translate(0,%f) scale(1,-1)">`+"\n", height)
	dur := float64(n) / fps
	for i, p := range a.Frames(n) {
		// show frame i only during [i/n, (i+1)/n) of the loop
		keyTimes := fmt.Sprintf("0;%f;%f;1", float64(i)/float64(n), float64(i+1)/float64(n))
		if i == 0 {
			keyTimes = fmt.Sprintf("0;0;%f;1", 1/float64(n))
		}
		fmt.Fprintf(&b, `<g display="none"><animate attributeName="display" calcMode="discrete" dur="%fs" repeatCount="indefinite" keyTimes="%s" values="%s"/>`+"\n",
			dur, keyTimes, map[bool]string{true: "inline;inline;none;none", false: "none;inline;none;none"}[i == 0])
		for _, path := range p {
			b.WriteString(path.ToSVG() + "\n")
		}
		b.WriteString("</g>\n")
	}
	b.WriteString("</g></svg>\n")
	return os.WriteFile(path, []byte(b.String()), 0644)
}

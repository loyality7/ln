package anim

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Home is the folder ln keeps its own tools in (LN_HOME, or .ln in the user's home folder).
func Home() string {
	if h := os.Getenv("LN_HOME"); h != "" {
		return h
	}
	d, err := os.UserHomeDir()
	if err != nil {
		return ".ln"
	}
	return filepath.Join(d, ".ln")
}

// FFmpeg finds ffmpeg: the path in LN_FFMPEG if set, then the copy `ln setup` puts in ln's own folder, then
// whatever is on PATH.
func FFmpeg() (string, error) {
	if p := os.Getenv("LN_FFMPEG"); p != "" {
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("LN_FFMPEG=%s does not exist", p)
		}
		return p, nil
	}
	name := "ffmpeg"
	if runtime.GOOS == "windows" {
		name = "ffmpeg.exe"
	}
	own := filepath.Join(Home(), "bin", name)
	if _, err := os.Stat(own); err == nil {
		return own, nil
	}
	p, err := exec.LookPath("ffmpeg")
	if err != nil {
		return "", fmt.Errorf("ffmpeg not found: run `ln setup`, or install ffmpeg and put it on your PATH (or set LN_FFMPEG)")
	}
	return p, nil
}

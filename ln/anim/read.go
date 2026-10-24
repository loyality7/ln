package anim

import (
	"bufio"
	"fmt"
	"image"
	"image/png"
	"io"
	"os/exec"
)

// ReadVideo decodes a video file (anything ffmpeg reads) and calls each with every
// frame, in order, scaled to width x height at fps frames per second. A height of 0
// keeps the aspect ratio. Return an error from each to stop early. Needs ffmpeg on PATH.
func ReadVideo(path string, width, height int, fps float64, each func(i int, frame image.Image) error) error {
	bin, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg not found on PATH: %w", err)
	}
	h := height
	if h == 0 {
		h = -2 // ffmpeg: keep aspect, even number
	}
	cmd := exec.Command(bin, "-v", "error", "-i", path, "-vf", fmt.Sprintf("fps=%g,scale=%d:%d", fps, width, h), "-f", "image2pipe", "-c:v", "png", "-")
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	in := bufio.NewReader(out)
	var failed error
	for i := 0; ; i++ {
		frame, err := png.Decode(in)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			failed = err
			break
		}
		if failed = each(i, frame); failed != nil {
			break
		}
	}
	if failed != nil {
		cmd.Process.Kill()
	}
	if err := cmd.Wait(); failed == nil && err != nil {
		failed = fmt.Errorf("ffmpeg: %w", err)
	}
	return failed
}

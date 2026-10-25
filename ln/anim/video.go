package anim

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os/exec"
	"path/filepath"
	"strings"
)

// Frame is anything that can rasterize itself: geom.Paths and *ink.Drawing both do.
type Frame interface {
	Image(width, height float64) image.Image
}

// FrameAnimation is Animation for any Frame type (e.g. ink drawings).
type FrameAnimation func(t float64) Frame

// WriteVideo renders n frames and streams them to ffmpeg; the container is
// picked from the extension (.mp4, .webm, .gif). Needs ffmpeg on PATH.
func (a Animation) WriteVideo(path string, n int, fps, width, height float64) error {
	return writeVideo(path, n, fps, width, height, func(t float64) Frame { return a(t) })
}

func (a FrameAnimation) WriteVideo(path string, n int, fps, width, height float64) error {
	return writeVideo(path, n, fps, width, height, a)
}

func writeVideo(path string, n int, fps, width, height float64, frame func(t float64) Frame) error {
	bin, err := FFmpeg()
	if err != nil {
		return err
	}
	args := []string{"-y", "-loglevel", "error", "-f", "image2pipe", "-framerate", fmt.Sprint(fps), "-c:v", "png", "-i", "-"}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".gif":
		args = append(args, "-vf", "split[a][b];[a]palettegen[p];[b][p]paletteuse", "-loop", "0")
	case ".webm":
		args = append(args, "-c:v", "libvpx-vp9", "-pix_fmt", "yuv420p")
	default: // h264 needs even dimensions
		args = append(args, "-vf", "scale=trunc(iw/2)*2:trunc(ih/2)*2", "-c:v", "libx264", "-crf", "14", "-pix_fmt", "yuv420p")
	}
	cmd := exec.Command(bin, append(args, path)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	in, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	var encErr error
	for i := 0; i < n && encErr == nil; i++ {
		encErr = png.Encode(in, frame(float64(i)/float64(n)).Image(width, height))
	}
	in.Close()
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ffmpeg: %w: %s", err, stderr.String())
	}
	return encErr
}

package anim

import (
	"image"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestReadVideoDeliversScaledFramesInOrder(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	clip := filepath.Join(t.TempDir(), "clip.mp4")
	if out, err := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "testsrc=size=160x90:rate=10", "-t", "1", "-pix_fmt", "yuv420p", clip).CombinedOutput(); err != nil {
		t.Fatalf("making the clip: %v %s", err, out)
	}
	n := 0
	err := ReadVideo(clip, 80, 0, 10, func(i int, f image.Image) error {
		if i != n || f.Bounds().Dx() != 80 || f.Bounds().Dy() != 46 { // 90*80/160 = 45, made even by ffmpeg
			t.Fatalf("frame %d: %v", i, f.Bounds())
		}
		n++
		return nil
	})
	if err != nil || n != 10 {
		t.Fatalf("read %d frames, err %v", n, err)
	}
}

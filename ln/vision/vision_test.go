package vision

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestReadFindsSteadyColourPatches(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	py, err := Python()
	if err != nil {
		t.Skip(err)
	}
	if err := exec.Command(py, "-c", "import cv2, numpy, skimage").Run(); err != nil {
		t.Skip("opencv-python, numpy and scikit-image not installed")
	}
	clip := filepath.Join(t.TempDir(), "clip.mp4")
	// three flat colour bars that slide sideways: the patches must be found and keep their colours
	if out, err := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "smptebars=size=320x180:rate=10", "-t", "1", "-pix_fmt", "yuv420p", clip).CombinedOutput(); err != nil {
		t.Fatalf("making the clip: %v %s", err, out)
	}
	frames := 0
	err = Read(clip, Options{Width: 320, FPS: 10, Colours: 6, MinArea: 100}, func(f Frame) error {
		frames++
		if len(f.Regions) < 4 {
			t.Fatalf("frame %d: only %d regions", f.Index, len(f.Regions))
		}
		if f.Regions[0].Area < f.Regions[len(f.Regions)-1].Area {
			t.Fatal("regions must come largest first")
		}
		if d := f.Drawing(Style{Colour: true}); len(d.Layers) != 1 || len(d.Layers[0].Fills) != len(f.Regions) {
			t.Fatal("every region must become a fill")
		}
		return nil
	})
	if err != nil || frames != 10 {
		t.Fatalf("read %d frames, err %v", frames, err)
	}
}

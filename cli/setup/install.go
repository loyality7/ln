package setup

import (
	"archive/zip"
	"fmt"
	"github.com/loyality7/ln/cli/ui"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/loyality7/ln/ln/anim"
	"github.com/loyality7/ln/ln/vision"
)

// SetupCmd installs the Python packages, and ffmpeg where it can do that without help (Windows).
// SetupCmd is `ln setup`.
func SetupCmd(args []string) int {
	fs := ui.NewFlagSet("setup")
	ui.Usage(fs, "ln setup", "Install what ln needs: the Python packages for `ln art`, and ffmpeg for video. On Windows ffmpeg is downloaded for you.\nPython itself you install once from python.org.", nil)
	if proceed, code := ui.ParseFlags(fs, args); !proceed {
		return code
	}
	return setup()
}

func setup() int {
	status := 0
	if py, err := vision.Python(); err != nil {
		ui.Fail("Python is not installed")
		ui.Hint("install Python 3 from https://www.python.org/downloads/ (tick \"Add Python to PATH\"), then run `ln setup` again")
		status = 1
	} else if exec.Command(py, "-c", "import cv2, numpy, skimage").Run() != nil {
		ui.Step("installing the Python packages: %s", pipPackages)
		cmd := exec.Command(py, append([]string{"-m", "pip", "install", "--quiet"}, strings.Fields(pipPackages)...)...)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			ui.Fail("pip failed: %v", err)
			status = 1
		} else {
			ui.OK("Python packages installed")
		}
	} else {
		ui.OK("Python packages already installed")
	}
	if _, err := anim.FFmpeg(); err == nil {
		ui.OK("ffmpeg already installed")
		return status
	}
	switch runtime.GOOS {
	case "windows":
		if err := fetchFFmpegWindows(); err != nil {
			ui.Fail("could not download ffmpeg: %v", err)
			ui.Hint("install it by hand from https://www.gyan.dev/ffmpeg/builds/ and put it on your PATH, or set LN_FFMPEG to its path")
			status = 1
		}
	case "darwin":
		ui.Fail("ffmpeg is not installed")
		ui.Hint("install it with:  brew install ffmpeg")
		status = 1
	default:
		ui.Fail("ffmpeg is not installed")
		ui.Hint("install it with your package manager, for example:  sudo apt install ffmpeg")
		status = 1
	}
	return status
}

// fetchFFmpegWindows downloads a static ffmpeg build into ln's own folder, so ln finds it first.
func fetchFFmpegWindows() error {
	const url = "https://github.com/BtbN/FFmpeg-Builds/releases/latest/download/ffmpeg-master-latest-win64-gpl.zip"
	ui.Step("downloading ffmpeg (about 150 MB) from github.com/BtbN/FFmpeg-Builds")
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("%s", resp.Status)
	}
	tmp, err := os.CreateTemp("", "ffmpeg-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return err
	}
	tmp.Close()
	z, err := zip.OpenReader(tmp.Name())
	if err != nil {
		return err
	}
	defer z.Close()
	dir := filepath.Join(anim.Home(), "bin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, f := range z.File {
		if !strings.HasSuffix(f.Name, "/bin/ffmpeg.exe") {
			continue
		}
		r, err := f.Open()
		if err != nil {
			return err
		}
		dst, err := os.Create(filepath.Join(dir, "ffmpeg.exe"))
		if err != nil {
			r.Close()
			return err
		}
		_, err = io.Copy(dst, r)
		r.Close()
		dst.Close()
		if err != nil {
			return err
		}
		ui.OK("ffmpeg installed to %s", filepath.Join(dir, "ffmpeg.exe"))
		return nil
	}
	return fmt.Errorf("ffmpeg.exe was not in the download")
}

// Install copies ln into its own folder, puts that folder on the PATH, and runs setup.
// Install is `ln install`.
func Install(args []string) int {
	fs := ui.NewFlagSet("install")
	skip := fs.Bool("nosetup", false, "only put ln on the PATH; do not install Python packages or ffmpeg")
	ui.Usage(fs, "ln install [options]",
		"Make `ln` a command you can run from any folder: copy it to "+filepath.Join(anim.Home(), "bin")+", add that folder to your PATH,\nand install what it needs. Run it once, from wherever you saved the download.", nil)
	if proceed, code := ui.ParseFlags(fs, args); !proceed {
		return code
	}
	self, err := os.Executable()
	if err != nil {
		return ui.Fail("cannot find where ln is running from: %v", err)
	}
	dir := filepath.Join(anim.Home(), "bin")
	name := "ln"
	if runtime.GOOS == "windows" {
		name = "ln.exe"
	}
	dest := filepath.Join(dir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ui.Fail("%v", err)
	}
	if same, _ := sameFile(self, dest); !same {
		if err := copyFile(self, dest); err != nil {
			return ui.Fail("could not copy ln to %s: %v", dest, err)
		}
	}
	ui.OK("ln installed to %s", dest)
	added, err := addToPath(dir)
	switch {
	case err != nil:
		ui.Fail("could not add %s to your PATH: %v", dir, err)
		ui.Hint("add it by hand, then open a new terminal")
	case added:
		ui.OK("added %s to your PATH", dir)
		fmt.Fprintln(os.Stderr, "    "+ui.Bold("open a new terminal")+" for the PATH change to take effect")
	default:
		ui.OK("%s is already on your PATH", dir)
	}
	if *skip {
		return 0
	}
	fmt.Fprintln(os.Stderr)
	return setup()
}

func sameFile(a, b string) (bool, error) {
	ia, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	ib, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	return os.SameFile(ia, ib), nil
}

func copyFile(from, to string) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(to, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// addToPath puts dir on the user's PATH for future terminals. It says whether it changed anything.
func addToPath(dir string) (bool, error) {
	if runtime.GOOS == "windows" {
		script := `$d = '` + strings.ReplaceAll(dir, "'", "''") + `'
$p = [Environment]::GetEnvironmentVariable('Path','User')
if ($p -split ';' -contains $d) { 'present' } else { [Environment]::SetEnvironmentVariable('Path', ($p.TrimEnd(';') + ';' + $d), 'User'); 'added' }`
		out, err := exec.Command("powershell", "-NoProfile", "-Command", script).CombinedOutput()
		if err != nil {
			return false, fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
		}
		return strings.TrimSpace(string(out)) == "added", nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}
	line := `export PATH="` + dir + `:$PATH"`
	changed := false
	for _, rc := range []string{".zshrc", ".bashrc", ".profile"} {
		path := filepath.Join(home, rc)
		data, err := os.ReadFile(path)
		if err != nil && rc != ".profile" {
			continue
		}
		if strings.Contains(string(data), line) {
			return false, nil
		}
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return false, err
		}
		fmt.Fprintf(f, "\n# added by ln install\n%s\n", line)
		f.Close()
		changed = true
		if rc != ".profile" {
			break
		}
	}
	return changed, nil
}

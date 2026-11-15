//go:build draw

package main

import (
	"bytes"
	"errors"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/janearc/slayer/internal/commandtest"
)

// the old command keeps its short flags: -w, -h and -n set the size and the
// rays of the picture it draws.
func TestHillsideStillTakesItsShortSizeFlags(t *testing.T) {
	const (
		// wide is the width of the picture in pixels, defined as the width
		// asked for with -w.
		wide = 48
		// tall is the height of the picture in pixels, defined as the height
		// asked for with -h.
		tall = 27
	)
	binary := filepath.Join(t.TempDir(), "hillside")
	if err := commandtest.Build(".", ".", binary, ""); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "hillside.png")
	run := exec.Command(binary, "-set", "../../sets/widow.yaml", "-w", strconv.Itoa(wide),
		"-h", strconv.Itoa(tall), "-n", "1", "-out", out)
	run.Env = append(os.Environ(), "GOCOVERDIR="+t.TempDir())
	told, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: %s", err, told)
	}
	written, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	picture, err := png.Decode(bytes.NewReader(written))
	if err != nil || picture.Bounds().Dx() != wide ||
		picture.Bounds().Dy() != tall {
		t.Fatalf("drew %v, %v; wanted %dx%d", picture.Bounds(), err, wide, tall)
	}
}

// usageExit is the number hillside exits with for a flag it cannot take;
// defined: two.
const usageExit = 2

// the command takes the terrain flags in its own form, one hyphen or two,
// and refuses one given with no set, by name, exit 2.
func TestHillsideTakesTheTerrainFlagsAndRefusesThemWithNoSet(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "hillside")
	if err := commandtest.Build(".", ".", binary, ""); err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{"-check-terrain", "--fetch-terrain"} {
		run := exec.Command(binary, flag)
		run.Env = append(os.Environ(), "GOCOVERDIR="+t.TempDir())
		told, err := run.CombinedOutput()
		var stopped *exec.ExitError
		if !errors.As(err, &stopped) || stopped.ExitCode() != usageExit ||
			!bytes.Contains(told, []byte("need -set")) {
			t.Errorf("%s: %v: %s", flag, err, told)
		}
	}
}

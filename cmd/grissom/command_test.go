package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/internal/commandtest"
)

var viewerCommand string

// TestMain keeps one real viewer executable for all command tests. Go's
// coverage directory is inherited by its children and collected by Go itself.
func TestMain(m *testing.M) {
	directory, err := os.MkdirTemp("", "slayer-grissom-command-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	viewerCommand = filepath.Join(directory, "grissom")
	err = commandtest.Build(".", ".", viewerCommand, testing.CoverMode())
	if err != nil {
		os.RemoveAll(directory)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(directory)
	os.Exit(code)
}

func TestViewerHelpAndMissingColourwayDirectoryUseRealProcess(t *testing.T) {
	help, err := viewerRun(t, "-help")
	if err != nil || !strings.Contains(string(help), "-earth-depth") || !strings.Contains(string(help), "-colourways") {
		t.Fatalf("help %q: %v", help, err)
	}
	missing := filepath.Join(t.TempDir(), "no-colourways")
	out, err := viewerRun(t, "-colourways", missing)
	if err == nil || !strings.Contains(string(out), "no colourway in "+missing) {
		t.Fatalf("missing colourways %q: %v", out, err)
	}
}

func TestViewerRendersAllThreeViewsFromSyntheticInputs(t *testing.T) {
	data, earth, sky, themes := viewerFiles(t)
	prefix := filepath.Join(t.TempDir(), "viewer")
	args := []string{"-data", data, "-earth", earth, "-sky", sky, "-colourways", themes, "-depth", "1", "-earth-depth", "1", "-wide-depth", "1", "-look", "-122.4,37.7", "-shot", prefix, "-view", "-1"}
	out, err := viewerRun(t, args...)
	if err != nil {
		t.Fatalf("real viewer: %v\n%s", err, out)
	}
	hashes := map[[32]byte]bool{}
	for _, name := range []string{"sky", "earth", "standing"} {
		file, err := os.Open(prefix + "-" + name + ".png")
		if err != nil {
			t.Fatal(err)
		}
		picture, err := png.Decode(file)
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
		if picture.Bounds().Dx() != 1400 || picture.Bounds().Dy() != 1000 {
			t.Fatalf("%s dimensions %v", name, picture.Bounds())
		}
		first := picture.At(0, 0)
		varied := false
		for y := 0; y < 1000; y += 7 {
			for x := 0; x < 1400; x += 7 {
				_, _, _, a := picture.At(x, y).RGBA()
				if a != 65535 {
					t.Fatalf("%s alpha %d", name, a)
				}
				varied = varied || picture.At(x, y) != first
			}
		}
		if !varied {
			t.Fatalf("%s is a flat image", name)
		}
		raw, err := os.ReadFile(prefix + "-" + name + ".png")
		if err != nil {
			t.Fatal(err)
		}
		hashes[sha256.Sum256(raw)] = true
		if name == "earth" {
			// The requested western longitude sees the fixture's blue hemisphere.
			// The coarse central hex marks the observer in white; inspect an
			// unmarked patch and average over thin outlines.
			red, blue := uint64(0), uint64(0)
			for y := 540; y < 580; y += 2 {
				for x := 1160; x < 1200; x += 2 {
					r, _, b, _ := picture.At(x, y).RGBA()
					red += uint64(r)
					blue += uint64(b)
				}
			}
			r, g, b, _ := picture.At(700, 650).RGBA()
			if r != 65535 || g != 65535 || b != 65535 {
				t.Fatal("observer hex lost its white marker")
			}
			if blue <= red {

				t.Fatalf("western hemisphere lost its blue photograph: R%d B%d", red, blue)
			}
		}
	}
	if len(hashes) != 3 {
		t.Fatal("the three views produced duplicate PNGs")
	}
}

func TestViewerReportsRealInputAndOutputFailures(t *testing.T) {
	data, earth, sky, themes := viewerFiles(t)
	// Sky layers load concurrently; hold the requested root and file cause.
	missingSky := filepath.Join(t.TempDir(), "missing-sky")
	base := []string{"-data", data, "-earth", earth, "-sky", sky, "-colourways", themes, "-depth", "1", "-earth-depth", "1", "-wide-depth", "1"}
	for _, test := range []struct {
		name string
		args []string
		want []string
	}{
		{"look", []string{"-look", "not-a-coordinate"}, []string{"invalid syntax"}},
		{"sky", []string{"-data", missingSky}, []string{missingSky, "no such file"}},
		{"earth", []string{"-earth", filepath.Join(t.TempDir(), "missing-earth")}, []string{"ne_50m_land.geojson"}},
		{"output", []string{"-shot", filepath.Join(t.TempDir(), "missing", "shot"), "-view", "0"}, []string{"no such file"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			args := append(append([]string{}, base...), test.args...)
			out, err := viewerRun(t, args...)
			if err == nil {
				t.Fatalf("accepted invalid %s request: %s", test.name, out)
			}
			for _, expected := range test.want {
				if !strings.Contains(string(out), expected) {
					t.Fatalf("missing %q: %v\n%s", expected, err, out)
				}
			}
		})
	}
}

// viewerRun bounds the actual viewer process, including unexpected UI startup.
func viewerRun(t *testing.T, args ...string) ([]byte, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	return exec.CommandContext(ctx, viewerCommand, args...).CombinedOutput()
}

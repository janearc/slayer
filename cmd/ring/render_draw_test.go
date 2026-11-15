//go:build draw

package main

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func ringPrint(t *testing.T, path string) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	picture, err := png.Decode(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if picture.Bounds() != image.Rect(0, 0, 8, 6) {
		t.Fatal(picture.Bounds())
	}
}

func TestRingCommandDrawsVisibleAndFalseColour(t *testing.T) {
	for _, radiation := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "ring.png")
		args := append(ringArgs(path), "-bands")
		if radiation {
			args = append(args, "-radiation")
		}
		out, err := runCoverageCommand(t, args...)
		if err != nil {
			t.Fatalf("%v %q", err, out)
		}
		for _, band := range []string{"x-ray", "gamma ray", "infrared"} {
			if !strings.Contains(string(out), band) {
				t.Fatalf("missing %s: %q", band, out)
			}
		}
		ringPrint(t, path)
		ringPrint(t, strings.TrimSuffix(path, ".png")+"-bands.png")
	}
}

func TestRingCommandReportsBothWriteStages(t *testing.T) {
	file := filepath.Join(t.TempDir(), "held")
	if err := os.WriteFile(file, []byte("held"), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "ring.png")
	if err := os.Mkdir(strings.TrimSuffix(path, ".png")+"-bands.png", 0700); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{ringArgs(filepath.Join(file, "x.png")), ringArgs(t.TempDir()), append(ringArgs(path), "-bands")} {
		out, err := runCoverageCommand(t, args...)
		if err == nil || !strings.Contains(string(out), "ring:") {
			t.Fatalf("%v %q", err, out)
		}
	}
}

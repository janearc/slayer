//go:build draw

package main

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/raw"
)

func moonArgs(out string) []string {
	return []string{"-w", "9", "-h", "9", "-n", "1", "-moon", "1.5", "-out", out}
}

func TestRealMoonriseWritesLinearRawAndOptionalPreview(t *testing.T) {
	for _, preview := range []bool{false, true} {
		dir := t.TempDir()
		path := filepath.Join(dir, "moon.pfm")
		args := moonArgs(path)
		print := filepath.Join(dir, "moon.png")
		if preview {
			args = append(args, "-preview", print)
		}
		out, err := runCoverageCommand(t, args...)
		if err != nil || !strings.Contains(string(out), "9x9, 1 rays a pixel") {
			t.Fatalf("%v %q", err, out)
		}
		picture, err := raw.Read(path)
		if err != nil {
			t.Fatal(err)
		}
		if picture.W != 9 || picture.H != 9 || picture.At(4, 4).R <= 0 {
			t.Fatal("raw lost the moon")
		}
		if preview {
			f, err := os.Open(print)
			if err != nil {
				t.Fatal(err)
			}
			shown, err := png.Decode(f)
			f.Close()
			if err != nil || shown.Bounds() != image.Rect(0, 0, 9, 9) {
				t.Fatalf("%v %v", shown, err)
			}
		}
	}
}

func TestMoonrisePropagatesRecipeAndBothOutputFailures(t *testing.T) {
	for _, args := range [][]string{
		append(moonArgs(filepath.Join(t.TempDir(), "x.pfm")), "-kind", "unknown"),
		moonArgs(t.TempDir()),
		append(moonArgs(filepath.Join(t.TempDir(), "x.pfm")), "-preview", t.TempDir()),
	} {
		out, err := runCoverageCommand(t, args...)
		if err == nil || !strings.Contains(string(out), "moonrise:") {
			t.Fatalf("%v %q", err, out)
		}
	}
}

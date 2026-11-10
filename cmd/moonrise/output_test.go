package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/frame"
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
func TestPlainPreviewClipsAndTransfersLinearLight(t *testing.T) {
	picture := frame.New(3, 1)
	picture.Pix = []frame.Pixel{{R: -1, G: 0, B: 2}, {R: .5, G: .5, B: .5}, {R: 1, G: 1, B: 1}}
	path := filepath.Join(t.TempDir(), "plain.png")
	if err := plain(picture, 0, path); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := png.Decode(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	for x, want := range []color.RGBA{{0, 0, 255, 255}, {188, 188, 188, 255}, {255, 255, 255, 255}} {
		if got.At(x, 0) != want {
			t.Fatalf("pixel%d: %v want %v", x, got.At(x, 0), want)
		}
	}
}

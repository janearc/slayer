package main

import (
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/render/frame"
)

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

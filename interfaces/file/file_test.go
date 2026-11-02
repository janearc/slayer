package file

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestASextantPixelIsDrawnThreeByFour(t *testing.T) {
	if w, h := Block(0.75); w != 3 || h != 4 {
		t.Fatalf("got %d by %d", w, h)
	}
	if w, h := Block(1); w != 1 || h != 1 {
		t.Fatalf("a square pixel got %d by %d", w, h)
	}
}

func TestAPngKeepsThePixelsShape(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 3))
	for y := 0; y < 3; y++ {
		for x := 0; x < 4; x++ {
			img.SetRGBA(x, y, color.RGBA{uint8(40 * x), 200, uint8(60 * y), 255})
		}
	}
	path := filepath.Join(t.TempDir(), "shown.png")
	if err := PNG(path, img, 0.75); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	back, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if got := back.Bounds().Size(); got != image.Pt(4*3, 3*4) {
		t.Fatalf("png is %v, want 12 by 12", got)
	}
}

// every pixel is drawn in its own block: the pixel at x, y fills the
// block from bw x, bh y, and no other.
func TestEachPixelFillsItsOwnBlock(t *testing.T) {
	img := image.NewRGBA(image.Rect(2, 5, 6, 8))
	for y := 5; y < 8; y++ {
		for x := 2; x < 6; x++ {
			img.SetRGBA(x, y, color.RGBA{uint8(x), uint8(y), 7, 255})
		}
	}
	grown := Grow(img, 3, 4)
	for y := 0; y < 12; y++ {
		for x := 0; x < 12; x++ {
			want := color.RGBA{uint8(2 + x/3), uint8(5 + y/4), 7, 255}
			if got := grown.RGBAAt(x, y); got != want {
				t.Fatalf("%d, %d is %v, want %v", x, y, got, want)
			}
		}
	}
}

// a block narrower or shorter than a pixel is drawn one pixel, not as
// nothing or backwards.
func TestABlockOfLessThanAPixelIsOne(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 3))
	for _, size := range [][2]int{{0, 1}, {-1, 1}, {1, 0}, {-3, -3}} {
		if got := Grow(img, size[0], size[1]).Bounds(); got !=
			image.Rect(0, 0, 4, 3) {
			t.Errorf("a block of %v draws %v", size, got)
		}
	}
}

// a picture that cannot be written leaves the file at its path as it
// was: nothing, or a picture of nothing, takes its place.
func TestAPictureThatCannotBeWrittenKeepsTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "precious.png")
	if err := os.WriteFile(path, []byte("precious"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, img := range []image.Image{nil,
		image.NewRGBA(image.Rect(0, 0, 0, 0))} {
		if err := PNG(path, img, 1); err == nil {
			t.Errorf("%v was written", img)
		}
		if kept, _ := os.ReadFile(path); string(kept) != "precious" {
			t.Fatalf("the file is now %q", kept)
		}
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Errorf("%d files beside the picture", len(entries)-1)
	}
}

// a picture is written readable by all and writable by its owner.
func TestAPictureIsWrittenInItsMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mode.png")
	if err := PNG(path, image.NewRGBA(image.Rect(0, 0, 1, 1)), 1); err !=
		nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(path); err != nil ||
		info.Mode().Perm() != 0o644 {
		t.Errorf("%v %v", info.Mode(), err)
	}
}

// a block is never more than 8 either way: the nearest shape within
// that, and 1 by 1 for a shape none can draw or a width that is none.
func TestABlockIsEightAtMost(t *testing.T) {
	for _, want := range []struct {
		pixel float64
		w, h  int
	}{
		{0.125, 1, 8}, {1.0 / 9, 1, 8}, {8, 8, 1}, {9, 1, 1},
		{0.05, 1, 1}, {0, 1, 1}, {-1, 1, 1}, {math.NaN(), 1, 1},
		{math.Inf(1), 1, 1},
	} {
		if w, h := Block(want.pixel); w != want.w || h != want.h {
			t.Errorf("a pixel %v wide draws %d by %d, want %d by %d",
				want.pixel, w, h, want.w, want.h)
		}
	}
}

// a picture with nowhere to go is an error, and a path that is a
// directory keeps it, with nothing left beside it.
func TestAPictureWithNowhereToGoIsAnError(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	root := t.TempDir()
	if err := PNG(filepath.Join(root, "no", "such.png"), img, 1); err ==
		nil {
		t.Error("a picture was written into no directory")
	}
	taken := filepath.Join(root, "taken")
	if err := os.MkdirAll(filepath.Join(taken, "inside"), 0o755); err !=
		nil {
		t.Fatal(err)
	}
	if err := PNG(taken, img, 1); err == nil {
		t.Error("a picture took a directory's place")
	}
	if entries, _ := os.ReadDir(root); len(entries) != 1 {
		t.Errorf("%d files left beside the directory", len(entries)-1)
	}
}

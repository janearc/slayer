package file

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestSequencePNGPreservesDecodedPixelsAndAtomicRefusals(t *testing.T) {
	img := image.NewRGBA(image.Rect(7, 11, 64, 43))
	for y := range img.Bounds().Dy() {
		for x := range img.Bounds().Dx() {
			a := uint8(255)
			if x%5 == 0 {
				a = 128
			}
			img.SetRGBA(x+7, y+11, color.RGBA{uint8(x*3) % a, uint8(y*5) % a, uint8(x+y) % a, a})
		}
	}
	dir := t.TempDir()
	legacy := filepath.Join(dir, "legacy.png")
	fast := filepath.Join(dir, "fast.png")
	if err := PNG(legacy, img, 1); err != nil {
		t.Fatal(err)
	}
	if err := SequencePNG(fast, img); err != nil {
		t.Fatal(err)
	}
	read := func(path string) image.Image {
		input, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer input.Close()
		made, err := png.Decode(input)
		if err != nil {
			t.Fatal(err)
		}
		return made
	}
	a, b := read(legacy), read(fast)
	if a.Bounds() != b.Bounds() {
		t.Fatal("fast writer changed dimensions")
	}
	for y := range a.Bounds().Dy() {
		for x := range a.Bounds().Dx() {
			r, g, blue, alpha := a.At(x, y).RGBA()
			rr, gg, bb, aa := b.At(x, y).RGBA()
			if r != rr || g != gg || blue != bb || alpha != aa {
				t.Fatalf("changed decoded pixel %d,%d", x, y)
			}
		}
	}
	before, err := os.ReadFile(fast)
	if err != nil {
		t.Fatal(err)
	}
	if err := SequencePNG(fast, nil); err == nil {
		t.Fatal("nil image accepted")
	}
	after, err := os.ReadFile(fast)
	if err != nil || string(before) != string(after) {
		t.Fatal("refusal changed old image")
	}
	if err := SequencePNG(filepath.Join(dir, "missing", "frame.png"), img); err == nil {
		t.Fatal("unwritable output accepted")
	}
}

func BenchmarkSequencePNG(b *testing.B) {
	img := image.NewRGBA(image.Rect(0, 0, 3840, 2160))
	// A seeded nonuniform field exercises filters and compression.
	bits := uint64(71)
	for i := range img.Pix {
		bits ^= bits << 13
		bits ^= bits >> 7
		bits ^= bits << 17
		img.Pix[i] = byte(bits)
	}
	target := filepath.Join(b.TempDir(), "frame.png")
	b.ResetTimer()
	for range b.N {
		if err := SequencePNG(target, img); err != nil {
			b.Fatal(err)
		}
	}
}

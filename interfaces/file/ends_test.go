package file

import (
	"image"
	"image/color"
	"testing"
	"time"
)

// patience is how long a call that has only a few small loops to run may
// take before the test says its loop does not end. Picked by the test
// author, 2026-11-17: these calls take microseconds.
const patience = 10 * time.Second

// finishes runs work and stops the test if it has not come back in
// patience, since a loop that counts the wrong way never does and would
// otherwise hold the whole run until its timeout.
func finishes(t *testing.T, what string, work func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		work()
	}()
	select {
	case <-done:
	case <-time.After(patience):
		t.Fatalf("%s did not come back in %v: a loop does not end", what,
			patience)
	}
}

// Block comes back for a pixel of any shape, including the ones no block
// fits, and for a pixel that is not a number.
func TestBlockComesBackForEveryShape(t *testing.T) {
	for _, pixel := range []float64{0.75, 1, 1.0 / 9, 0.05, 9, 0.5} {
		finishes(t, "Block", func() { Block(pixel) })
	}
	var w, h int
	finishes(t, "Block", func() { w, h = Block(0.5) })
	if w != 1 || h != 2 {
		t.Errorf("a pixel half as wide as it is tall draws %d by %d, "+
			"not 1 by 2", w, h)
	}
}

// Grow comes back for a picture grown wider and taller than itself, with
// every block filled.
func TestGrowComesBackAndFillsEveryBlock(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.SetRGBA(1, 1, color.RGBA{9, 8, 7, 255})
	var grown *image.RGBA
	finishes(t, "Grow", func() { grown = Grow(img, 2, 3) })
	if grown.Bounds() != image.Rect(0, 0, 4, 6) {
		t.Fatalf("a picture of 2 by 2 grown 2 by 3 is %v", grown.Bounds())
	}
	if got := grown.RGBAAt(3, 5); got != (color.RGBA{9, 8, 7, 255}) {
		t.Errorf("the last block's last pixel is %v", got)
	}
	if got := grown.RGBAAt(2, 3); got != (color.RGBA{9, 8, 7, 255}) {
		t.Errorf("the last block's first pixel is %v", got)
	}
	if got := grown.RGBAAt(0, 0); got != (color.RGBA{}) {
		t.Errorf("the first block's first pixel is %v", got)
	}
}

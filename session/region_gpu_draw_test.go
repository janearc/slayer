//go:build draw && darwin

package session

import (
	"image"
	"testing"

	"github.com/janearc/slayer/scene/thing"
)

func TestSoftwareGPURegionDrawsTheFullFramesPixels(t *testing.T) {
	full := usualAsked()
	full.width, full.height, full.samples, full.onGPU = 12, 10, 2, true
	whole, _, err := drawMeasured(thing.NewWorld(), full, 0)
	if err != nil {
		t.Fatal(err)
	}
	box := image.Rect(2, 3, 7, 8)
	partAsk := full
	partAsk.region = &box
	part, _, err := drawMeasured(thing.NewWorld(), partAsk, 0)
	if err != nil {
		t.Fatal(err)
	}
	if part.W != box.Dx() || part.H != box.Dy() {
		t.Fatalf("region is %dx%d", part.W, part.H)
	}
	for y := 0; y < part.H; y++ {
		for x := 0; x < part.W; x++ {
			if part.At(x, y) != whole.At(box.Min.X+x, box.Min.Y+y) {
				t.Fatalf("pixel %d,%d differs", x, y)
			}
		}
	}
}

package searching

import (
	"bytes"
	"testing"

	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
)

// TestExposurePreservesDisplay compares the deferred path to the former
// immediate tone-and-grade sequence, including bloom and a second Develop.
func TestExposurePreservesDisplay(t *testing.T) {
	linear := frame.New(8, 4)
	for index := range linear.Pix {
		linear.Pix[index] = frame.Pixel{R: float32(index) / 3, G: .3, B: .8, Cover: 1}
	}
	tone := colour.Tone{Exposure: 1.5, Knee: .5, Top: 12, Bloom: 1}
	want := tone.Develop(linear)
	grade(want, .6)
	exposure := &Exposure{linear: linear, tone: tone, graded: true, saturation: .6}
	got := exposure.Develop()
	if !bytes.Equal(got.Pix, want.Pix) {
		t.Fatal("moving display work changed the pixels")
	}
	if exposure.linear != nil {
		t.Fatal("completed exposure still retains the linear frame")
	}
	clear(linear.Pix)
	if second := exposure.Develop(); second != got || !bytes.Equal(second.Pix, want.Pix) {
		t.Fatal("second Develop changed the completed picture")
	}
}

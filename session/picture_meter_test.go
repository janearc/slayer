package session

import (
	"math"
	"testing"

	"github.com/janearc/slayer/render/frame"
)

// TestTheLogAverageIsTheGeometricMean says that the meter reads a
// picture of two greys, one a hundred times the other, as their
// geometric mean: a tenth of the brighter, not half of it, so one bright
// region does not set the exposure alone.
func TestTheLogAverageIsTheGeometricMean(t *testing.T) {
	drawn := &frame.Frame{W: 2, H: 1, Pix: []frame.Pixel{
		{R: 1, G: 1, B: 1}, {R: 0.01, G: 0.01, B: 0.01}}}
	got, lit := logAverage(drawn)
	if !lit || math.Abs(got-0.1) > 1e-6 {
		t.Errorf("log-average %v (lit %v), want 0.1", got, lit)
	}
}

// TestADarkPictureMetersNothing is the anti-test: a picture of no light,
// or of none at all, is not lit, and the meter leaves the exposure.
func TestADarkPictureMetersNothing(t *testing.T) {
	dark := &frame.Frame{W: 2, H: 1, Pix: make([]frame.Pixel, 2)}
	if _, lit := logAverage(dark); lit {
		t.Error("a black picture is lit")
	}
	if _, lit := logAverage(nil); lit {
		t.Error("no picture is lit")
	}
}

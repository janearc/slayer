package colour

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// bloomSide is the side of the frame, in pixels, and bloomSigma the
// bloom's sigma in pixels. Chosen: the frame holds eight sigmas each way.
const (
	bloomSide  = 81
	bloomSigma = 5.0
)

// pointOf is a frame of no light with one pixel of the given brightness
// at its middle, and the bloom of it at an exposure of one.
func pointOf(brightness float64) (bloomed []maths.Light, middle int) {
	lights := make([]maths.Light, bloomSide*bloomSide)
	middle = bloomSide / 2
	lights[middle*bloomSide+middle] = maths.Grey(brightness)
	return Bloom(lights, bloomSide, bloomSide, 1, bloomSigma), middle
}

// A star so bright that its glow is still seen past the usual reach of a
// bloom glows round, not square. A point of a million whites glows as much
// at a distance along the row as at the same distance on the diagonal, past
// three sigmas, where a glow cut off at three sigmas is nothing along the
// row and a square corner on the diagonal.
func TestAVeryBrightPointBloomsRound(t *testing.T) {
	bloomed, middle := pointOf(1e6)
	const along = 4 * bloomSigma // 20 pixels, four sigmas
	row := bloomed[middle*bloomSide+middle+along].G
	diagonal := bloomed[(middle+14)*bloomSide+middle+14].G // 19.8 pixels
	if !(row > 0 && diagonal > 0) {
		t.Fatalf("the glow is %v along the row and %v on the diagonal",
			row, diagonal)
	}
	if ratio := row / diagonal; !(ratio > 0.5 && ratio < 2) {
		t.Errorf("the glow is %v along the row and %v on the diagonal",
			row, diagonal)
	}
}

// The anti-test: a glow too dim to see is not drawn farther. A point of two whites
// spills a little, which is gone by the usual reach of three sigmas, and
// the bloom reaches no farther for it than before.
func TestADimPointBloomsNoFartherThanTheUsualReach(t *testing.T) {
	bloomed, middle := pointOf(2)
	past := bloomed[middle*bloomSide+middle+int(reachSigmas*bloomSigma)+1].G
	if past != 0 {
		t.Errorf("a dim point glows %v past the usual reach", past)
	}
}

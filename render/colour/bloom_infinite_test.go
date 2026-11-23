package colour

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// dim9 is a nine by nine frame of grey 0.1 with star at the middle.
func dim9(star maths.Light) []maths.Light {
	lights := make([]maths.Light, 81)
	for i := range lights {
		lights[i] = maths.Grey(0.1)
	}
	lights[4*9+4] = star
	return lights
}

// isNaN is whether any part of a light is no number.
func isNaN(l maths.Light) bool {
	return math.IsNaN(l.R) || math.IsNaN(l.G) || math.IsNaN(l.B)
}

// One infinite pixel at the middle of nine by nine, bloomed at a radius
// of one, reaches three pixels each way, so a seven by seven block holds
// the glow. The glow of an infinite light is infinite and the shoulder
// bleaches it to white. Nothing is no number, and what is past the reach
// is as it was.
func TestABloomOfOneInfinitePixelHasNoNaN(t *testing.T) {
	endless := maths.Grey(math.Inf(1))
	out := bloomed(dim9(endless), 9, 9, 1, 1)
	nan := 0
	for y := 0; y < 9; y++ {
		for x := 0; x < 9; x++ {
			l := out[y*9+x]
			if isNaN(l) {
				nan++
				continue
			}
			inside := x >= 1 && x <= 7 && y >= 1 && y <= 7
			if inside && !(l.R == math.Inf(1) && l.G == math.Inf(1) &&
				l.B == math.Inf(1)) {
				t.Errorf("pixel %d,%d within reach is %v, want infinite", x, y, l)
			}
			if !inside && !(l == maths.Grey(0.1)) {
				t.Errorf("pixel %d,%d past reach is %v, want 0.1 grey", x, y, l)
			}
		}
	}
	if nan != 0 {
		t.Errorf("%d of 81 pixels are NaN after bloom of one Inf pixel", nan)
	}
}

// An infinite red part holds no light in green or blue. The glow gives
// the neighbours infinite red and leaves their other parts as they were.
func TestABloomOfAnInfiniteColourKeepsItsHue(t *testing.T) {
	endlessRed := maths.RGB(math.Inf(1), 0, 0)
	out := bloomed(dim9(endlessRed), 9, 9, 1, 1)
	for i, l := range out {
		if isNaN(l) {
			t.Fatalf("pixel %d is %v", i, l)
		}
	}
	if l := out[4*9+5]; !(math.IsInf(l.R, 1) && l.G == 0.1 && l.B == 0.1) {
		t.Errorf("beside the star: %v, want infinite red and 0.1 of the rest", l)
	}
}

// A radius so small that its weights underflow to zero leaves nothing to
// carry the glow, and infinity times zero is not met.
func TestABloomWithZeroWeightsHasNoNaN(t *testing.T) {
	endless := maths.Grey(math.Inf(1))
	out := bloomed(dim9(endless), 9, 9, 1, 1e-300)
	for i, l := range out {
		if isNaN(l) {
			t.Fatalf("pixel %d is %v", i, l)
		}
	}
	if !math.IsInf(out[4*9+4].R, 1) {
		t.Errorf("the star is %v, want infinite", out[4*9+4])
	}
}

// An exposure so small that its reciprocal is infinite does not make a
// zero glow into NaN.
func TestABloomAtAnExposureWithAnInfiniteReciprocalHasNoNaN(t *testing.T) {
	endless := maths.Grey(math.Inf(1))
	out := bloomed(dim9(endless), 9, 9, 1e-310, 1)
	for i, l := range out {
		if isNaN(l) {
			t.Fatalf("pixel %d is %v", i, l)
		}
	}
}

// A pixel that is no number holds no light beyond white that can be
// told, so it spills nothing: its neighbours are as they were. It stays
// no number itself, which Encode draws as black.
func TestANaNPixelDoesNotSpreadInABloom(t *testing.T) {
	notANumber := maths.Grey(math.NaN())
	lights := dim9(notANumber)
	lights[1*9+1] = maths.Grey(5) // something real to bloom
	out := bloomed(lights, 9, 9, 1, 1)
	for i, l := range out {
		if i != 4*9+4 && isNaN(l) {
			t.Errorf("pixel %d is %v: the NaN spread", i, l)
		}
	}
	if l := out[4*9+5]; !(l == maths.Grey(0.1)) {
		t.Errorf("beside the NaN pixel: %v, want 0.1 grey", l)
	}
	if !isNaN(out[4*9+4]) || Encode(out[4*9+4].R) != 0 {
		t.Errorf("the NaN pixel is %v, want it kept and drawn black", out[4*9+4])
	}
}

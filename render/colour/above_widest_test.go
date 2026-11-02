package colour

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// this file sorts before the others on purpose. go test runs files in
// name order and the mutants run stops at the first failure.
//
// so a change that lets a bloom grow without bound or a loop run on is
// noticed here, in a picture a pixel across, before a later test can
// be asked to wait on it.

// refusal is what a call panics with, as a string, or "" when it does
// not panic.
func refusal(call func()) (said string) {
	defer func() {
		said, _ = recover().(string)
	}()
	call()
	return ""
}

// a bloom exactly as wide as widest is allowed, and one past it is
// refused by name. the radii are small enough that a guard turned off
// costs a few megabytes of weights, not gigabytes.
func TestTheWidestBloomIsAllowedAndOnePastItIsNot(t *testing.T) {
	star := []maths.Light{maths.Grey(2)}
	var got []maths.Light
	if said := refusal(func() { got = bloomed(star, 1, 1, 1, widest) }); said != "" {
		t.Fatalf("a bloom of %v pixels was refused: %q", widest, said)
	}
	// a star one pixel across keeps its own light, and gains the sliver
	// of its spill that a kernel 600000 pixels across leaves at its
	// middle: about 1/3 * (1/250663)^2, which is 5e-12
	if l := got[0].R; !(l > 2 && l < 2.0001) {
		t.Errorf("a star under the widest bloom is %v", got[0])
	}
	said := refusal(func() { bloomed(star, 1, 1, 1, 150000) })
	if !strings.Contains(said, "bloom") {
		t.Errorf("a bloom of 150000 pixels: %q", said)
	}
}

// a picture with nothing beyond white is returned as it was, before the
// radius is looked at: white itself is not beyond white, and an
// endless radius is no business of a picture that does not bloom.
func TestAPictureWithNothingBeyondWhiteIsLeftWhateverTheRadius(t *testing.T) {
	lights := []maths.Light{maths.Grey(1), maths.Grey(0.5), maths.Grey(0)}
	var got []maths.Light
	if said := refusal(func() { got = bloomed(lights, 3, 1, 1, 1e9) }); said != "" {
		t.Fatalf("a picture with nothing past white was refused: %q", said)
	}
	for i := range lights {
		if got[i] != lights[i] {
			t.Errorf("light %d is %v, was %v", i, got[i], lights[i])
		}
	}
}

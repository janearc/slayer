package main

import (
	"math"
	"testing"
)

// whites is a pixel's luminance in whites of the print, gain whites to a
// unit of light.
func whites(p *picture, pixel int, gain float64) float64 {
	at := pixel * 3
	return (0.2126*p.light[at] + 0.7152*p.light[at+1] +
		0.0722*p.light[at+2]) * gain
}

// The swell pass's flame core, sampled from its first frame, reaches the
// print at about two hundred whites at the moon's 23 stops; its edge is
// taken as ten thousand times fainter. Through a ceiling of 0.6 whites
// the core lands just under the ceiling, its red, green and blue in the
// proportions it had, and the edge keeps nearly all its light.
func TestTheFireShoulderKeepsTheFlamesColourUnderItsCeiling(t *testing.T) {
	const top = 0.6
	gain := math.Exp2(23)
	core := [3]float64{7.371519e-05, 1.960431e-05, 3.828279e-07}
	edge := [3]float64{core[0] * 1e-4, core[1] * 1e-4, core[2] * 1e-4}
	fire := &picture{2, 1, []float64{core[0], core[1], core[2],
		edge[0], edge[1], edge[2]}}
	before := whites(fire, 1, gain)
	shoulder(fire, top, gain)
	if got := whites(fire, 0, gain); !(got >= 0.99*top && got <= top) {
		t.Errorf("the core is %v whites through a ceiling of %v", got, top)
	}
	for c, want := range []float64{core[1] / core[0], core[2] / core[0]} {
		got := fire.light[1+c] / fire.light[0]
		if !(math.Abs(got-want) <= 1e-12*want) {
			t.Errorf("the core's channel %d over its red is %v, was %v",
				1+c, got, want)
		}
	}
	if after := whites(fire, 1, gain); !(after <= before &&
		after >= 0.97*before) {
		t.Errorf("the edge is %v whites, was %v", after, before)
	}
}

// A pixel of endless light has no colour to keep: it is the ceiling, in
// white, and not the not-a-number that endless over endless would make.
func TestTheFireShoulderPrintsEndlessLightAsTheCeilingInWhite(t *testing.T) {
	const top, gain = 0.6, 1e6
	fire := &picture{1, 1, []float64{math.Inf(1), 1e-6, 0}}
	shoulder(fire, top, gain)
	for c, got := range fire.light {
		if !(math.Abs(got*gain-top) <= 1e-12) {
			t.Errorf("channel %d is %v whites, not the ceiling %v", c,
				got*gain, top)
		}
	}
}

// Dark, negative and not-a-number pixels are left as they were: the
// shoulder only rolls off light.
func TestTheFireShoulderLeavesDarkPixelsAlone(t *testing.T) {
	fire := &picture{3, 1, []float64{0, 0, 0, -1, 0, 0, math.NaN(), 0, 0}}
	shoulder(fire, 0.6, 1e6)
	if fire.light[0] != 0 || fire.light[3] != -1 ||
		!math.IsNaN(fire.light[6]) {
		t.Errorf("the dark pixels moved: %v", fire.light)
	}
}

package maths_test

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// light adds, scales and multiplies part by part.
func TestLightsArithmetic(test *testing.T) {
	first, tint := light(test, "a"), light(test, "tint")
	grey := maths.Grey(picked(test, "half"))
	if got, want := maths.RGB(unity, unity, unity), maths.Grey(unity); got != want {
		test.Errorf("rgb of three ones is %v, not grey %v", got, want)
	}
	if got, want := first.Add(grey), light(test, "a plus half grey"); got != want {
		test.Errorf("add: %v, want %v", got, want)
	}
	if got, want := first.Scale(picked(test, "doubling")), light(test, "a doubled"); got != want {
		test.Errorf("scale: %v, want %v", got, want)
	}
	if got, want := first.Times(tint), light(test, "a through the tint"); got != want {
		test.Errorf("times: %v, want %v", got, want)
	}
}

// grey is the same amount of all three.
func TestGreyIsTheSameOfAllThree(test *testing.T) {
	half := picked(test, "half")
	if got := maths.Grey(half); got.R != half || got.G != half || got.B != half {
		test.Errorf("grey of a half is %v", got)
	}
}

// The luminance is the eye's weighting. Each part has its weight from
// Rec. 709.
func TestLuminanceIsTheEyesWeighting(test *testing.T) {
	for part, want := range map[maths.Light]float64{
		maths.RGB(unity, nought, nought): picked(test, "weight-red"),
		maths.RGB(nought, unity, nought): picked(test, "weight-green"),
		maths.RGB(nought, nought, unity): picked(test, "weight-blue"),
	} {
		if got := part.Luminance(); got != want {
			test.Errorf("%v looks %v, want %v", part, got, want)
		}
	}
}

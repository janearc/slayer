package beamstep_test

import "testing"

// The three budgets show in what a march does, and each case is worked
// from a beam a quarter of a metre in radius, ten metres long, with no
// divergence, and a ray along east at five metres up. Each is run with
// the switch unset and set, which agree on a ray that goes through the
// beam's cylinder and its box alike.

// The refinement envelope is three radii: the ray at 0.74 m beside the
// axis, inside 3 * 0.25 = 0.75 m, is refined, and the ray at 0.76 m is
// not. A cell a metre long takes a step of a quarter radius, 0.0625 m, so
// 16 splits.
//
// Anti: the ray at 0.76 m, past the envelope, is one step.
func TestTheEnvelopeIsThreeRadii(test *testing.T) {
	for _, value := range []string{"", "1"} {
		asked(test, value)
		beams := beamsAt(test, 0)
		below := across(5, 0.74)
		above := across(5, 0.76)
		expect(test, beams.Ray(below, farEnough), 1.5, 2.5, 16)
		expect(test, beams.Ray(above, farEnough), 1.5, 2.5, 1)
	}
}

// Four samples to a radius: a cell a metre long, inside the beam's
// stretch, from 1.25 to 2.25 on a ray through its axis, is cut at
// 0.25 / 4 = 0.0625 m into 16.
//
// Anti: a cell a metre long that the stretch does not reach, from 10 to
// 11, is one step, not 16.
func TestTheSamplesToARadiusAreFour(test *testing.T) {
	for _, value := range []string{"", "1"} {
		asked(test, value)
		refine := beamsAt(test, 0).Ray(across(5, 0), farEnough)
		expect(test, refine, 1.25, 2.25, 16)
		expect(test, refine, 10, 11, 1)
	}
}

// At most 64 splits to a cell: a cell 20 m long over the stretch would
// take 20 / 0.0625 = 320, and takes 64.
//
// Anti: a cell of 20 m that the stretch does not reach, from 20 to 40 on
// a ray that runs on, is one step.
func TestTheMostSplitsToACellAreSixtyFour(test *testing.T) {
	for _, value := range []string{"", "1"} {
		asked(test, value)
		refine := beamsAt(test, 0).Ray(across(5, 0), farEnough)
		expect(test, refine, 0, 20, 64)
		expect(test, refine, 20, 40, 1)
	}
}

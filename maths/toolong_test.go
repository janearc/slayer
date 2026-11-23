package maths_test

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// numbersOf are the nine numbers of an orientation, three to an arrow.
func numbersOf(turn maths.Orientation) []float64 {
	var found []float64
	for _, arrow := range []maths.Vec{turn.X, turn.Y, turn.Z} {
		found = append(found, arrow.X, arrow.Y, arrow.Z)
	}
	return found
}

// turnsFor are the orientations Rotation, Upright and LookAtUp give for a
// vector put in each of the places they take one: the axis, the up, the up of
// a look, and either end of a look.
func turnsFor(test reporter, put maths.Vec) map[string]maths.Orientation {
	test.Helper()
	turn, origin := picked(test, "turn-radians"), vector(test, "origin")
	eye, target := vector(test, "eye"), vector(test, "target")
	return map[string]maths.Orientation{
		"a turn about it":               maths.Rotation(put, turn),
		"standing on it":                maths.Upright(put),
		"an up of it":                   maths.LookAtUp(eye, target, put),
		"looking from it to the origin": maths.LookAtUp(put, origin, maths.Up),
		"looking at it from the origin": maths.LookAtUp(origin, put, maths.Up),
	}
}

// A vector too long for Len to measure, a part past about 1e154, gives an
// orientation of no numbers from each of Rotation, Upright and LookAtUp, as
// View refuses it. It does not give a turn that is not orthonormal, nor the
// zero orientation that reads as no turn.
func TestTooLongToMeasureIsNoNumbers(test *testing.T) {
	for name, got := range turnsFor(test, vector(test, "long")) {
		for place, found := range numbersOf(got) {
			if !math.IsNaN(found) {
				test.Errorf("%s: number %d of %v is %v, not no number", name, place,
					got, found)
			}
		}
	}
}

// anti-test: the same five calls with a vector Len can measure give a turn of
// numbers, none of them no number, so the refusal above is not passed by
// calls that give no numbers for everything.
func TestAVectorLenCanMeasureIsATurn(test *testing.T) {
	for name, got := range turnsFor(test, vector(test, "a")) {
		for place, found := range numbersOf(got) {
			if math.IsNaN(found) || math.IsInf(found, infinity) ||
				math.IsInf(found, -infinity) {
				test.Errorf("%s: number %d of %v is %v", name, place, got, found)
			}
		}
	}
}

// placing is the three vectors a view is made of.
type placing struct{ from, target, up maths.Vec }

// View refuses a position, a target and an up that are too long for Len to
// measure, and says what is wrong with them: not that they are no number.
func TestViewSaysATooLongVectorIsTooLong(test *testing.T) {
	long, eye, target := vector(test, "long"), vector(test, "eye"),
		vector(test, "target")
	for name, put := range map[string]placing{
		"a position": {long, target, maths.Up},
		"a target":   {eye, long, maths.Up},
		"an up":      {eye, target, long},
	} {
		view, err := maths.View(put.from, put.target, put.up)
		if err == nil || view != nil {
			test.Fatalf("accepted %s that is too long", name)
		}
		if !strings.Contains(err.Error(), "too long") {
			test.Errorf("%s: the error %q does not say it is too long", name, err)
		}
	}
}

// anti-test: View of a position, target and up Len can measure is a view,
// so the refusal above is not passed by a View that refuses everything.
func TestViewAcceptsWhatLenCanMeasure(test *testing.T) {
	eye, target := vector(test, "eye"), vector(test, "target")
	if view, err := maths.View(eye, target, maths.Up); err != nil || view == nil {
		test.Fatalf("refused a position, target and up Len can measure: %v", err)
	}
}

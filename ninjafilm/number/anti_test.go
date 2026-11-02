package number_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/number"
	"pgregory.net/rapid"
)

// anti-tests: zero and less are never positive, whatever their size.
func TestNoughtAndLessAreNeverPositive(t *testing.T) {
	rapid.Check(t, noughtAndLessAreNotPositive)
}

func noughtAndLessAreNotPositive(t *rapid.T) {
	amount := rapid.Float64Range(-math.MaxFloat64, none).Draw(t, "amount")
	if number.Positive(amount) {
		t.Fatalf("%v was positive", amount)
	}
}

// anti-tests: what is lost is never positive. That is true of what is
// endless and of what is no number at all.
func TestWhatIsLostIsNeverPositive(t *testing.T) {
	rapid.Check(t, lostIsNotPositive)
}

func lostIsNotPositive(t *rapid.T) {
	odd := rapid.SampledFrom([]float64{math.Inf(upward), math.Inf(downward),
		math.NaN()}).Draw(t, "lost")
	if number.Positive(odd) {
		t.Fatalf("%v was positive", odd)
	}
}

// anti-tests: a positive number added to its own negation is not
// positive. Nor is it left a positive when it is only negated.
func TestAPositiveNumberAndItsNegationAreNotPositive(t *testing.T) {
	rapid.Check(t, positiveAndNegationAreNot)
}

func positiveAndNegationAreNot(t *rapid.T) {
	amount := rapid.Float64Range(math.SmallestNonzeroFloat64,
		math.MaxFloat64).Draw(t, "amount")
	if number.Positive(amount + -amount) {
		t.Fatalf("%v and its negation came to a positive", amount)
	}
}

// anti-tests: sums are not made of fewer lamps. Dark lamps add nothing.
// Lit ones add exactly themselves.
func TestASumIsNotMadeOfFewerLamps(t *testing.T) {
	darkBlue := number.Sum(maths.RGB(unity, pair, none))
	if darkBlue != float64(unity+pair) {
		t.Errorf("a dark lamp changed the sum to %v", darkBlue)
	}
	if lit := number.Sum(maths.RGB(unity, pair, quad)); lit == darkBlue {
		t.Errorf("a lit lamp left the sum at %v", lit)
	}
}

// anti-tests: no lamp is counted twice. Lighting a lamp moves the sum by
// that lamp and by no more.
func TestNoLampIsCountedTwice(t *testing.T) {
	rapid.Check(t, noLampIsCountedTwice)
}

func noLampIsCountedTwice(t *rapid.T) {
	lamp := rapid.Float64Range(-lampLimit, lampLimit)
	red, green := lamp.Draw(t, "red"), lamp.Draw(t, "green")
	blue := lamp.Draw(t, "blue")
	without := number.Sum(maths.RGB(red, green, none))
	with := number.Sum(maths.RGB(red, green, blue))
	if math.Abs(with-without-blue) > toleranceLoose*lampLimit {
		t.Fatalf("lighting %v moved the sum from %v to %v", blue, without,
			with)
	}
}

// anti-tests: one lost lamp spoils the light. It does so whichever lamp
// it is and whatever the other two are.
func TestOneLostLampSpoilsTheLight(t *testing.T) {
	rapid.Check(t, oneLostLampSpoilsTheLight)
}

func oneLostLampSpoilsTheLight(t *rapid.T) {
	odd := rapid.SampledFrom([]float64{math.Inf(upward), math.Inf(downward),
		math.NaN()}).Draw(t, "lost")
	lamp := rapid.Float64Range(-bounded, bounded)
	other, another := lamp.Draw(t, "other"), lamp.Draw(t, "another")
	for _, light := range []maths.Light{maths.RGB(odd, other, another),
		maths.RGB(other, odd, another), maths.RGB(other, another, odd)} {
		if number.Finite(light) {
			t.Fatalf("%v was finite", light)
		}
	}
}

// anti-tests: lamps that fit together are not refused. Three lamps each
// a quarter of the largest number a float holds are finite together.
func TestLampsThatFitTogetherAreNotRefused(t *testing.T) {
	fits := maths.Grey(math.MaxFloat64 / quad)
	if !number.Finite(fits) {
		t.Errorf("%v was not finite", fits)
	}
}

// anti-tests: a light of lamps that are all numbers is never refused. The
// lamps drawn are bounded, so the three cannot pass the largest number.
func TestALightOfNumbersIsNeverRefused(t *testing.T) {
	rapid.Check(t, aLightOfNumbersIsFinite)
}

func aLightOfNumbersIsFinite(t *rapid.T) {
	lamp := rapid.Float64Range(-bounded, bounded)
	red, green, blue := lamp.Draw(t, "red"), lamp.Draw(t, "green"),
		lamp.Draw(t, "blue")
	if !number.Finite(maths.RGB(red, green, blue)) {
		t.Fatalf("%v, %v and %v were refused", red, green, blue)
	}
}

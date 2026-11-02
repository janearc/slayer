package number_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/number"
	"pgregory.net/rapid"
)

// what these tests count in, each defined
const (
	none     = 0  // defined: no amount at all
	unity    = 1  // defined: one
	upward   = 1  // defined: math.Inf's code for infinity above all numbers
	downward = -1 // defined: math.Inf's code for infinity below all numbers
	either   = 0  // defined: math.IsInf's code for infinity of either sign
	lamps    = 3  // defined: a light has three lamps
	pair     = 2  // defined: a count of two
	quad     = 4  // defined: a count of four
)

// what these tests choose, each picked in picked-numbers.yml
var (
	lampLimit      = data.Picked("number-lamp-limit")
	bounded        = data.Picked("number-bounded")
	toleranceLoose = data.Picked("tolerance-loose")
)

// lost is whether a number is endless or not a number at all.
func lost(amount float64) bool {
	return math.IsInf(amount, either) || math.IsNaN(amount)
}

func TestPositiveIsAboveNoughtAndFinite(t *testing.T) {
	yes := []float64{math.SmallestNonzeroFloat64, math.MaxFloat64}
	for _, amount := range yes {
		if !number.Positive(amount) {
			t.Errorf("%v was not positive", amount)
		}
	}
	no := []float64{none, math.Copysign(none, downward),
		-math.SmallestNonzeroFloat64, -math.MaxFloat64,
		math.Inf(upward), math.Inf(downward), math.NaN()}
	for _, amount := range no {
		if number.Positive(amount) {
			t.Errorf("%v was positive", amount)
		}
	}
}

// Any float at all is positive when it is greater than zero and neither endless
// nor lost. The other way of saying that is the check.
func TestPositiveIsWhatItIsToBeAboveNoughtAndFinite(t *testing.T) {
	rapid.Check(t, positiveIsPlain)
}

func positiveIsPlain(t *rapid.T) {
	amount := rapid.Float64().Draw(t, "amount")
	want := amount > none && !lost(amount)
	if got := number.Positive(amount); got != want {
		t.Fatalf("%v: positive is %v, want %v", amount, got, want)
	}
}

// Positive numbers stay positive when they are added, unless the total
// is too large to hold. Negating a positive number never leaves one.
func TestPositiveHoldsUnderAddingAndFailsUnderNegating(t *testing.T) {
	rapid.Check(t, positiveSurvivesAdding)
}

func positiveSurvivesAdding(t *rapid.T) {
	smallest, largest := math.SmallestNonzeroFloat64, math.MaxFloat64
	one := rapid.Float64Range(smallest, largest).Draw(t, "one")
	other := rapid.Float64Range(smallest, largest).Draw(t, "other")
	total := one + other
	if !math.IsInf(total, either) && !number.Positive(total) {
		t.Fatalf("%v and %v are positive and %v is not", one, other, total)
	}
	if number.Positive(-one) {
		t.Fatalf("%v is positive", -one)
	}
}

func TestSumAddsTheThreeLamps(t *testing.T) {
	if got, want := number.Sum(maths.RGB(unity, pair, quad)),
		float64(unity+pair+quad); got != want {
		t.Errorf("one, two and four came to %v, want %v", got, want)
	}
	var dark maths.Light
	if got := number.Sum(dark); got != none {
		t.Errorf("no light at all came to %v", got)
	}
}

// Each lamp is added once, whichever lamp it is. Grey comes to three
// times one lamp, being the same in every lamp.
func TestSumIsEveryLampAddedOnce(t *testing.T) {
	rapid.Check(t, sumAddsEachLampOnce)
}

func sumAddsEachLampOnce(t *rapid.T) {
	lamp := rapid.Float64Range(-lampLimit, lampLimit)
	red, green := lamp.Draw(t, "red"), lamp.Draw(t, "green")
	blue := lamp.Draw(t, "blue")
	light := maths.RGB(red, green, blue)
	if got, want := number.Sum(light), red+green+blue; got != want {
		t.Fatalf("%v, %v and %v came to %v, want %v", red, green, blue,
			got, want)
	}
	grey := number.Sum(maths.Grey(red))
	if math.Abs(grey-lamps*red) > toleranceLoose*math.Abs(red) {
		t.Fatalf("grey %v came to %v, want %v", red, grey, lamps*red)
	}
	for _, alone := range []float64{red, green, blue} {
		if got := number.Sum(lampAmong(alone, red, green, blue)); got !=
			alone {
			t.Fatalf("one lamp alone, %v, came to %v", alone, got)
		}
	}
}

// lampAmong is a light with just the lamp that is the value of the
// first of its lamps that equals it, lit and the others dark.
func lampAmong(alone, red, green, blue float64) maths.Light {
	switch alone {
	case red:
		return maths.RGB(red, none, none)
	case green:
		return maths.RGB(none, green, none)
	}
	return maths.RGB(none, none, blue)
}

func TestFiniteIsEveryLampANumber(t *testing.T) {
	var dark maths.Light
	yes := []maths.Light{dark, maths.RGB(-unity, -pair, -lamps),
		maths.RGB(math.MaxFloat64, none, none)}
	for _, light := range yes {
		if !number.Finite(light) {
			t.Errorf("%v was not finite", light)
		}
	}
	for _, bad := range []float64{math.Inf(upward), math.Inf(downward),
		math.NaN()} {
		no := []maths.Light{maths.RGB(bad, none, none),
			maths.RGB(none, bad, none), maths.RGB(none, none, bad)}
		for _, light := range no {
			if number.Finite(light) {
				t.Errorf("%v was finite", light)
			}
		}
	}
}

// Three lamps each as large as a float goes are together too large to
// hold, and Finite says so. That is how it sums. It is stated here so
// that nobody takes it for a light that is only very bright.
func TestFiniteCountsTheLampsTogether(t *testing.T) {
	huge := maths.Grey(math.MaxFloat64)
	if number.Finite(huge) {
		t.Errorf("%v was finite", huge)
	}
}

// Light is finite when no lamp is endless or lost. The lamps drawn here
// are bounded so that the three together cannot pass the largest
// number, which Finite also asks.
func TestFiniteFollowsEachLamp(t *testing.T) {
	rapid.Check(t, finiteFollowsLamps)
}

func finiteFollowsLamps(t *rapid.T) {
	odd := rapid.SampledFrom([]float64{math.Inf(upward),
		math.Inf(downward), math.NaN()})
	lamp := rapid.OneOf(rapid.Float64Range(-bounded, bounded), odd)
	red, green, blue := lamp.Draw(t, "red"), lamp.Draw(t, "green"),
		lamp.Draw(t, "blue")
	want := !lost(red) && !lost(green) && !lost(blue)
	if got := number.Finite(maths.RGB(red, green, blue)); got != want {
		t.Fatalf("%v, %v and %v: finite is %v, want %v", red, green, blue,
			got, want)
	}
}

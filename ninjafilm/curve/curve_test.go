package curve_test

import (
	"math"
	"os"
	"testing"

	"github.com/janearc/slayer/ninjafilm/curve"
	"github.com/janearc/slayer/ninjafilm/data"
	"pgregory.net/rapid"
)

// what these tests count in, each defined
const (
	none     = 0   // defined: no amount at all
	pair     = 2   // defined: a count of two
	tenfold  = 10  // defined: ten times, the base of the logarithm of exposure
	halfway  = 0.5 // defined: half of the way between two ends
	upward   = 1   // defined: math.Inf's code for infinity above all numbers
	downward = -1  // defined: math.Inf's code for infinity below all numbers
	either   = 0   // defined: math.IsInf's code for infinity of either sign
	unity    = 1   // defined: the exposure at the middle of a curve
)

// the table of curves that are refused
const refusedFile = "testdata/refused.csv"

// what these tests choose, each picked in picked-numbers.yml
var (
	floorMost      = data.Picked("curve-floor-most")
	roomLeast      = data.Picked("curve-room-least")
	roomMost       = data.Picked("curve-room-most")
	slopeLeast     = data.Picked("curve-slope-least")
	slopeMost      = data.Picked("curve-slope-most")
	decades        = data.Picked("curve-decades")
	stopsMost      = data.Picked("curve-stops-most")
	oddLeast       = data.Picked("curve-odd-least")
	oddMost        = data.Picked("curve-odd-most")
	belowMost      = data.Picked("curve-below-most")
	step           = data.Picked("curve-step")
	slopeError     = data.Picked("curve-slope-error")
	exampleLeast   = data.Picked("curve-example-least")
	exampleMost    = data.Picked("curve-example-most")
	exampleSlope   = data.Picked("curve-example-slope")
	toleranceTight = data.Picked("tolerance-tight")
	toleranceLoose = data.Picked("tolerance-loose")
)

// within is whether two numbers are within a tolerance of each other.
func within(one, other, tolerance float64) bool {
	return math.Abs(one-other) <= tolerance
}

// shape is the three numbers of a curve a film could have: how dense it
// is at least and at most, and how steep it is in the middle.
func shape(t *rapid.T) (least, most, slope float64) {
	least = rapid.Float64Range(none, floorMost).Draw(t, "least")
	most = least + rapid.Float64Range(roomLeast, roomMost).Draw(t, "room")
	slope = rapid.Float64Range(slopeLeast, slopeMost).Draw(t, "slope")
	return least, most, slope
}

// drawn is a curve a film could have, and its numbers.
func drawn(t *rapid.T) (tone curve.Curve, least, most, slope float64) {
	least, most, slope = shape(t)
	tone, err := curve.New(least, most, slope)
	if err != nil {
		t.Fatalf("a curve of %v, %v and %v was refused: %v", least, most,
			slope, err)
	}
	return tone, least, most, slope
}

// exposure is a light a film could meet, from a millionth of the middle
// to a million times it.
func exposure(t *rapid.T, label string) float64 {
	return math.Pow(tenfold, rapid.Float64Range(-decades, decades).Draw(t,
		label))
}

// example is the curve of a slide, or the test stops.
func example(t *testing.T) curve.Curve {
	t.Helper()
	tone, err := curve.New(exampleLeast, exampleMost, exampleSlope)
	if err != nil {
		t.Fatal(err)
	}
	return tone
}

func TestACurveHasRoomAndASlope(t *testing.T) {
	tone := example(t)
	if got := tone.Density(none); got != exampleMost {
		t.Errorf("the curve is %v where no light fell", got)
	}
	middle := (exampleLeast + exampleMost) * halfway
	if got := tone.Density(unity); !within(got, middle, toleranceTight) {
		t.Errorf("the curve is %v in the middle, want %v", got, middle)
	}
}

// anti-tests: a curve with no slope or no room is not made. Nor is one with a
// floor less than zero. What comes back is the zero curve.
func TestACurveRefusesWhatItCannotBe(t *testing.T) {
	text, err := os.ReadFile(refusedFile)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := data.Parse(string(text))
	if err != nil {
		t.Fatal(err)
	}
	var zero curve.Curve
	for _, row := range rows {
		label, _ := row.Text("label")
		least, _ := row.Float("least")
		most, _ := row.Float("most")
		slope, _ := row.Float("slope")
		got, err := curve.New(least, most, slope)
		if err == nil || got != zero {
			t.Errorf("%s was not refused: %v, %v", label, got, err)
		}
	}
}

// oddNumbers draws ordinary numbers and the odd ones: the lost, the endless and
// zero.
func oddNumbers() *rapid.Generator[float64] {
	odd := rapid.SampledFrom([]float64{math.NaN(), math.Inf(upward),
		math.Inf(downward), none})
	return rapid.OneOf(rapid.Float64Range(oddLeast, oddMost), odd)
}

// sound is whether a number is neither endless nor lost.
func sound(amount float64) bool {
	return !math.IsNaN(amount) && !math.IsInf(amount, either)
}

// Whether a curve is made follows from its numbers. Each is a number. The floor
// is zero or more. The most is above the least. The slope is greater than zero.
// The numbers drawn here are often the odd ones.
func TestACurveIsMadeExactlyWhenItCanBe(t *testing.T) {
	rapid.Check(t, curveIsMadeWhenItCanBe)
}

func curveIsMadeWhenItCanBe(t *rapid.T) {
	odd := oddNumbers()
	least, most, slope := odd.Draw(t, "least"), odd.Draw(t, "most"),
		odd.Draw(t, "slope")
	want := sound(least) && sound(most) && sound(slope) &&
		least >= none && most > least && slope > none
	if _, err := curve.New(least, most, slope); (err == nil) != want {
		t.Fatalf("a curve of %v, %v and %v: made is %v, want %v", least,
			most, slope, err == nil, want)
	}
}

func TestACurveFallsFromMostToLeastThroughItsMiddle(t *testing.T) {
	tone := example(t)
	for _, light := range []float64{none, -unity, math.Inf(upward)} {
		want := exampleMost
		if light > none {
			want = exampleLeast
		}
		if got := tone.Density(light); !within(got, want, toleranceTight) {
			t.Errorf("an exposure of %v is a density of %v, want %v",
				light, got, want)
		}
	}
	last := tone.Density(none)
	for stops := -stopsMost; stops <= stopsMost; stops++ {
		here := tone.Density(math.Exp2(stops))
		if here >= last {
			t.Fatalf("at %v stops the density is %v, after %v: it "+
				"did not fall", stops, here, last)
		}
		last = here
	}
}

// More light never makes the positive denser. Twice the light makes it
// less dense unless the curve has run out. The density stays between
// the curve's ends.
func TestMoreLightIsLessDensity(t *testing.T) {
	rapid.Check(t, moreLightIsLessDense)
}

func moreLightIsLessDense(t *rapid.T) {
	tone, least, most, _ := drawn(t)
	dim, bright := exposure(t, "dim"), exposure(t, "bright")
	dim, bright = math.Min(dim, bright), math.Max(dim, bright)
	fewer, more := tone.Density(bright), tone.Density(dim)
	if fewer > more {
		t.Fatalf("an exposure of %v is denser (%v) than one of %v (%v)",
			bright, fewer, dim, more)
	}
	open := more < most-toleranceLoose && fewer > least+toleranceLoose
	if bright >= pair*dim && open && !(fewer < more) {
		t.Fatalf("twice the light, %v against %v, left the density at %v",
			bright, dim, more)
	}
	for _, each := range []float64{fewer, more} {
		if each < least-toleranceTight || each > most+toleranceTight {
			t.Fatalf("a density of %v is outside %v to %v", each, least,
				most)
		}
	}
}

// No light at all is the most density, or less than none: exposures less than
// zero are none. Light without end is the least.
func TestNoLightIsTheMostDensity(t *testing.T) {
	rapid.Check(t, noLightIsMostDense)
}

func noLightIsMostDense(t *rapid.T) {
	tone, least, most, _ := drawn(t)
	below := -rapid.Float64Range(none, belowMost).Draw(t, "below")
	for _, nothing := range []float64{none, below} {
		if got := tone.Density(nothing); !within(got, most, toleranceTight) {
			t.Fatalf("an exposure of %v is %v, want %v", nothing, got,
				most)
		}
	}
	if got := tone.Density(math.Inf(upward)); !within(got, least,
		toleranceTight) {
		t.Fatalf("endless light is %v, want %v", got, least)
	}
}

// Exposure of one is the middle of the curve, whatever curve it is.
func TestOneIsTheMiddleOfTheCurve(t *testing.T) {
	rapid.Check(t, oneIsTheMiddle)
}

func oneIsTheMiddle(t *rapid.T) {
	tone, least, most, _ := drawn(t)
	want := (least + most) * halfway
	if got := tone.Density(unity); !within(got, want, toleranceTight) {
		t.Fatalf("the middle is %v, want %v", got, want)
	}
}

// The curve is a logistic in the logarithm of exposure, so it is
// symmetric about its middle: as much more light than the middle takes
// off as the same amount less puts on.
func TestTheCurveIsSymmetricAboutItsMiddle(t *testing.T) {
	rapid.Check(t, curveIsSymmetric)
}

func curveIsSymmetric(t *rapid.T) {
	tone, least, most, _ := drawn(t)
	light := exposure(t, "light")
	sum := tone.Density(light) + tone.Density(unity/light)
	if !within(sum, least+most, toleranceLoose) {
		t.Fatalf("exposures %v and its reciprocal sum to %v, want %v",
			light, sum, least+most)
	}
}

// the slope at the middle is the curve's own, in density for each
// tenfold of exposure
func TestACurvesSlopeAtItsMiddleIsItsOwn(t *testing.T) {
	rapid.Check(t, slopeIsItsOwn)
}

func slopeIsItsOwn(t *rapid.T) {
	tone, _, _, slope := drawn(t)
	over := tone.Density(math.Pow(tenfold, step))
	under := tone.Density(math.Pow(tenfold, -step))
	if got := (under - over) / (pair * step); !within(got, slope,
		slopeError) {
		t.Fatalf("a slope of %v measures %v at the middle", slope, got)
	}
}

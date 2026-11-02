package curve_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/ninjafilm/curve"
	"pgregory.net/rapid"
)

// anti-tests: the middle of a curve is neither of its ends. No light is
// the most and endless light the least, and one is neither.
func TestTheMiddleOfACurveIsNeitherEnd(t *testing.T) {
	tone := example(t)
	got := tone.Density(unity)
	if within(got, exampleMost, toleranceTight) || within(got, exampleLeast,
		toleranceTight) {
		t.Errorf("the middle of the curve is %v, an end", got)
	}
}

// anti-tests: a curve is refused whenever one of its numbers is out, and
// it comes back as the zero curve. The numbers are drawn so that exactly
// one of the three is wrong.
func TestACurveWithOneNumberOutIsRefused(t *testing.T) {
	rapid.Check(t, oneNumberOutIsRefused)
}

func oneNumberOutIsRefused(t *rapid.T) {
	least, most, slope := shape(t)
	out := rapid.IntRange(none, pair).Draw(t, "out")
	wrong := rapid.Float64Range(-floorMost, none).Draw(t, "wrong")
	switch out {
	case none:
		least = -roomLeast - math.Abs(wrong)
	case unity:
		most = least - math.Abs(wrong)
	default:
		slope = wrong
	}
	var zero curve.Curve
	if got, err := curve.New(least, most, slope); err == nil || got != zero {
		t.Fatalf("%v, %v and %v made a curve", least, most, slope)
	}
}

// anti-tests: no density is past the ends of its curve, whatever light
// there is. That includes light that is lost.
func TestNoDensityIsPastTheEndsOfItsCurve(t *testing.T) {
	rapid.Check(t, noDensityIsPastTheEnds)
}

func noDensityIsPastTheEnds(t *rapid.T) {
	tone, least, most, _ := drawn(t)
	light := rapid.Float64Range(-oddMost, oddMost*oddMost).Draw(t, "light")
	got := tone.Density(light)
	if got < least-toleranceTight || got > most+toleranceTight ||
		math.IsNaN(got) {
		t.Fatalf("light %v gave a density of %v outside %v to %v", light, got,
			least, most)
	}
}

// anti-tests: the density does not stay put where the curve is open.
// Twice the light that leaves it unchanged is only where the curve has
// run out.
func TestADensityThatStaysPutIsAtAnEnd(t *testing.T) {
	rapid.Check(t, stayingPutIsAtAnEnd)
}

func stayingPutIsAtAnEnd(t *rapid.T) {
	tone, least, most, _ := drawn(t)
	dim := exposure(t, "dim")
	now, doubled := tone.Density(dim), tone.Density(pair*dim)
	if now != doubled {
		return
	}
	atLeast := within(now, least, toleranceLoose)
	if !atLeast && !within(now, most, toleranceLoose) {
		t.Fatalf("the density stayed at %v between %v and %v", now, least,
			most)
	}
}

// anti-tests: some light is never the most density. From the middle up
// the curve has begun to fall, and below it is the least.
func TestSomeLightIsNeverTheMostDensity(t *testing.T) {
	rapid.Check(t, someLightIsNotTheMost)
}

func someLightIsNotTheMost(t *rapid.T) {
	tone, least, most, _ := drawn(t)
	light := math.Pow(tenfold, rapid.Float64Range(none, decades).Draw(t,
		"light"))
	got := tone.Density(light)
	if got >= most || got < least-toleranceTight {
		t.Fatalf("light %v gave %v on a curve of %v to %v", light, got,
			least, most)
	}
}

// anti-tests: no exposure two stops from one is the middle. The middle
// is where the exposure is one and nowhere else.
func TestNoExposureTwoStopsOffIsTheMiddle(t *testing.T) {
	rapid.Check(t, twoStopsOffIsNotTheMiddle)
}

func twoStopsOffIsNotTheMiddle(t *rapid.T) {
	tone, least, most, _ := drawn(t)
	middle := (least + most) * halfway
	up := math.Pow(pair, pair)
	for _, light := range []float64{up, unity / up} {
		if within(tone.Density(light), middle, toleranceLoose) {
			t.Fatalf("exposure %v is the middle", light)
		}
	}
}

// anti-tests: exposures that are not reciprocals do not sum to the two
// ends. The symmetry is about the middle and about no other place.
func TestExposuresThatAreNotReciprocalsDoNotSumToTheEnds(t *testing.T) {
	rapid.Check(t, notReciprocalsDoNotSum)
}

func notReciprocalsDoNotSum(t *rapid.T) {
	tone, least, most, _ := drawn(t)
	light := math.Pow(pair, rapid.Float64Range(-unity, unity).Draw(t,
		"stops"))
	other := pair * unity / light
	sum := tone.Density(light) + tone.Density(other)
	if within(sum, least+most, toleranceLoose) {
		t.Fatalf("%v and %v summed to the two ends", light, other)
	}
}

// anti-tests: the slope far from the middle is not the slope at it. The
// curve flattens toward its ends.
func TestTheSlopeFarFromTheMiddleIsNotTheMiddlesSlope(t *testing.T) {
	rapid.Check(t, slopeFlattensTowardTheEnds)
}

func slopeFlattensTowardTheEnds(t *rapid.T) {
	tone, _, _, slope := drawn(t)
	far := math.Pow(tenfold, decades)
	over := tone.Density(far * math.Pow(tenfold, step))
	under := tone.Density(far * math.Pow(tenfold, -step))
	got := (under - over) / (pair * step)
	if got >= slope {
		t.Fatalf("the slope far out is %v, the middle's is %v", got, slope)
	}
}

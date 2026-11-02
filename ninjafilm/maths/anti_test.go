package maths_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/ninjafilm/maths"
	"pgregory.net/rapid"
)

// anti-tests: powers just less than zero are refused, however small the
// shortfall. What comes back is no power.
func TestAPowerJustBelowNoughtIsRefused(t *testing.T) {
	rapid.Check(t, powerBelowNoughtIsRefused)
}

func powerBelowNoughtIsRefused(t *rapid.T) {
	amount := -rapid.Float64Range(math.SmallestNonzeroFloat64,
		sunsMost).Draw(t, "amount")
	var zero maths.Power
	if made, err := maths.Watts(amount); err == nil || made != zero {
		t.Fatalf("%v watts made a power", amount)
	}
}

// anti-tests: two suns are not one sun. The luminosity is the number of
// suns times the sun's, and not the sun's.
func TestTwoSunsAreNotOneSun(t *testing.T) {
	one, err := maths.SolarLuminosities(unity)
	if err != nil {
		t.Fatal(err)
	}
	two, err := maths.SolarLuminosities(unity + unity)
	if err != nil {
		t.Fatal(err)
	}
	if two.Watts() == one.Watts() || two.Watts() != sun*(unity+unity) {
		t.Errorf("one sun is %v W and two are %v W", one.Watts(), two.Watts())
	}
}

// anti-tests: endless suns and suns that are no number make no power.
func TestEndlessOrLostSunsMakeNoPower(t *testing.T) {
	var zero maths.Power
	for _, count := range []float64{math.Inf(upward), math.Inf(downward),
		math.NaN()} {
		if made, err := maths.SolarLuminosities(count); err == nil ||
			made != zero {
			t.Errorf("%v suns made %v", count, made)
		}
	}
}

// anti-tests: a temperature that is refused gives no light. What comes
// back is the zero temperature, and it is silent.
func TestARefusedTemperatureGivesNoLight(t *testing.T) {
	rapid.Check(t, refusedTemperatureIsSilent)
}

func refusedTemperatureIsSilent(t *rapid.T) {
	amount := -rapid.Float64Range(math.SmallestNonzeroFloat64,
		heatMost).Draw(t, "amount")
	made, err := maths.Kelvin(amount)
	if err == nil {
		t.Fatalf("%v kelvin made a temperature", amount)
	}
	if got := made.Exitance(waveLeast, waveMost); got != none {
		t.Fatalf("the refused temperature gave %v W", got)
	}
}

// anti-tests: an area that is refused is no area. It holds no square
// metres, however it was asked for.
func TestARefusedAreaHoldsNoSquareMetres(t *testing.T) {
	rapid.Check(t, refusedAreaIsEmpty)
}

func refusedAreaIsEmpty(t *rapid.T) {
	amount := -rapid.Float64Range(none, sunsMost).Draw(t, "amount")
	made, err := maths.SquareMetres(amount)
	if err == nil || made.SquareMetres() != none {
		t.Fatalf("%v square metres made %v", amount, made)
	}
}

// anti-tests: a piece inside a stretch gives less than the stretch when
// the rest of it gives anything. The piece is never the whole.
func TestAPieceGivesLessThanTheStretchWhenTheRestGivesAnything(
	t *testing.T) {
	rapid.Check(t, pieceGivesLessThanWhole)
}

func pieceGivesLessThanWhole(t *rapid.T) {
	kelvin := rapid.Float64Range(heatLeast, heatMost).Draw(t, "heat")
	heat := heated(t, kelvin)
	short, long := bandAt(t, kelvin, bandRatio)
	middle := math.Sqrt(short * long)
	rest := heat.Exitance(middle, long)
	if rest > none && !(heat.Exitance(short, middle) < heat.Exitance(short,
		long)) {
		t.Fatalf("%v K: the first half gave as much as the whole", kelvin)
	}
}

// anti-tests: halves that leave a part out fall short of the band. Power
// in the gap between them is not counted.
func TestHalvesThatLeaveAPartOutFallShortOfTheBand(t *testing.T) {
	rapid.Check(t, gappedHalvesFallShort)
}

func gappedHalvesFallShort(t *rapid.T) {
	kelvin := rapid.Float64Range(heatLeast, heatMost).Draw(t, "heat")
	heat := heated(t, kelvin)
	short, long := bandAt(t, kelvin, bandRatio)
	third := math.Cbrt(bandRatio)
	before, after := short*third, short*third*third
	if heat.Exitance(before, after) <= none {
		t.Skip("no gap with power in it")
	}
	gapped := heat.Exitance(short, before) + heat.Exitance(after, long)
	if whole := heat.Exitance(short, long); !(gapped < whole) {
		t.Fatalf("%v to %v nm gives %v W and the halves with a gap %v W",
			short, long, whole, gapped)
	}
}

// anti-tests: a colder body gives less in a band where it gives
// anything. It is never equal to a hotter one there.
func TestAColderBodyGivesLessInABandWhereItGivesAnything(t *testing.T) {
	rapid.Check(t, colderGivesLess)
}

func colderGivesLess(t *rapid.T) {
	kelvin := rapid.Float64Range(heatLeast, heatMost).Draw(t, "heat")
	more := kelvin + rapid.Float64Range(unity, hotterMost).Draw(t, "more")
	short, long := bandAt(t, more, bandRatio)
	cooler, hotter := heated(t, kelvin), heated(t, more)
	got := cooler.Exitance(short, long)
	if got > none && !(got < hotter.Exitance(short, long)) {
		t.Fatalf("%v K gives as much as %v K in %v to %v nm", kelvin, more,
			short, long)
	}
}

// anti-tests: scaling the temperature alone does not scale the power by
// the fourth power. The wavelengths have to be scaled the other way too.
func TestScalingOnlyTheTemperatureIsNotScalingByTheFourthPower(
	t *testing.T) {
	rapid.Check(t, temperatureAloneDoesNotScale)
}

func temperatureAloneDoesNotScale(t *rapid.T) {
	kelvin := rapid.Float64Range(heatLeast, heatMost).Draw(t, "heat")
	factor := pair + rapid.Float64Range(scaleLeast, scaleMost).Draw(t,
		"factor")
	short, long := bandAt(t, kelvin, bandRatio)
	base := heated(t, kelvin).Exitance(short, long)
	alone := heated(t, kelvin*factor).Exitance(short, long)
	if base > none && within(alone, base*math.Pow(factor, fourth),
		additiveError) {
		t.Fatalf("%v K scaled by %v alone scaled the power by its fourth",
			kelvin, factor)
	}
}

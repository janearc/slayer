package maths_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/maths"
	"pgregory.net/rapid"
)

// what these tests count in, each defined
const (
	none     = 0  // defined: no amount at all
	upward   = 1  // defined: math.Inf's code for infinity above all numbers
	downward = -1 // defined: math.Inf's code for infinity below all numbers
	either   = 0  // defined: math.IsInf's code for infinity of either sign
	unity    = 1  // defined: one sun, one square metre, or one place on
)

// what these tests choose, each picked in picked-numbers.yml
var (
	oddLeast = data.Picked("maths-odd-least")
	oddMost  = data.Picked("maths-odd-most")
	sunsMost = data.Picked("maths-suns-most")
	sun      = data.Picked("maths-sun")
)

// odd draws an ordinary amount or an odd one.
func odd() *rapid.Generator[float64] {
	strange := rapid.SampledFrom([]float64{math.NaN(), math.Inf(upward),
		math.Inf(downward), none, -math.SmallestNonzeroFloat64})
	return rapid.OneOf(rapid.Float64Range(oddLeast, oddMost), strange)
}

// sound is whether an amount is neither endless nor lost.
func sound(amount float64) bool {
	return !math.IsNaN(amount) && !math.IsInf(amount, either)
}

// anti-tests: powers less than zero, endless or lost are not made. What comes
// back is no power.
func TestAPowerRefusesWhatAPowerCannotBe(t *testing.T) {
	var zero maths.Power
	for _, amount := range []float64{-unity, math.NaN(), math.Inf(upward),
		math.Inf(downward)} {
		got, err := maths.Watts(amount)
		if err == nil || got != zero {
			t.Errorf("Watts(%v) was taken: %v, %v", amount, got, err)
		}
	}
	if _, err := maths.Watts(none); err != nil {
		t.Errorf("no power at all is a power: %v", err)
	}
}

// Whether a power is made follows from its amount alone, and it is read
// back as it was given.
func TestAPowerIsMadeExactlyWhenItIsNotBelowNought(t *testing.T) {
	rapid.Check(t, powerIsMadeWhenItCan)
}

func powerIsMadeWhenItCan(t *rapid.T) {
	amount := odd().Draw(t, "amount")
	made, err := maths.Watts(amount)
	want := sound(amount) && amount >= none
	if (err == nil) != want {
		t.Fatalf("Watts(%v): made is %v, want %v", amount, err == nil, want)
	}
	if err == nil && made.Watts() != amount {
		t.Fatalf("%v watts came back as %v", amount, made.Watts())
	}
}

func TestOneSunIsTheIAUs(t *testing.T) {
	one, err := maths.SolarLuminosities(unity)
	if err != nil {
		t.Fatal(err)
	}
	if one.Watts() != sun {
		t.Errorf("one sun is %v W, and IAU 2015 B3 says %v", one.Watts(),
			sun)
	}
}

// Suns are the sun's luminosity times their number, and a number of them less
// than zero is not a power.
func TestSunsAreTheSunsLuminosityTimesTheirNumber(t *testing.T) {
	rapid.Check(t, sunsMultiply)
}

func sunsMultiply(t *rapid.T) {
	count := rapid.Float64Range(none, sunsMost).Draw(t, "suns")
	made, err := maths.SolarLuminosities(count)
	if err != nil || made.Watts() != count*sun {
		t.Fatalf("%v suns made %v W, %v: want %v", count, made.Watts(),
			err, count*sun)
	}
	if _, err := maths.SolarLuminosities(-count - unity); err == nil {
		t.Fatalf("%v suns were made", -count-unity)
	}
}

// Whether a temperature is made follows from its amount alone, and it
// is as hot as it was said to be.
func TestATemperatureIsMadeExactlyWhenItIsNotBelowNought(t *testing.T) {
	rapid.Check(t, temperatureIsMadeWhenItCan)
}

func temperatureIsMadeWhenItCan(t *rapid.T) {
	amount := odd().Draw(t, "amount")
	_, err := maths.Kelvin(amount)
	want := sound(amount) && amount >= none
	if (err == nil) != want {
		t.Fatalf("Kelvin(%v): made is %v, want %v", amount, err == nil,
			want)
	}
}

// anti-tests: temperatures less than zero, endless or lost are not made.
func TestATemperatureRefusesWhatATemperatureCannotBe(t *testing.T) {
	var zero maths.Temperature
	for _, amount := range []float64{-math.SmallestNonzeroFloat64,
		math.NaN(), math.Inf(upward)} {
		got, err := maths.Kelvin(amount)
		if err == nil || got != zero {
			t.Errorf("Kelvin(%v) was taken: %v, %v", amount, got, err)
		}
	}
}

// anti-tests: areas of nothing are not made. Nor are areas below
// nothing, endless or lost. What comes back is no area.
func TestAnAreaRefusesWhatAnAreaCannotBe(t *testing.T) {
	var zero maths.Area
	for _, amount := range []float64{none, -unity, math.NaN(),
		math.Inf(upward), math.Inf(downward)} {
		got, err := maths.SquareMetres(amount)
		if err == nil || got != zero {
			t.Errorf("an area of %v square metres was taken: %v, %v",
				amount, got, err)
		}
	}
}

// Whether an area is made follows from its amount alone, and it is as
// large as it was said to be.
func TestAnAreaIsMadeExactlyWhenItIsAboveNought(t *testing.T) {
	rapid.Check(t, areaIsMadeWhenItCan)
}

func areaIsMadeWhenItCan(t *rapid.T) {
	amount := odd().Draw(t, "amount")
	made, err := maths.SquareMetres(amount)
	want := sound(amount) && amount > none
	if (err == nil) != want {
		t.Fatalf("an area of %v: made is %v, want %v", amount, err == nil,
			want)
	}
	if err == nil && made.SquareMetres() != amount {
		t.Fatalf("%v square metres came back as %v", amount,
			made.SquareMetres())
	}
}

// datum is a picked number by its name.
func datum(name string) float64 { return data.Picked(name) }

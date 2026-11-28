package generate

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// The pair of the first slice: a 2.45 solar mass primary 12 million
// years old, and a hot white dwarf given by its numbers, 9.2 AU apart on
// an orbit of eccentricity 0.16. The white dwarf's radius is from the
// Nauenberg 1972 mass and radius relation, from memory, to shelve.
func theFirstPair() Binary {
	return Binary{
		Name:    "first pair",
		Primary: Star{Name: "primary", Mass: measure.SolarMasses(2.45), Age: 1.2e7},
		Companion: Remnant{Name: "dwarf", Mass: measure.SolarMasses(0.62),
			Temperature: measure.Kelvin(25000),
			Radius:      measure.SolarRadii(0.0122)},
		Orbit: Orbit{Distance: measure.Metres(9.2 * AstronomicalUnit),
			Eccentricity: 0.16},
	}
}

// at is a time as a share of the pair's year, in years.
func at(pair Pair, share float64) Years { return Years(share * float64(pair.Year)) }

// Each star's distance from the centre is in proportion to the other's
// mass, and the two together are the separation at that moment: the
// heavier star sits nearer the centre.
func TestEachStarSitsInProportionToTheOthersMass(t *testing.T) {
	binary := theFirstPair()
	pair, err := binary.Place(0)
	if err != nil {
		t.Fatal(err)
	}
	one, other := pair.Primary.At.Len(), pair.Companion.At.Len()
	periapsis := 9.2 * AstronomicalUnit * (1 - 0.16)
	if got := one + other; math.Abs(got-periapsis) > periapsis*1e-9 {
		t.Errorf("apart %v m at periapsis, want %v", got, periapsis)
	}
	m1, m2 := 2.45, 0.62
	if got, want := one/other, m2/m1; math.Abs(got-want) > want*1e-9 {
		t.Errorf("distances are in the ratio %v, want %v", got, want)
	}
	if !(one < other) {
		t.Errorf("the heavier star is at %v m, the lighter at %v", one,
			other)
	}
}

// The anti-test: the stars never fail to balance about the centre of
// mass. Their masses times their places sum to nothing at any time, on a
// tipped orbit as on a flat one.
func TestTheStarsAlwaysBalanceAboutTheCentreOfMass(t *testing.T) {
	binary := theFirstPair()
	binary.Orbit.Inclination = measure.Degrees(37)
	first, _ := binary.Place(0)
	for share := 0.0; share < 2; share += 0.0625 {
		pair, err := binary.Place(at(first, share))
		if err != nil {
			t.Fatal(err)
		}
		m1 := pair.Primary.Shine.Mass.InSolarMasses()
		m2 := pair.Companion.Shine.Mass.InSolarMasses()
		moment := pair.Primary.At.Scale(m1).Add(pair.Companion.At.Scale(m2))
		span := pair.Primary.At.Sub(pair.Companion.At).Len()
		if moment.Len() > span*1e-9 {
			t.Fatalf("at %v of a year the moments are %v, not nought",
				share, moment)
		}
	}
}

// Kepler's third law: 9.2 AU about 3.07 solar masses is a year of the
// root of 9.2 cubed over 3.07 years, about 15.9. The star is nearest at
// periapsis and farthest, by the eccentricity, half a year later.
func TestThePairTakesKeplersYearAndStretchesByItsEccentricity(t *testing.T) {
	binary := theFirstPair()
	pair, err := binary.Place(0)
	if err != nil {
		t.Fatal(err)
	}
	want := math.Sqrt(9.2 * 9.2 * 9.2 / 3.07)
	if got := float64(pair.Year); math.Abs(got-want) > want*2e-3 {
		t.Errorf("a year of %v, want %v", got, want)
	}
	far, _ := binary.Place(at(pair, 0.5))
	apoapsis := 9.2 * AstronomicalUnit * 1.16
	got := far.Primary.At.Sub(far.Companion.At).Len()
	if math.Abs(got-apoapsis) > apoapsis*1e-9 {
		t.Errorf("apart %v m at apoapsis, want %v", got, apoapsis)
	}
}

// The root of Kepler's equation satisfies it, at every eccentricity the
// pair may have, and a circle's is the mean anomaly itself.
func TestKeplersEquationIsSolved(t *testing.T) {
	for _, stretch := range []float64{0, 0.16, 0.5, 0.9, maxEccentricity} {
		for mean := 0.0; mean < 2*math.Pi; mean += 0.25 {
			anomaly := solveKepler(mean, stretch)
			if got := anomaly - stretch*math.Sin(anomaly); math.Abs(got-mean) > 1e-9 {
				t.Errorf("e %v, M %v: E %v gives M %v", stretch, mean,
					anomaly, got)
			}
		}
	}
	if got := solveKepler(1.25, 0); got != 1.25 {
		t.Errorf("a circle's E is %v, not its M 1.25", got)
	}
}

// The primary is the A-type star the pair is named for: its state comes
// from celestial/stars and is hot, white and many times the Sun's light.
func TestThePrimaryIsAnAType(t *testing.T) {
	shine, err := theFirstPair().Primary.Shine()
	if err != nil {
		t.Fatal(err)
	}
	heat := shine.Temperature.InKelvin()
	// celestial's table puts 2.45 solar masses at 10076 K: the hot end of
	// A, which runs 7500 to 10000, so the test allows a little over
	if !(heat >= 7500 && heat <= 10500) {
		t.Errorf("a primary of %v K is not A-type, 7500 to 10500", heat)
	}
	t.Logf("primary: %v K, %v solar luminosities, %v solar radii", heat,
		shine.Luminosity.InSolarLuminosities(),
		shine.Radius.InSolarRadii())
	if got := shine.Luminosity.InSolarLuminosities(); !(got > 10) {
		t.Errorf("a primary of %v solar luminosities is dim", got)
	}
}

// The dwarf's light follows from its given numbers: a sun's radius at
// the Sun's temperature gives the Sun's luminosity.
func TestARemnantsLuminosityIsStefanBoltzmanns(t *testing.T) {
	sun := Remnant{Name: "sun", Mass: measure.SolarMasses(1),
		Temperature: measure.Kelvin(5772), Radius: measure.SolarRadii(1)}
	shine, err := sun.Shine()
	if err != nil {
		t.Fatal(err)
	}
	if got := shine.Luminosity.InSolarLuminosities(); math.Abs(got-1) > 0.01 {
		t.Errorf("the Sun's numbers give %v solar luminosities", got)
	}
}

// The anti-tests for numbers that cannot make a pair: each is refused,
// and by name, not placed.
func TestNumbersThatCannotMakeAPairAreRefused(t *testing.T) {
	for name, spoil := range map[string]func(*Binary){
		"no separation":        func(b *Binary) { b.Orbit.Distance = measure.Metres(0) },
		"an eccentricity of 1": func(b *Binary) { b.Orbit.Eccentricity = 1 },
		"a negative eccentricity": func(b *Binary) {
			b.Orbit.Eccentricity = -0.1
		},
		"a year as well":  func(b *Binary) { b.Orbit.Year = 15 },
		"no primary mass": func(b *Binary) { b.Primary.Mass = measure.Kilograms(0) },
		"a negative age":  func(b *Binary) { b.Primary.Age = -1 },
		"a dwarf with no radius": func(b *Binary) {
			b.Companion.Radius = measure.Metres(0)
		},
		"a dwarf with no heat": func(b *Binary) {
			b.Companion.Temperature = measure.Kelvin(0)
		},
		"a primary past its life": func(b *Binary) {
			b.Primary.Age = 1.2e10
		},
	} {
		binary := theFirstPair()
		spoil(&binary)
		if _, err := binary.Place(0); err == nil {
			t.Errorf("%s was placed", name)
		}
	}
}

// A tipped orbit moves the stars out of the plane by the inclination, a
// quarter of the way round, and keeps their separation.
func TestAnInclinedOrbitTipsAboutTheLineOfPeriapsis(t *testing.T) {
	binary := theFirstPair()
	binary.Orbit.Eccentricity = 0
	binary.Orbit.Inclination = measure.Degrees(90)
	first, _ := binary.Place(0)
	pair, err := binary.Place(at(first, 0.25))
	if err != nil {
		t.Fatal(err)
	}
	apart := pair.Companion.At.Sub(pair.Primary.At)
	if want := maths.V(0, 9.2*AstronomicalUnit, 0); apart.Sub(want).Len() > 1e-6*au {
		t.Errorf("a quarter round a polar orbit they are %v apart, want %v",
			apart, want)
	}
}

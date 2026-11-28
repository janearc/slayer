package generate

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
)

// earthLight is the sunlight at the Earth, in watts a square metre: the
// solar constant, 1361 (Kopp and Lean 2011, from memory, to shelve).
const earthLight = 1361.0

// theLavaDrop is the drop of the second slice: a tenth of a solar mass,
// molten at 3000 K, old, 48 AU from the first pair's centre of mass.
func theLavaDrop() Drop {
	return Drop{Name: "the lava drop", Mass: measure.SolarMasses(0.1),
		Temperature: measure.Kelvin(3000), Age: 1e9,
		Distance: measure.Metres(48 * AstronomicalUnit)}
}

// A body of a tenth of a solar mass, held up by degenerate electrons at
// two nucleons each, is 0.027 solar radii, 2.9 Earth radii (Nauenberg
// 1972, from memory, to shelve), and the relation puts the white dwarf
// of 0.62 solar masses at 0.0124, near the 0.0122 the first slice gave.
func TestADegenerateBodyOfATenthOfASunIsThreeEarths(t *testing.T) {
	radius, err := DegenerateRadius(measure.SolarMasses(0.1), 2)
	if err != nil {
		t.Fatal(err)
	}
	if got := radius.InSolarRadii(); got < 0.0265 || got > 0.0275 {
		t.Errorf("a tenth of a solar mass is %v solar radii, want 0.027", got)
	}
	if got := radius.InMetres() / earthRadius; got < 2.8 || got > 3.0 {
		t.Errorf("%v Earth radii, want about 2.9", got)
	}
	dwarf, err := DegenerateRadius(measure.SolarMasses(0.62), 2)
	if err != nil {
		t.Fatal(err)
	}
	if got := dwarf.InSolarRadii(); got < 0.0120 || got > 0.0127 {
		t.Errorf("0.62 solar masses is %v solar radii, want 0.0124", got)
	}
}

// The anti-test: a mass that no degenerate electrons hold up, none or
// past the Chandrasekhar mass, has no radius from the relation, and a
// body with no nucleons for its electrons has none either.
func TestNoRadiusIsMadeForABodyThatIsNotDegenerate(t *testing.T) {
	for _, mass := range []float64{0, -1, 1.46, 3} {
		if _, err := DegenerateRadius(measure.SolarMasses(mass), 2); err == nil {
			t.Errorf("%v solar masses was given a radius", mass)
		}
	}
	if _, err := DegenerateRadius(measure.SolarMasses(0.1), 0); err == nil {
		t.Errorf("a body of no nucleons for each electron was given a radius")
	}
}

// The drop's light is the Stefan-Boltzmann law at its radius and 3000 K,
// and at 930,000 km it gives a planet about the Earth's sunlight, a
// little over: the numbers say 1.38 times, and nothing is changed to make
// it one.
func TestTheDropAt930000KmGivesAboutTheEarthsSunlight(t *testing.T) {
	shine, err := theLavaDrop().Shine()
	if err != nil {
		t.Fatal(err)
	}
	radius := shine.Radius.InMetres()
	want := 4 * math.Pi * radius * radius * stefanBoltzmann * math.Pow(3000, 4)
	if got := shine.Luminosity.InWatts(); math.Abs(got-want) > want*1e-9 {
		t.Errorf("a luminosity of %v W, want %v", got, want)
	}
	const apart = 930000e3
	flux := shine.Luminosity.InWatts() / (4 * math.Pi * apart * apart)
	t.Logf("%.0f W/m^2 at 930,000 km, %.2f times the Earth's", flux,
		flux/earthLight)
	if got := flux / earthLight; got < 1.2 || got > 1.6 {
		t.Errorf("%v times the Earth's sunlight, want about 1.4", got)
	}
	if !shine.Molten {
		t.Errorf("the drop's shine does not say it is molten")
	}
}

// The drop is the size the cold degenerate relation gives, to the metre:
// no share is taken off it. The anti-test: a drop's lines, young or old,
// say nothing of its radius being corrected, since nothing is, and a
// young drop, whose heat is still made, says nothing at all.
func TestTheDropKeepsTheRelationsRadiusAndSaysNothingMadeUpOfIt(t *testing.T) {
	drop := theLavaDrop()
	shine, err := drop.Shine()
	if err != nil {
		t.Fatal(err)
	}
	radius, err := DegenerateRadius(drop.Mass, silicateNucleons)
	if err != nil {
		t.Fatal(err)
	}
	if shine.Radius.InMetres() != radius.InMetres() {
		t.Errorf("the drop is %v m, the relation gives %v m",
			shine.Radius.InMetres(), radius.InMetres())
	}
	for _, line := range drop.MadeUp() {
		if strings.Contains(line, "radius") ||
			strings.Contains(line, "per cent") {
			t.Errorf("a drop says its radius is made up: %q", line)
		}
	}
	drop.Age = 1e6
	if lines := drop.MadeUp(); len(lines) != 0 {
		t.Errorf("a young drop says %q", lines)
	}
}

// A planet that is given the light it gets, and no distance, is put where
// the drop's light comes to that, to a few per cent: the Earth's sunlight
// is a distance of about 1.09 million km from the drop at 3000 K, whose
// angular size from there is near 2 degrees. The anti-test: twice the
// light is nearer by the square root of two, and a planet given both a
// distance and a light, or a light and no sun, is refused.
func TestAPlanetGivenItsLightIsPutWhereTheDropGivesIt(t *testing.T) {
	shine, err := theLavaDrop().Shine()
	if err != nil {
		t.Fatal(err)
	}
	planet := Planet{Name: "caprica", Mass: measure.Kilograms(EarthMass),
		Makeup: Makeup{Metal: 0.32, Rock: 0.6798, Ice: 0.0002},
		Sun: Sun{Mass: shine.Mass,
			Luminosity: shine.Luminosity.InSolarLuminosities()},
		Orbit: Orbit{Flux: 1}, Locked: true, Age: 4.5e9, Seed: 1}
	where, err := planet.Distance()
	if err != nil {
		t.Fatal(err)
	}
	apart := where.InMetres()
	t.Logf("%.0f m from the drop, %.0f km", apart, apart/1e3)
	flux := shine.Luminosity.InWatts() / (4 * math.Pi * apart * apart)
	if got := flux / earthLight; got < 0.97 || got > 1.03 {
		t.Errorf("%v times the Earth's sunlight at %v m, want 1", got, apart)
	}
	if got := apart / 1e9; got < 1.05 || got > 1.13 {
		t.Errorf("%v million km from the drop, want about 1.09", got)
	}
	size := 2 * math.Atan(shine.Radius.InMetres()/apart) * 180 / math.Pi
	if size < 1.9 || size > 2.05 {
		t.Errorf("the drop is %v degrees across, want about 2", size)
	}
	planet.Orbit.Flux = 2
	if nearer, _ := planet.Distance(); math.Abs(nearer.InMetres()*
		math.Sqrt2/apart-1) > 1e-9 {
		t.Errorf("twice the light is at %v m, not 1/sqrt 2 of %v",
			nearer.InMetres(), apart)
	}
	planet.Orbit = Orbit{Flux: 1, Distance: measure.Metres(1e9)}
	if _, err := planet.Distance(); err == nil {
		t.Errorf("a planet given a distance and a light was placed")
	}
	planet.Orbit = Orbit{Flux: 1}
	planet.Sun = Sun{}
	if _, err := planet.Distance(); err == nil {
		t.Errorf("a planet given a light and no sun was placed")
	}
}

// The anti-test: a body under the hydrogen-burning limit is no lava drop,
// and is refused, and so is one with no heat.
func TestABodyUnderTheBurningLimitIsNotADrop(t *testing.T) {
	light := theLavaDrop()
	light.Mass = measure.SolarMasses(0.05)
	if _, err := light.Shine(); err == nil {
		t.Errorf("a body of 0.05 solar masses was made a lava drop")
	}
	cold := theLavaDrop()
	cold.Temperature = measure.Kelvin(0)
	if _, err := cold.Shine(); err == nil {
		t.Errorf("a drop with no heat was given a shine")
	}
}

// A drop whose heat is long gone is held at its temperature by nothing,
// and says so in a line that begins "made up:" and names the temperature;
// the anti-test: a drop young enough that its aluminium and iron still
// heat it says nothing.
func TestAnOldDropSaysItsTemperatureIsMadeUp(t *testing.T) {
	lines := theLavaDrop().MadeUp()
	if len(lines) != 1 || !strings.HasPrefix(lines[0], "made up:") ||
		!strings.Contains(lines[0], "3000 K") {
		t.Errorf("an old drop says %q", lines)
	}
	young := theLavaDrop()
	young.Age = 1e6
	if lines := young.MadeUp(); len(lines) != 0 {
		t.Errorf("a young drop says %q", lines)
	}
}

// theDroppedPair is the first pair with the lava drop going round it.
func theDroppedPair() Binary {
	binary := theFirstPair()
	drop := theLavaDrop()
	binary.Drop = &drop
	return binary
}

// The drop goes round the pair's centre of mass at the distance it is
// given, its year Kepler's for the three masses, about 187 years, and a
// pair with no drop has none.
func TestTheDropGoesRoundTheCentreOfMassAtItsDistance(t *testing.T) {
	pair, err := theDroppedPair().Place(0)
	if err != nil {
		t.Fatal(err)
	}
	want := 48 * AstronomicalUnit
	if got := pair.Drop.At.Len(); math.Abs(got-want) > want*1e-9 {
		t.Errorf("the drop stands %v m out, want %v", got, want)
	}
	year := math.Sqrt(48 * 48 * 48 / 3.17)
	if got := float64(pair.DropYear); math.Abs(got-year) > year*5e-3 {
		t.Errorf("a year of %v, want %v", got, year)
	}
	later, _ := theDroppedPair().Place(pair.DropYear / 4)
	if got := later.Drop.At.Dot(pair.Drop.At); math.Abs(got) > want*want*1e-9 {
		t.Errorf("a quarter of a year on, the drop is not square to where "+
			"it was: %v", got)
	}
	if alone, _ := theFirstPair().Place(0); alone.Drop != nil {
		t.Errorf("a pair with no drop was given one")
	}
}

// The anti-test: a drop nearer than the hierarchy law allows is refused,
// since it would throw a star out, and a drop with no distance too.
func TestADropTooNearThePairIsRefused(t *testing.T) {
	binary := theDroppedPair()
	binary.Drop.Distance = measure.Metres(20 * AstronomicalUnit)
	if _, err := binary.Place(0); err == nil ||
		!strings.Contains(err.Error(), "stable") {
		t.Errorf("a drop at 20 AU was placed: %v", err)
	}
	binary.Drop.Distance = measure.Metres(0)
	if _, err := binary.Place(0); err == nil {
		t.Errorf("a drop at no distance was placed")
	}
}

// A planet locked to the drop at 930,000 km has a day of 13.6 hours, its
// year, and the drop's light is its only sun.
func TestAPlanetLockedToTheDropTurnsOnceAYear(t *testing.T) {
	planet := Planet{Name: "caprica", Mass: measure.Kilograms(EarthMass),
		Locked: true, Sun: Sun{Mass: measure.SolarMasses(0.1),
			Luminosity: 5.3e-5},
		Orbit: Orbit{Distance: measure.Metres(930000e3)}}
	turn, err := Turn(planet)
	if err != nil {
		t.Fatal(err)
	}
	if got := turn.Hours(); got < 13.5 || got > 13.7 {
		t.Errorf("a day of %v hours, want 13.6", got)
	}
	year, _ := planet.Year()
	if turn != year {
		t.Errorf("a locked planet turns in %v, its year is %v", turn, year)
	}
}

// The anti-test: a locked planet given a day that is not its year is
// refused, one with no sun to be locked to is, and a planet not locked
// turns in the day it is given, whatever its year.
func TestALockedPlanetsDayIsNeverItsOwn(t *testing.T) {
	planet := Planet{Name: "caprica", Mass: measure.Kilograms(EarthMass),
		Locked: true, Day: 24 * time.Hour,
		Sun:   Sun{Mass: measure.SolarMasses(0.1), Luminosity: 5.3e-5},
		Orbit: Orbit{Distance: measure.Metres(930000e3)}}
	if _, err := Turn(planet); err == nil {
		t.Errorf("a locked planet was given a day of 24 hours")
	}
	planet.Sun = Sun{}
	planet.Day = 0
	if _, err := Turn(planet); err == nil {
		t.Errorf("a locked planet with no sun was given a day")
	}
	planet.Locked, planet.Day = false, 24*time.Hour
	if turn, err := Turn(planet); err != nil || turn != 24*time.Hour {
		t.Errorf("a planet not locked turns in %v, %v", turn, err)
	}
}

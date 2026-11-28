package generate

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/stars"
	"github.com/janearc/slayer/maths"
)

// Star is what a caller gives to make a star burning on the main
// sequence: its birth mass and its age. Its life stage, temperature,
// luminosity and radius come from celestial/stars. Metallicity, spin and
// a seed, which doc/system-generators-design.md names, are not read yet.
type Star struct {
	Name string
	Mass measure.Mass
	Age  Years
}

// Remnant is a white dwarf given by its own numbers. Celestial has no
// model of one yet, so these three are GIVEN, not derived: whoever makes
// the Remnant says them, and nothing here checks them against a cooling
// track or a mass and radius relation.
type Remnant struct {
	Name        string
	Mass        measure.Mass
	Temperature measure.Temperature
	Radius      measure.Length
}

// Shine is what a star gives, as a stage draws it: the numbers of its
// light and its size. Molten is a body whose surface is crust drifting on
// magma, a lava drop's, which a stage draws with the crust.
type Shine struct {
	Name        string
	Mass        measure.Mass
	Temperature measure.Temperature
	Luminosity  measure.Power
	Radius      measure.Length
	Molten      bool
}

// Binary is what a caller gives to make two stars going round each
// other: a main-sequence primary, a companion given by its own numbers,
// and their orbit. The orbit's Distance is the semi-major axis of one
// star's orbit about the other, their separation; the Year follows from
// Kepler's third law and is not given. Eccentricity and Inclination are
// the orbit's; the periapsis lies along the world's x, and the orbit
// along x and z when the inclination is nought.
type Binary struct {
	Name      string
	Primary   Star
	Companion Remnant
	Orbit     Orbit
	// Drop is a lava drop going round the pair's centre of mass, or nil
	// where there is none.
	Drop *Drop
}

// Placed is a star at a place, in metres from the pair's centre of mass.
type Placed struct {
	Shine Shine
	At    maths.Vec
}

// Pair is the two stars of a Binary at one time, and the time they take
// to go once round. Drop is the lava drop, where the binary has one, and
// DropYear the time it takes to go once round the pair.
type Pair struct {
	Primary, Companion Placed
	Year               Years
	Drop               *Placed
	DropYear           Years
}

// Stefan-Boltzmann's constant in watts a square metre a kelvin to the
// fourth: CODATA 2018, exact from the SI constants.
const stefanBoltzmann = 5.670374419e-8

// maxEccentricity is the greatest eccentricity a binary may have. Picked,
// by eye, 2026-11-26: Newton's step in solveKepler converges well below
// it, and a pair with more is nearly a plunge, which the design leaves
// to integrated orbits.
const maxEccentricity = 0.99

// keplerTolerance is how close, in radians, the eccentric anomaly must
// come to the root of Kepler's equation. Picked: far below a pixel.
const keplerTolerance = 1e-13

// keplerSteps is the most Newton steps solveKepler takes. Picked: from
// the start M + e sin M it needs under ten at the greatest eccentricity.
const keplerSteps = 50

// Shine is the star's state at its age: celestial/stars gives its
// temperature, luminosity and radius. A star that has left the main
// sequence is refused; a remnant is made as a Remnant.
func (star Star) Shine() (Shine, error) {
	if !star.Mass.Positive() {
		return Shine{}, fmt.Errorf("generate: %s has no mass", star.Name)
	}
	if star.Age < 0 {
		return Shine{}, fmt.Errorf("generate: %s is %v years old, less "+
			"than nothing", star.Name, float64(star.Age))
	}
	state, err := stars.NewStar(star.Mass,
		measure.Years(float64(star.Age)))
	if err != nil {
		return Shine{}, fmt.Errorf("generate: %s: %w", star.Name, err)
	}
	if state.Life() != stars.Burning() {
		return Shine{}, fmt.Errorf("generate: %s has left the main "+
			"sequence at %v years: give it as a remnant", star.Name,
			float64(star.Age))
	}
	return Shine{Name: star.Name, Mass: star.Mass,
		Temperature: state.Temperature(), Luminosity: state.Luminosity(),
		Radius: state.Radius()}, nil
}

// Shine is the remnant's light: the numbers given, and the luminosity
// from the Stefan-Boltzmann law, four pi radius squared sigma temperature
// to the fourth. It refuses a number that is not above nought.
func (remnant Remnant) Shine() (Shine, error) {
	heat, radius := remnant.Temperature.InKelvin(),
		remnant.Radius.InMetres()
	if !remnant.Mass.Positive() || !(heat > 0) || !(radius > 0) {
		return Shine{}, fmt.Errorf("generate: %s needs a mass, a "+
			"temperature and a radius above nought", remnant.Name)
	}
	watts := 4 * math.Pi * radius * radius * stefanBoltzmann *
		math.Pow(heat, 4)
	return Shine{Name: remnant.Name, Mass: remnant.Mass,
		Temperature: remnant.Temperature, Luminosity: measure.Watts(watts),
		Radius: remnant.Radius}, nil
}

// Place puts both stars on their orbits at a time after periapsis, round
// the centre of mass, which is at the origin. The two orbit it on
// opposite sides, each at a distance in proportion to the other's mass:
// the primary's m2 over the total of the separation, the companion's m1
// over the total (Murray and Dermott, Solar System Dynamics, 1.2; from
// memory, to shelve). It refuses a number that cannot make a pair.
func (binary Binary) Place(time Years) (Pair, error) {
	primary, err := binary.Primary.Shine()
	if err != nil {
		return Pair{}, err
	}
	companion, err := binary.Companion.Shine()
	if err != nil {
		return Pair{}, err
	}
	year, err := binary.year(primary.Mass, companion.Mass)
	if err != nil {
		return Pair{}, err
	}
	relative := binary.relative(float64(time) / float64(year))
	total := primary.Mass.InKilograms() + companion.Mass.InKilograms()
	pair := Pair{
		Primary: Placed{primary,
			relative.Scale(-companion.Mass.InKilograms() / total)},
		Companion: Placed{companion,
			relative.Scale(primary.Mass.InKilograms() / total)},
		Year: year}
	if binary.Drop == nil {
		return pair, nil
	}
	drop, dropYear, err := binary.Drop.placed(binary, pair, time)
	if err != nil {
		return Pair{}, err
	}
	pair.Drop, pair.DropYear = &drop, dropYear
	return pair, nil
}

// year is the time once round, in years, by Kepler's third law for the
// two masses together, or a refusal of an orbit that cannot be placed.
func (binary Binary) year(one, other measure.Mass) (Years, error) {
	orbit := binary.Orbit
	switch {
	case !(orbit.Distance.InMetres() > 0):
		return 0, fmt.Errorf("generate: %s has no separation", binary.Name)
	case !(orbit.Eccentricity >= 0 && orbit.Eccentricity <= maxEccentricity):
		return 0, fmt.Errorf("generate: %s has an eccentricity of %v, "+
			"outside nought to %v", binary.Name, orbit.Eccentricity,
			maxEccentricity)
	case orbit.Year != 0:
		return 0, fmt.Errorf("generate: %s is given a separation, and its "+
			"year follows from it", binary.Name)
	}
	heavy := measure.Kilograms(one.InKilograms() + other.InKilograms())
	pull := measure.GM(heavy).InMetres() * speedOfLight * speedOfLight
	axis := orbit.Distance.InMetres()
	seconds := 2 * math.Pi * math.Sqrt(axis*axis*axis/pull)
	return Years(seconds / julianYear), nil
}

// relative is where the companion is from the primary, in metres, after
// a number of turns since periapsis: round an ellipse in the plane of x
// and z, tipped about x by the inclination. The eccentric anomaly E
// solves Kepler's equation M = E - e sin E, and the place is a(cos E - e)
// along the periapsis and a root(1 - e squared) sin E across it
// (Murray and Dermott 2.4; from memory, to shelve).
func (binary Binary) relative(turns float64) maths.Vec {
	orbit := binary.Orbit
	axis, stretch := orbit.Distance.InMetres(), orbit.Eccentricity
	anomaly := solveKepler(2*math.Pi*math.Mod(turns, 1), stretch)
	along := axis * (math.Cos(anomaly) - stretch)
	across := axis * math.Sqrt(1-stretch*stretch) * math.Sin(anomaly)
	tilt := orbit.Inclination.InRadians()
	return maths.V(along, across*math.Sin(tilt), across*math.Cos(tilt))
}

// solveKepler is the eccentric anomaly E, in radians, for a mean anomaly
// M and an eccentricity e below one: the root of M = E - e sin E, by
// Newton's method from E = M + e sin M.
func solveKepler(mean, stretch float64) float64 {
	anomaly := mean + stretch*math.Sin(mean)
	for range keplerSteps {
		step := (anomaly - stretch*math.Sin(anomaly) - mean) /
			(1 - stretch*math.Cos(anomaly))
		anomaly -= step
		if math.Abs(step) < keplerTolerance {
			break
		}
	}
	return anomaly
}

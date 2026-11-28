package generate

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// Drop is what a caller gives to make a lava drop: a body above the
// hydrogen-burning limit, mostly silicate, held up by the pressure of
// its degenerate electrons, its magma ocean melted by the decay of
// aluminium-26 and iron-60 (Mishra et al. 2026, arXiv 2605.19241, sec
// 4.1.1: "predominantly silicate", a degenerate core, a magma ocean, an
// outgassed atmosphere; the paper gives no structure or numbers beyond
// that). It goes on a circle round the centre of mass of a Binary.
//
// Its radius is not given: it follows from its mass by the cold
// degenerate relation (see DegenerateRadius). Its atmosphere is not
// drawn: at thousands of times the Earth's gravity it is metres thick.
type Drop struct {
	Name string
	Mass measure.Mass
	// Temperature is the magma's, in kelvin.
	Temperature measure.Temperature
	// Age is how long since it melted, in years. The heat that melted it
	// decays with it; see MadeUp.
	Age Years
	// Distance is the radius of its circle round the binary's centre of
	// mass.
	Distance measure.Length
	// Phase is where on its circle it stands at the binary's periapsis.
	Phase measure.Angle
}

// DegenerateRadius is the radius of a cold body held up by degenerate
// electrons alone, from its mass and the nucleons it has for each
// electron: Nauenberg 1972 (Am. J. Phys. 40, 1133; from memory, to
// shelve), with the Chandrasekhar mass 5.816 over the square of that
// number, in the Sun's masses, and the radius
//
//	0.0225 / perElectron * sqrt((M / Mch)^(-2/3) - (M / Mch)^(2/3))
//
// in the Sun's radii. It has no Coulomb correction between the ions,
// which for silicate and iron makes a body somewhat smaller, so the
// radius is an upper bound, within a fifth at a tenth of a solar mass.
// It refuses a mass that is not above nought and below the
// Chandrasekhar mass.
func DegenerateRadius(mass measure.Mass, perElectron float64) (measure.Length,
	error) {
	if !(perElectron > 0) {
		return measure.Length{}, fmt.Errorf("generate: %v nucleons for "+
			"each electron is no number to build a body from",
			perElectron)
	}
	limit := chandrasekharCoefficient / (perElectron * perElectron)
	share := mass.InSolarMasses() / limit
	if !(share > 0 && share < 1) {
		return measure.Length{}, fmt.Errorf("generate: a body of %.4g "+
			"solar masses is not held up by degenerate electrons, whose "+
			"limit is %.4g", mass.InSolarMasses(), limit)
	}
	stretch := math.Pow(share, -twoThirds) - math.Pow(share, twoThirds)
	return measure.SolarRadii(nauenbergRadius / perElectron *
		math.Sqrt(stretch)), nil
}

// The numbers of Nauenberg's relation, in the Sun's masses and radii; from
// memory, to shelve.
const (
	chandrasekharCoefficient = 5.816
	nauenbergRadius          = 0.0225
	twoThirds                = 2.0 / 3.0
)

// silicateNucleons is the nucleons for each electron of a body of
// silicate: magnesium, silicon and oxygen all have two, as carbon and
// oxygen do in a white dwarf.
const silicateNucleons = 2.0

// leastDrop is the least a lava drop weighs, in the Sun's masses: the
// hydrogen-burning limit, near 0.075 (Chabrier and Baraffe 2000, from
// memory, to shelve). Under it a body is a brown dwarf, not held up by
// its electrons alone.
const leastDrop = 0.075

// The heat of a lava drop's magma: iron-60's half-life of 2.62 million
// years (Rugel et al. 2009, from memory, to shelve), the longer of the
// two the paper names, and how many half-lives its heat is counted to
// last, ten, which leaves a thousandth of it. Picked, 2026-11-26.
const (
	heatHalfLife   = 2.62e6
	heatHalfLives  = 10
	heatLastsYears = heatHalfLife * heatHalfLives
)

// hierarchyFactor is the nearest a third body may go round a pair, in
// the pair's separations, for the system to be stable on a circle:
// 2.8 (1 + q)^(2/5), q the third's mass over the pair's (Mardling and
// Aarseth 2001, from memory, to shelve).
const hierarchyFactor = 2.8

// hierarchyPower is the power of the mass ratio in the same law.
const hierarchyPower = 2.0 / 5.0

// Shine is the drop's light: the radius from the degenerate relation, as
// it stands, with no correction made to it, and the luminosity from the
// Stefan-Boltzmann law, four pi radius squared sigma temperature to the
// fourth, at the magma's temperature. It is molten, so a stage draws it
// with a crust. It refuses a mass under the hydrogen-burning limit and a
// temperature that is not above nought.
func (drop Drop) Shine() (Shine, error) {
	if drop.Mass.InSolarMasses() < leastDrop {
		return Shine{}, fmt.Errorf("generate: %s weighs %.4g solar "+
			"masses, under the hydrogen-burning limit of %v: it is not "+
			"a lava drop", drop.Name, drop.Mass.InSolarMasses(), leastDrop)
	}
	radius, err := DegenerateRadius(drop.Mass, silicateNucleons)
	if err != nil {
		return Shine{}, fmt.Errorf("%s: %w", drop.Name, err)
	}
	shine, err := Remnant{Name: drop.Name, Mass: drop.Mass,
		Temperature: drop.Temperature, Radius: radius}.Shine()
	shine.Molten = true
	return shine, err
}

// MadeUp is a line for each number of the drop that is made up, and no
// line where every number stands on a source. The drop's heat comes from
// aluminium-26 and iron-60, which is gone after heatLastsYears: a drop
// older than that is held at its temperature by nothing, and the
// temperature is made up. Its radius is the relation's and is not. A
// caller that draws the drop says each line.
func (drop Drop) MadeUp() []string {
	var lines []string
	if float64(drop.Age) > heatLastsYears {
		lines = append(lines, fmt.Sprintf("made up: %s is molten at %.0f "+
			"K, and its heat, the decay of aluminium-26 and iron-60 "+
			"(Mishra et al. 2026, sec 4.1.1), is gone after %.3g years: "+
			"it is %.3g years old, so nothing holds it molten", drop.Name,
			drop.Temperature.InKelvin(), float64(heatLastsYears),
			float64(drop.Age)))
	}
	return lines
}

// placed is where the drop stands at a time after the binary's
// periapsis, on a circle round the centre of mass (the origin) in the
// binary's own plane, at a phase from the periapsis's side; and its year,
// by Kepler's third law for the three masses together. The pair goes on
// as it does alone: the drop's pull on it, a thirtieth of the pair's own,
// is left out. It refuses a drop closer than the hierarchy law allows,
// which would throw one of the three out, and a distance that is not
// above nought.
func (drop Drop) placed(binary Binary, pair Pair, time Years) (Placed, Years,
	error) {
	shine, err := drop.Shine()
	if err != nil {
		return Placed{}, 0, err
	}
	inner := pair.Primary.Shine.Mass.InKilograms() +
		pair.Companion.Shine.Mass.InKilograms()
	axis := drop.Distance.InMetres()
	if err := hierarchical(drop, binary, inner, axis); err != nil {
		return Placed{}, 0, err
	}
	pull := measure.GM(measure.Kilograms(inner+drop.Mass.InKilograms())).
		InMetres() * speedOfLight * speedOfLight
	year := Years(2 * math.Pi * math.Sqrt(axis*axis*axis/pull) / julianYear)
	angle := drop.Phase.InRadians() + 2*math.Pi*float64(time)/float64(year)
	tilt := binary.Orbit.Inclination.InRadians()
	along, across := axis*math.Cos(angle), axis*math.Sin(angle)
	return Placed{Shine: shine, At: maths.V(along, across*math.Sin(tilt),
		across*math.Cos(tilt))}, year, nil
}

// hierarchical is nothing when a drop at a distance, round a pair of a
// mass in kilograms and a separation, is further than the hierarchy law
// asks, and the refusal that says so when it is not.
func hierarchical(drop Drop, binary Binary, inner, axis float64) error {
	if !(axis > 0) {
		return fmt.Errorf("generate: %s has no distance from the pair",
			drop.Name)
	}
	ratio := drop.Mass.InKilograms() / inner
	least := hierarchyFactor * math.Pow(1+ratio, hierarchyPower) *
		binary.Orbit.Distance.InMetres()
	if axis < least {
		return fmt.Errorf("generate: %s is %.3g AU from the pair, nearer "+
			"than the %.3g AU its orbit is stable at: one of the three "+
			"would be thrown out", drop.Name, axis/AstronomicalUnit,
			least/AstronomicalUnit)
	}
	return nil
}

package materials

import (
	"math"

	"github.com/janearc/celestial/radiation"
)

// electronvolts is how many electronvolts are in one MeV, the unit
// NIST's tables write a photon's energy in
const electronvolts = 1e6

// centimetres is how many centimetres are in a metre. A coefficient in
// square centimetres a gram, times a density in grams a cubic
// centimetre, is what a beam loses each centimetre; a thickness is
// given in metres.
const centimetres = 100

// Attenuation is a mass attenuation coefficient, mu over rho, at one
// photon energy, with where it was read. The energy is in MeV and the
// coefficient in square centimetres a gram, as NIST's tables write
// them. Nothing is stored of what gets through a thickness: a
// material's Passes works it out from the coefficient, the density and
// the thickness.
type Attenuation struct {
	energy    float64
	muOverRho float64
	where     Source
}

// newAttenuation is mu over rho, in square centimetres a gram, at a
// photon energy in MeV, read somewhere. It refuses an energy or a
// coefficient that is not a positive number, and a source never made.
func newAttenuation(energy, muOverRho float64,
	where Source) (Attenuation, error) {
	if !positive(energy) {
		return Attenuation{}, refuse("an attenuation at energy %v MeV",
			energy)
	}
	if !positive(muOverRho) {
		return Attenuation{}, refuse("an attenuation of %v cm2/g",
			muOverRho)
	}
	if where.release == "" {
		return Attenuation{}, refuse("an attenuation read nowhere")
	}
	return Attenuation{energy: energy, muOverRho: muOverRho,
		where: where}, nil
}

// Energy is the photon energy the coefficient was read at, in MeV
func (stop Attenuation) Energy() float64 { return stop.energy }

// MuOverRho is the mass attenuation coefficient, in square centimetres
// a gram
func (stop Attenuation) MuOverRho() float64 { return stop.muOverRho }

// Where is where the coefficient was read
func (stop Attenuation) Where() Source { return stop.where }

// Wavelength is the photon's wavelength in nanometres, worked out from
// its energy by celestial, which holds the constants
func (stop Attenuation) Wavelength() float64 {
	return radiation.FromEnergy(stop.energy * electronvolts)
}

// Attenuations are a material's mass attenuation coefficients, each at
// its own photon energy. Only NewAttenuations makes them. Nothing is
// interpolated between them: a coefficient is answered at an energy a
// table gives, or not at all.
type Attenuations struct {
	readings []Attenuation
}

// NewAttenuations is coefficients, each at its own energy. It refuses
// none, a coefficient never made, and two at one energy.
func NewAttenuations(readings ...Attenuation) (Attenuations, error) {
	if len(readings) == 0 {
		return Attenuations{}, refuse("attenuations of no readings")
	}
	seen := make(map[float64]bool, len(readings))
	for _, stop := range readings {
		if stop.energy == 0 {
			return Attenuations{}, refuse("an attenuation never made")
		}
		if seen[stop.energy] {
			return Attenuations{}, refuse("two attenuations at %v MeV",
				stop.energy)
		}
		seen[stop.energy] = true
	}
	return Attenuations{
		readings: append([]Attenuation(nil), readings...)}, nil
}

// At is the coefficient read at a photon energy in MeV, and whether
// there is one. The energy is matched exactly, as its table writes it.
func (stops Attenuations) At(energy float64) (Attenuation, bool) {
	for _, stop := range stops.readings {
		if stop.energy == energy {
			return stop, true
		}
	}
	return Attenuation{}, false
}

// All is every coefficient, in the order they were given; the caller's
// to keep, and none for coefficients never made
func (stops Attenuations) All() []Attenuation {
	return append([]Attenuation(nil), stops.readings...)
}

// Within is every coefficient whose photon falls in one of celestial's
// ranges, in the order they were given; none when the range holds none
func (stops Attenuations) Within(span radiation.Range) []Attenuation {
	var held []Attenuation
	for _, stop := range stops.readings {
		if span.Holds(stop.Wavelength()) {
			held = append(held, stop)
		}
	}
	return held
}

// atDensity is the material at a bulk density, in grams a cubic
// centimetre, read somewhere. It refuses a material never made, a
// density that is not a positive number, and a source never made.
func (kind Material) atDensity(density float64,
	where Source) (Material, error) {
	if kind.name == "" {
		return Material{}, refuse("a density for a material never made")
	}
	if !positive(density) {
		return Material{}, refuse("%s at density %v", kind.name, density)
	}
	if where.release == "" {
		return Material{}, refuse("%s at a density read nowhere",
			kind.name)
	}
	kind.density = density
	kind.weighed = where
	return kind, nil
}

// DensityFrom is where the material's density was read; a source never
// made when the density was given with none, or none is cited
func (kind Material) DensityFrom() Source { return kind.weighed }

// Attenuating is the material with its mass attenuation coefficients.
// It refuses a material never made and coefficients never made.
func (kind Material) Attenuating(stops Attenuations) (Material, error) {
	if kind.name == "" {
		return Material{}, refuse("attenuations for a material never made")
	}
	if len(stops.readings) == 0 {
		return Material{}, refuse("%s has no attenuations", kind.name)
	}
	kind.stops = stops
	return kind, nil
}

// Attenuations is the material's mass attenuation coefficients,
// coefficients never made when it has none
func (kind Material) Attenuations() Attenuations { return kind.stops }

// Passes is the share of a beam, at a photon energy in MeV, that gets
// through a thickness of the material in metres: e to the minus mu over
// rho, times the density, times the thickness. It is worked out every
// time it is asked. It refuses an energy the material has no
// coefficient at, a material with no density cited, and a thickness
// that is negative or not a number.
func (kind Material) Passes(energy, thickness float64) (float64, error) {
	stop, there := kind.stops.At(energy)
	if !there {
		return 0, refuse("%s has no attenuation at %v MeV",
			kind.name, energy)
	}
	if kind.density == 0 {
		return 0, refuse("%s has no density cited", kind.name)
	}
	if negative(thickness) {
		return 0, refuse("%s through a thickness of %v m",
			kind.name, thickness)
	}
	return math.Exp(-stop.muOverRho * kind.density *
		thickness * centimetres), nil
}

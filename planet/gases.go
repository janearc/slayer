package planet

import (
	"math"
)

// boltzmann is joules a kelvin a molecule, exact since 2019
const boltzmann = 1.380649e-23

// sellmeier is a gas's refractive index less one at a wavelength in
// nanometres, from the refractiveindex.info database's formula 6: a
// constant, then for each pair a numerator over a pole less one over
// the wavelength in micrometres, squared.
func sellmeier(constant float64, pairs ...[2]float64) func(nm float64) float64 {
	return func(nm float64) float64 {
		inverse := 1 / (nm * nm * 1e-6)
		sum := constant
		for _, pair := range pairs {
			sum += pair[0] / (pair[1] - inverse)
		}
		return sum
	}
}

// scatters is a gas's rayleigh cross-section at a wavelength in
// nanometres, square metres a molecule, from its refractive index less
// one at a number of molecules a cubic metre and its king factor:
//
//	sigma = 24 pi^3 / (n^2 wavelength^4) ((m^2 - 1) / (m^2 + 2))^2 fk
//
// he and others' equation 1 (acp-21-14927), m the index and n the
// number at which it was measured.
func scatters(index func(nm float64) float64, number,
	king float64) func(nm float64) float64 {
	return func(nm float64) float64 {
		bent := 1 + index(nm)
		lorentz := (bent*bent - 1) / (bent*bent + 2)
		metres := nm * 1e-9
		return 24 * math.Pow(math.Pi, 3) / (number * number *
			math.Pow(metres, 4)) * lorentz * lorentz * king
	}
}

// standard is molecules a cubic metre at zero celsius and 101,325 pascals,
// where bideau-mehu and the others' indices hold.
var standard = 101325 / (boltzmann * 273.15)

// the noble gases' indices: bideau-mehu, guern, abjean and
// johannin-gilles 1981, as the refractiveindex.info database gives them
// (rii-ne-, rii-kr- and rii-xe-bideau-mehu).
//
// their fit joins the authors' own 140 to 254 nm to older, longer
// measurements. neon holds to 546 nm and krypton and xenon to 623 nm;
// past that it is the fit carried on, modelled. their king factor is
// one, by convention for atoms (acp-21-14927, table 1).
var (
	neonIndex = sellmeier(0, [2]float64{0.00128145, 184.661},
		[2]float64{0.0220486, 376.840})
	kryptonIndex = sellmeier(0, [2]float64{0.00253637, 65.4742},
		[2]float64{0.00273649, 73.698}, [2]float64{0.0620802, 181.08})
	xenonIndex = sellmeier(0, [2]float64{0.00322869, 46.301},
		[2]float64{0.00355393, 59.578}, [2]float64{0.0606764, 112.74})
)

// methaneIndex is methane's refractive index less one: he and the
// others' table 1 (acp-21-14927), after sneep and ubachs 2005. times
// 1e8 it is 4869.8 plus 4.1023e14 over 1.133e10 less the wavenumber
// squared, in reciprocal centimetres.
//
// it holds for 250 to 650 nm, at 288.15 k and 101,325 pascals, and
// 2.546899e19 molecules a cubic centimetre. its king factor is one.
func methaneIndex(nm float64) float64 {
	wavenumber := 1e7 / nm
	return (4869.8 + 4.1023e14/(1.133e10-wavenumber*wavenumber)) * 1e-8
}

// methaneNumber is the number at which methane's index holds, a cubic
// metre: theirs.
const methaneNumber = 2.546899e25

// the molar masses, kilograms a mole: iupac's standard atomic weights,
// recalled and not yet read from a pinned source.
const (
	neonMolar    = 0.0201797
	kryptonMolar = 0.083798
	xenonMolar   = 0.131293
	methaneMolar = 0.016043
)

// Neon is neon, by its scattering.
func Neon() Gas {
	return Gas{name: "neon", molar: neonMolar,
		section: scatters(neonIndex, standard, 1)}
}

// Krypton is krypton, by its scattering.
func Krypton() Gas {
	return Gas{name: "krypton", molar: kryptonMolar,
		section: scatters(kryptonIndex, standard, 1)}
}

// Xenon is xenon, by its scattering.
func Xenon() Gas {
	return Gas{name: "xenon", molar: xenonMolar,
		section: scatters(xenonIndex, standard, 1)}
}

// Methane is methane, by its scattering, absorbing as given.
func Methane(absorbing func(nm float64) float64) Gas {
	return Gas{name: "methane", molar: methaneMolar,
		section:   scatters(methaneIndex, methaneNumber, 1),
		absorbing: absorbing}
}

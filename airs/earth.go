package airs

import (
	"math"

	"github.com/janearc/celestial/atmosphere"
	"github.com/janearc/celestial/measure"
)

// Units of Earth's data, none of them a pick.
const (
	// partsPerMillion turns a share in parts a million into a fraction.
	partsPerMillion = 1e-6 // defined: the unit
	// kilogramsPerGram turns a molar mass in grams a mole into
	// kilograms a mole.
	kilogramsPerGram = 1e-3 // defined: the unit
)

// colour is what Earth's air does to light of one wavelength, apart from
// how much of it lies in a shell: the gas's and the ozone's cross
// sections in square metres a molecule, and the aerosol's optical depth
// for the whole column, its albedo and its asymmetry.
type colour struct {
	gasCross         float64
	ozoneCross       float64
	aerosolDepth     float64
	aerosolAlbedo    float64
	aerosolAsymmetry float64
}

// Earth is Earth's air at a wavelength in nanometres: the shells from the
// ground to the top of the standard atmosphere, 86 km, and the ground's
// albedo. It refuses a wavelength outside what the ozone's cross sections
// and Rayleigh's are known over, and a file of data that could not be
// read.
func Earth(nanometres float64) (Column, error) {
	if earthError != nil {
		return Column{}, earthError
	}
	return earthRead.column(nanometres)
}

// edges are the heights the air is cut at: each run of the data's cut
// into equal shells no thicker than the run's, one run on the last.
func (read earthData) edges() []float64 {
	edges := []float64{0}
	for _, run := range read.Shells.Runs {
		pieces := cuts([]float64{edges[len(edges)-1], run.To}, run.Thickness)
		edges = append(edges, pieces[1:]...)
	}
	return edges
}

// colour is the air's cross sections and the aerosol's depth, albedo and
// asymmetry at a wavelength in nanometres.
func (read earthData) colour(nanometres float64) (colour, error) {
	gas, err := read.Gas.cross(nanometres)
	if err != nil {
		return colour{}, err
	}
	ozone, err := read.Ozone.cross(nanometres)
	if err != nil {
		return colour{}, err
	}
	haze := read.Aerosol
	return colour{gasCross: gas, ozoneCross: ozone,
		aerosolDepth:     haze.depthAt(nanometres),
		aerosolAlbedo:    haze.Albedo.at(nanometres),
		aerosolAsymmetry: haze.Asymmetry.at(nanometres)}, nil
}

// column is the air at a wavelength in nanometres.
func (read earthData) column(nanometres float64) (Column, error) {
	hue, err := read.colour(nanometres)
	if err != nil {
		return Column{}, err
	}
	sampled, err := standardAt(read.edges())
	if err != nil {
		return Column{}, err
	}
	column := Column{Radius: read.Planet.Radius.Value,
		GroundAlbedo: read.Ground.albedo(nanometres)}
	for shell := 0; shell < sampled.shells(); shell++ {
		column.Layers = append(column.Layers, read.shell(sampled, shell, hue))
	}
	return column, nil
}

// shell is the air between two of the profile's edges: the gas by its
// cross section and the mass of air in the shell, the ozone by its cross
// section and its amount, and the share of the aerosol that lies there.
func (read earthData) shell(sampled profile, shell int, hue colour) Layer {
	bottom, top := sampled.edges[shell], sampled.edges[shell+1]
	thickness := top - bottom
	molecules := sampled.mass(shell) / thickness / read.Gas.molarMass() *
		avogadro
	dobson := read.Ozone.amount(sampled.pressure[shell],
		sampled.pressure[shell+1])
	return Layer{Bottom: bottom, Top: top,
		GasScattering: molecules * hue.gasCross,
		GasAbsorption: dobson * moleculesPerDobson / thickness *
			hue.ozoneCross,
		ParticleExtinction: hue.aerosolDepth *
			read.aerosolShare(bottom, top) / thickness,
		ParticleAlbedo:    hue.aerosolAlbedo,
		ParticleAsymmetry: hue.aerosolAsymmetry}
}

// molarMass is the gas's mean molar mass, in kilograms a mole: Bodhaine
// and others' dry air with its carbon dioxide.
func (gas earthGas) molarMass() float64 {
	grams := gas.DryAirMass.Value + gas.CarbonDioxideMassSlope.Value*
		gas.CarbonDioxide.Value*partsPerMillion
	return grams * kilogramsPerGram
}

// cross is the cross section of the gas's average molecule at a
// wavelength in nanometres, in square metres: celestial's dry air with
// the data's carbon dioxide, which is Peck and Reeder's index with the
// King factors of Bates and of Bodhaine and others.
func (gas earthGas) cross(nanometres float64) (float64, error) {
	air, err := atmosphere.DryAir(gas.CarbonDioxide.Value)
	if err != nil {
		return 0, err
	}
	area, err := air.CrossSection(measure.Metres(nanometres *
		metresPerNanometre))
	if err != nil {
		return 0, err
	}
	return area.InSquareMetres(), nil
}

// depthAt is the aerosol's optical depth for the column at a wavelength
// in nanometres: the depth at the reference wavelength, changed by
// Angstrom's law, the depth falling as the wavelength to the minus
// Angstrom exponent.
func (haze aerosolData) depthAt(nanometres float64) float64 {
	return haze.Depth.Value * math.Pow(nanometres/haze.Reference.Value,
		-haze.Angstrom.Value)
}

// aerosolShare is the share of the aerosol's depth that lies between two
// heights in metres: it falls by the scale height from the ground to the
// top of the air, and the shares of all the shells make one.
func (read earthData) aerosolShare(bottom, top float64) float64 {
	height := read.Aerosol.ScaleHeight.Value
	return (math.Exp(-bottom/height) - math.Exp(-top/height)) /
		(1 - math.Exp(-read.Gas.Top.Value/height))
}

package airs

import (
	_ "embed"
	"fmt"
	"math"

	"github.com/janearc/puzzlemouth/yamlfile"
)

// earthRaw is earth.yaml, the numbers Earth's air is made from, each with
// where it was read or why it was picked.
//
//go:embed earth.yaml
var earthRaw []byte

// earthRead is Earth's data as read and checked once, and earthError why
// it could not be: a file that cannot be read stops Earth, as Venus's
// stops Venus.
var earthRead, earthError = readEarth(earthRaw)

// earthPlanet is the planet.
type earthPlanet struct {
	Radius          number
	Gravity         number
	SurfacePressure number `yaml:"surface-pressure"`
}

// earthGas is what the gas is made of and how tall it is. Its density by
// height is the standard atmosphere's, not the data's.
type earthGas struct {
	CarbonDioxide          number `yaml:"carbon-dioxide"`
	DryAirMass             number `yaml:"dry-air-mass"`
	CarbonDioxideMassSlope number `yaml:"carbon-dioxide-mass-slope"`
	Top                    number
}

// runData is a run of equal shells: they end at a height, in metres,
// and are no thicker than a thickness, in metres.
type runData struct {
	To        float64
	Thickness float64
}

// shellData is how the air is cut: runs, one above another from the
// ground, and why they are as they are.
type shellData struct {
	Picked string
	Runs   []runData
}

// ozoneLayer is the ozone of one layer of the source's, between two
// pressures in hectopascals, the lower first, in Dobson units.
type ozoneLayer struct {
	Name   string
	Bottom float64
	Top    float64
	Amount float64
	Read   string
}

// ozoneData is the ozone: the temperature its cross sections are for,
// its whole column, the column's layers, and the cross sections.
type ozoneData struct {
	Temperature  number
	Total        number
	Layers       []ozoneLayer
	CrossSection tableData `yaml:"cross-section"`
}

// aerosolData is a clear day's haze: its optical depth at a reference
// wavelength, the exponent it changes with colour by, the height it falls
// by e in, and its albedo and asymmetry by wavelength.
type aerosolData struct {
	Depth       number
	Reference   number
	Angstrom    number
	ScaleHeight number `yaml:"scale-height"`
	Albedo      tableData
	Asymmetry   tableData
}

// earthData is everything earth.yaml holds.
type earthData struct {
	Planet  earthPlanet
	Gas     earthGas
	Shells  shellData
	Ozone   ozoneData
	Aerosol aerosolData
	Ground  groundData
}

// layerTolerance is how far the ozone's layers may be from the total, in
// Dobson units. Picked: rounding in the layers as printed, and no more.
const layerTolerance = 1e-9

// readEarth reads the data and checks it, so that making a column from it
// afterwards has nothing to refuse of it.
func readEarth(raw []byte) (earthData, error) {
	var read earthData
	if err := yamlfile.Read("earth.yaml", raw, &read); err != nil {
		return earthData{}, err
	}
	if err := read.check(); err != nil {
		return earthData{}, err
	}
	return read, nil
}

// numbers is every number in the data that carries a note.
func (read earthData) numbers() []entry {
	return []entry{
		{"planet radius", "m", read.Planet.Radius, true},
		{"planet gravity", "m s^-2", read.Planet.Gravity, true},
		{"planet surface pressure", "Pa", read.Planet.SurfacePressure, true},
		{"gas carbon dioxide", "parts a million", read.Gas.CarbonDioxide, true},
		{"gas dry air mass", "g mol^-1", read.Gas.DryAirMass, true},
		{"gas carbon dioxide mass slope", "g mol^-1",
			read.Gas.CarbonDioxideMassSlope, true},
		{"gas top", "m", read.Gas.Top, true},
		{"ozone temperature", "K", read.Ozone.Temperature, true},
		{"ozone total", "Dobson units", read.Ozone.Total, true},
		{"aerosol depth", "", read.Aerosol.Depth, true},
		{"aerosol reference wavelength", "nm", read.Aerosol.Reference, true},
		{"aerosol angstrom exponent", "", read.Aerosol.Angstrom, false},
		{"aerosol scale height", "m", read.Aerosol.ScaleHeight, true},
	}
}

// check is why the data is refused, if it is: any number, the shells, the
// ozone, the aerosol or the ground.
func (read earthData) check() error {
	for _, each := range read.numbers() {
		if err := each.check(); err != nil {
			return err
		}
	}
	for _, check := range []func() error{read.checkShells,
		read.checkOzone, read.checkAerosol, read.Ground.check} {
		if err := check(); err != nil {
			return err
		}
	}
	return nil
}

// checkShells is why the shells are refused, if they are: no note, no
// runs, a run with no thickness or that does not rise, or runs that end
// below or above the top of the air.
func (read earthData) checkShells() error {
	if read.Shells.Picked == "" || len(read.Shells.Runs) == 0 {
		return fmt.Errorf("airs: the shells have no picked note or no runs")
	}
	previous := 0.0
	for index, run := range read.Shells.Runs {
		if !(run.Thickness > 0 && run.To > previous) {
			return fmt.Errorf("airs: shell run %d, to %g with shells of %g, "+
				"has no thickness or does not rise", index+1, run.To,
				run.Thickness)
		}
		previous = run.To
	}
	if previous != read.Gas.Top.Value {
		return fmt.Errorf("airs: the shells end at %g, not at the top of the "+
			"air, %g", previous, read.Gas.Top.Value)
	}
	return nil
}

// checkOzone is why the ozone is refused, if it is: layers that do not
// run on from the surface pressure down, or do not make the total, or
// cross sections that are not a table of figures above nought.
func (read earthData) checkOzone() error {
	ozone := read.Ozone
	if err := ozone.checkLayers(read.Planet.SurfacePressure.Value /
		pascalsPerHectopascal); err != nil {
		return err
	}
	return ozone.CrossSection.checkWithin("the ozone cross section",
		math.SmallestNonzeroFloat64, math.MaxFloat64)
}

// checkLayers is why the layers are refused, if they are, given the
// pressure at the ground in hectopascals: unnoted, not beginning where
// the last one ended, not falling in pressure, of less than nought, or
// not adding to the total.
func (ozone ozoneData) checkLayers(ground float64) error {
	previous, sum := ground, 0.0
	for _, layer := range ozone.Layers {
		switch {
		case layer.Read == "":
			return fmt.Errorf("airs: %s has no note", layer.Name)
		case layer.Bottom != previous:
			return fmt.Errorf("airs: %s begins at %g hPa, a gap or an "+
				"overlap after %g", layer.Name, layer.Bottom, previous)
		case !(0 < layer.Top && layer.Top < layer.Bottom):
			return fmt.Errorf("airs: %s has no thickness", layer.Name)
		case !(layer.Amount >= 0):
			return fmt.Errorf("airs: %s holds %g Dobson units", layer.Name,
				layer.Amount)
		}
		previous, sum = layer.Top, sum+layer.Amount
	}
	if math.Abs(sum-ozone.Total.Value) > layerTolerance {
		return fmt.Errorf("airs: the ozone's layers make %g Dobson units, "+
			"not the total, %g", sum, ozone.Total.Value)
	}
	return nil
}

// checkAerosol is why the aerosol's tables are refused, if they are: an
// albedo outside nought to one or an asymmetry outside minus one to one,
// or either one without a note.
func (read earthData) checkAerosol() error {
	if err := read.Aerosol.Albedo.checkWithin("the aerosol's albedo", 0,
		1); err != nil {
		return err
	}
	return read.Aerosol.Asymmetry.checkWithin("the aerosol's asymmetry", -1, 1)
}

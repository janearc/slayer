package airs

import (
	_ "embed"
	"fmt"
	"math"

	"github.com/janearc/puzzlemouth/yamlfile"
)

// venusRaw is venus.yaml, the numbers Venus's air is made from, each with
// where it was read or why it was picked.
//
//go:embed venus.yaml
var venusRaw []byte

// venusRead is Venus's data as read and checked once, and venusError
// why it could not be: a file that cannot be read stops Venus, as the
// other embedded data files stop what they feed.
var venusRead, venusError = readData(venusRaw)

// number is a number with where it came from: read says in what source,
// where and in what words; picked says that nobody read it and why it is
// what it is. One of them is there and not both. Low and high are the
// range a pick was made within, where the source gives one.
type number struct {
	Value  float64
	Unit   string
	Read   string
	Picked string
	Low    float64
	High   float64
}

// planetData is the planet.
type planetData struct {
	Radius          number
	Gravity         number
	SurfacePressure number `yaml:"surface-pressure"`
	Distance        number
}

// refractionData is carbon dioxide's refractive index, less one, as a
// constant and a sum of terms, each a numerator over a pole less the
// inverse square of the wavelength in micrometres; the wavelengths in
// nanometres it holds over; and the temperature and pressure it was
// measured at.
type refractionData struct {
	Constant    float64
	Terms       [][2]float64
	Shortest    float64
	Longest     float64
	Temperature float64
	Pressure    float64
	Read        string
}

// gasData is the gas: what it is, how thick it lies, what scatters.
type gasData struct {
	CarbonDioxide     number `yaml:"carbon-dioxide"`
	Nitrogen          number
	CarbonDioxideMass number `yaml:"carbon-dioxide-mass"`
	NitrogenMass      number `yaml:"nitrogen-mass"`
	SurfaceDensity    number `yaml:"surface-density"`
	ScaleHeight       number `yaml:"scale-height"`
	Top               number
	Shell             number
	Absorption        number
	StandIn           number `yaml:"stand-in"`
	Refraction        refractionData
}

// particleData is what the droplets are made of, as light sees it, and
// the wavelength the layers' depths are given at.
type particleData struct {
	Real      number
	Imaginary number
	Reference number
}

// modeData is one mode of a layer's particles: a log-normal population,
// its diameter in micrometres, its number density in a cubic centimetre
// (only the modes' shares of the layer are used), and its width.
type modeData struct {
	Diameter float64
	Number   float64
	Read     string
	Width    number
}

// layerData is a cloud or a haze: its heights, its optical depth at the
// reference wavelength and the modes of its particles.
type layerData struct {
	Name   string
	Bottom float64
	Top    float64
	Read   string
	Depth  number
	Modes  []modeData
}

// absorberData is an absorption by something the cloud and the air hold
// and that is not yet in: so much a metre, between two heights, at
// wavelengths shorter than a limit in nanometres.
type absorberData struct {
	Name       string
	Bottom     float64
	Top        float64
	Below      float64
	Read       string
	Absorption number
}

// tableData is a figure that changes with wavelength, a table of
// wavelength in nanometres and the figure, and either where it was read
// or why it was picked: one of them, and not both.
type tableData struct {
	Read   string
	Picked string
	Points [][2]float64
}

// groundData is the ground's albedo, as a table.
type groundData = tableData

// data is everything venus.yaml holds.
type data struct {
	Planet    planetData
	Gas       gasData
	Particles particleData
	Layers    []layerData
	Absorbers []absorberData
	Ground    groundData
}

// shareTolerance is how far the gas's two shares may be from a whole.
// Picked: rounding in the two shares as printed, and no more.
const shareTolerance = 1e-9

// readData reads the data and checks it, so that making a column from it
// afterwards has nothing to refuse of it.
func readData(raw []byte) (data, error) {
	var read data
	if err := yamlfile.Read("venus.yaml", raw, &read); err != nil {
		return data{}, err
	}
	if err := read.check(); err != nil {
		return data{}, err
	}
	return read, nil
}

// entry is a number, what it is called in a refusal, the unit it must be
// in, and whether it must be above nought rather than nought or more.
type entry struct {
	name     string
	unit     string
	number   number
	positive bool
}

// check is why a number is refused, if it is: a note missing or doubled,
// the wrong unit, a size below what it may be, a pick outside its range.
func (each entry) check() error {
	number := each.number
	switch {
	case number.Read == "" && number.Picked == "":
		return fmt.Errorf("airs: %s has neither a read nor a picked note",
			each.name)
	case number.Read != "" && number.Picked != "":
		return fmt.Errorf("airs: %s has both a read and a picked note",
			each.name)
	case number.Unit != each.unit:
		return fmt.Errorf("airs: %s is in %q, not %q", each.name,
			number.Unit, each.unit)
	case number.Value < 0 || (each.positive && number.Value == 0):
		return fmt.Errorf("airs: %s is %g, which is too small", each.name,
			number.Value)
	case (number.Low != 0 || number.High != 0) &&
		!(number.Low <= number.Value && number.Value <= number.High):
		return fmt.Errorf("airs: %s is %g, outside its range %g to %g",
			each.name, number.Value, number.Low, number.High)
	}
	return nil
}

// numbers is every number in the data that carries a note.
func (read data) numbers() []entry {
	entries := []entry{
		{"planet radius", "m", read.Planet.Radius, true},
		{"planet gravity", "m s^-2", read.Planet.Gravity, true},
		{"planet surface pressure", "bar", read.Planet.SurfacePressure, true},
		{"planet distance", "au", read.Planet.Distance, true},
		{"gas carbon dioxide share", "fraction by volume", read.Gas.CarbonDioxide, true},
		{"gas nitrogen share", "fraction by volume", read.Gas.Nitrogen, true},
		{"gas carbon dioxide mass", "kg mol^-1", read.Gas.CarbonDioxideMass, true},
		{"gas nitrogen mass", "kg mol^-1", read.Gas.NitrogenMass, true},
		{"gas surface density", "kg m^-3", read.Gas.SurfaceDensity, true},
		{"gas scale height", "m", read.Gas.ScaleHeight, true},
		{"gas top", "m", read.Gas.Top, true},
		{"gas shell", "m", read.Gas.Shell, true},
		{"gas absorption", "m^-1", read.Gas.Absorption, false},
		{"gas stand-in", "parts a million", read.Gas.StandIn, true},
		{"particle real index", "", read.Particles.Real, true},
		{"particle imaginary index", "", read.Particles.Imaginary, false},
		{"particle reference wavelength", "nm", read.Particles.Reference, true},
	}
	return append(append(entries, read.layerNumbers()...), read.absorberNumbers()...)
}

// layerNumbers are the layers' depths and modes' widths.
func (read data) layerNumbers() []entry {
	var entries []entry
	for _, layer := range read.Layers {
		entries = append(entries, entry{layer.Name + " depth", "", layer.Depth, true})
		for index, mode := range layer.Modes {
			entries = append(entries, entry{
				fmt.Sprintf("%s mode %d width", layer.Name, index+1), "",
				mode.Width, true})
		}
	}
	return entries
}

// absorberNumbers are the absorbers' strengths.
func (read data) absorberNumbers() []entry {
	var entries []entry
	for _, absorber := range read.Absorbers {
		entries = append(entries, entry{absorber.Name + " absorption", "m^-1",
			absorber.Absorption, false})
	}
	return entries
}

// check is why the data is refused, if it is: any number, the shares of
// the gas, its index, the layers, the absorbers or the ground.
func (read data) check() error {
	for _, each := range read.numbers() {
		if err := each.check(); err != nil {
			return err
		}
	}
	shares := read.Gas.CarbonDioxide.Value + read.Gas.Nitrogen.Value
	if math.Abs(shares-1) > shareTolerance {
		return fmt.Errorf("airs: the gas's shares by volume make %g, not "+
			"one", shares)
	}
	for _, check := range []func() error{read.Gas.Refraction.check,
		read.checkLayers, read.checkAbsorbers, read.Ground.check} {
		if err := check(); err != nil {
			return err
		}
	}
	return nil
}

// check is why the index is refused, if it is.
func (index refractionData) check() error {
	switch {
	case index.Read == "" || len(index.Terms) == 0:
		return fmt.Errorf("airs: the carbon dioxide index has no note or " +
			"no terms")
	case !(0 < index.Shortest && index.Shortest < index.Longest):
		return fmt.Errorf("airs: the carbon dioxide index holds from %g to "+
			"%g nm", index.Shortest, index.Longest)
	case !(index.Temperature > 0 && index.Pressure > 0):
		return fmt.Errorf("airs: the carbon dioxide index is for a "+
			"temperature of %g and a pressure of %g", index.Temperature,
			index.Pressure)
	}
	return nil
}

// checkLayers is why the layers are refused, if they are: each noted,
// with a thickness, ranged in depth, with modes; each one on the last;
// the first above the ground and the last at the top of the air.
func (read data) checkLayers() error {
	previous := 0.0
	for _, layer := range read.Layers {
		if err := layer.check(previous); err != nil {
			return err
		}
		previous = layer.Top
	}
	if previous != read.Gas.Top.Value {
		return fmt.Errorf("airs: the layers end at %g, not at the top of the "+
			"air, %g", previous, read.Gas.Top.Value)
	}
	return nil
}

// check is why a layer is refused if it is, given where the last one
// ended, or nought for the first.
func (layer layerData) check(previous float64) error {
	switch {
	case layer.Read == "":
		return fmt.Errorf("airs: %s has no note", layer.Name)
	case !(layer.Top > layer.Bottom):
		return fmt.Errorf("airs: %s has no thickness", layer.Name)
	case previous == 0 && !(layer.Bottom > 0):
		return fmt.Errorf("airs: %s must begin above the ground", layer.Name)
	case previous != 0 && layer.Bottom != previous:
		return fmt.Errorf("airs: %s begins at %g, a gap or an overlap after "+
			"%g", layer.Name, layer.Bottom, previous)
	case !(layer.Depth.Low > 0 && layer.Depth.High > layer.Depth.Low):
		return fmt.Errorf("airs: %s has no range for its depth", layer.Name)
	case len(layer.Modes) == 0:
		return fmt.Errorf("airs: %s has no modes", layer.Name)
	}
	for index, mode := range layer.Modes {
		if err := mode.check(fmt.Sprintf("%s mode %d", layer.Name,
			index+1)); err != nil {
			return err
		}
	}
	return nil
}

// check is why a mode is refused, if it is, called by the name given.
func (mode modeData) check(name string) error {
	switch {
	case mode.Read == "":
		return fmt.Errorf("airs: %s has no note", name)
	case !(mode.Diameter > 0 && mode.Number > 0):
		return fmt.Errorf("airs: %s has a diameter of %g and a number of %g, "+
			"which are not above nought", name, mode.Diameter, mode.Number)
	case !(mode.Width.Value > 1):
		return fmt.Errorf("airs: %s has a width of %g, which is not above "+
			"one", name, mode.Width.Value)
	}
	return nil
}

// checkAbsorbers is why the absorbers are refused, if they are: each
// noted, within the air, and with a wavelength to be shorter than.
func (read data) checkAbsorbers() error {
	for _, absorber := range read.Absorbers {
		if absorber.Read == "" || !(0 <= absorber.Bottom &&
			absorber.Bottom < absorber.Top &&
			absorber.Top <= read.Gas.Top.Value) || !(absorber.Below > 0) {
			return fmt.Errorf("airs: the absorber %q has no note, or is not "+
				"within the air, or has no wavelength limit", absorber.Name)
		}
	}
	return nil
}

// check is why the ground's table is refused, if it is: see checkWithin,
// with an albedo held to nought to one.
func (ground groundData) check() error {
	return ground.checkWithin("the ground's albedo", 0, 1)
}

// checkWithin is why a table, called by the name given, is refused, if it
// is: with no note or both notes, of fewer than two points, not ascending
// in wavelength, or with a figure outside the range from low to high.
func (table tableData) checkWithin(name string, low, high float64) error {
	if (table.Read == "") == (table.Picked == "") || len(table.Points) < 2 {
		return fmt.Errorf("airs: %s has not one note, read or picked, or "+
			"has fewer than two points", name)
	}
	for index, point := range table.Points {
		if !(low <= point[1] && point[1] <= high) ||
			(index > 0 && !(point[0] > table.Points[index-1][0])) {
			return fmt.Errorf("airs: %s point %d, %g at %g nm, is outside "+
				"%g to %g or not after the last", name, index+1, point[1],
				point[0], low, high)
		}
	}
	return nil
}

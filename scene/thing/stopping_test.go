package thing

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/materials"
	"github.com/janearc/slayer/maths"
	"pgregory.net/rapid"
)

// centimetre is a thing's unit of one centimetre, in metres, for the
// tests
const centimetre = 0.01 // defined: a centimetre is a hundredth of a metre

// stoppingFormula is the csv file of the formula a test gives a substance.
const stoppingFormula = "testdata/stopping-formula.csv"

// stoppingBand is the band of a wavelength, in nanometres.
func stoppingBand(nanometres float64) maths.Band {
	return maths.NewBand(nanometres)
}

// filmBand is the band of the film energy's photon, read off lead's own
// coefficient so the test and the material agree on the wavelength
func filmBand(t *testing.T) maths.Band {
	t.Helper()
	lead, err := materials.ByName("lead")
	if err != nil {
		t.Fatal(err)
	}
	stop, there := lead.Attenuations().At(materials.FilmEnergy)
	if !there {
		t.Fatal("lead has no coefficient at the film energy")
	}
	return stoppingBand(stop.Wavelength())
}

// stoppingSoftXray is the band of a soft x-ray.
func stoppingSoftXray(t *testing.T) maths.Band {
	t.Helper()
	return stoppingBand(picked(t, "stopping-soft-xray-nanometres"))
}

// stoppingOthers is bands that are not the film's own, by what each is.
func stoppingOthers(t *testing.T, film maths.Band) map[string]maths.Band {
	t.Helper()
	little := picked(t, "stopping-little-off-factor")
	hair := picked(t, "stopping-hair-off-offset")
	gamma := picked(t, "stopping-gamma-nanometres")
	radio := picked(t, "stopping-radio-nanometres")
	return map[string]maths.Band{
		"a soft x-ray":          stoppingSoftXray(t),
		"a gamma ray":           stoppingBand(gamma),
		"a radio wave":          stoppingBand(radio),
		"a photon a little off": stoppingBand(film.Nanometres * little),
		"a photon a hair off":   stoppingBand(film.Nanometres * (unity + hair)),
		"every band at once":    stoppingBand(nought),
	}
}

// One unit of a stopping thing passes what the material's rows say one
// unit of it passes, for each of the x-ray materials: the Radiant adds
// no number of its own.
func TestStoppingPassesWhatTheMaterialsRowsGive(t *testing.T) {
	band := filmBand(t)
	for _, name := range []string{"lead", "aluminium", "water"} {
		kind, err := materials.ByName(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		stops, err := Stopping(kind, centimetre)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		want, err := kind.Passes(materials.FilmEnergy, centimetre)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got := stops.Through(band); got != want {
			t.Errorf("a centimetre of %s passes %v; its rows give %v",
				name, got, want)
		}
	}
}

// the anti-test of the one above: a band no row was read at is stopped
// whole, however near, and a stopping thing gives off and sends back
// nothing in any band
func TestStoppingStopsEveryBandItHasNoRowAt(t *testing.T) {
	water, err := materials.ByName("water")
	if err != nil {
		t.Fatal(err)
	}
	stops, err := Stopping(water, centimetre)
	if err != nil {
		t.Fatal(err)
	}
	film := filmBand(t)
	for name, band := range stoppingOthers(t, film) {
		if share := stops.Through(band); share != nought {
			t.Errorf("%s: water passes %v with no row read there", name, share)
		}
	}
	for name, band := range map[string]maths.Band{
		"the film's band": film, "a soft x-ray": stoppingSoftXray(t)} {
		if gives := stops.Emits(band); gives != nought {
			t.Errorf("%s: water gives off %v", name, gives)
		}
		if back := stops.Scatters(band); back != nought {
			t.Errorf("%s: water sends back %v", name, back)
		}
	}
	if stops.Through(film) == nought {
		t.Error("water stops the film's own band whole")
	}
}

// A band within rounding of the row's photon is that photon: an energy
// turned to a wavelength, to an energy, and back is still answered.
// The anti-test: a band a thousandth off is another photon, which no
// row was read at, and is stopped whole.
func TestStoppingAnswersTheRowsPhotonWithinRounding(t *testing.T) {
	water, err := materials.ByName("water")
	if err != nil {
		t.Fatal(err)
	}
	stops, err := Stopping(water, centimetre)
	if err != nil {
		t.Fatal(err)
	}
	film := filmBand(t)
	rounding := picked(t, "stopping-rounding-offset")
	near := stoppingBand(film.Nanometres * (unity + rounding))
	if stops.Through(near) != stops.Through(film) {
		t.Errorf("a band a part in a million million off passes %v, not %v",
			stops.Through(near), stops.Through(film))
	}
	thousandth := picked(t, "stopping-thousandth-offset")
	far := stoppingBand(film.Nanometres * (unity + thousandth))
	if got := stops.Through(far); got != nought ||
		!(stops.Through(film) > nought) {
		t.Errorf("a band a thousandth off passes %v, and the film's own %v",
			got, stops.Through(film))
	}
}

// stoppingRefusesNamed checks that Stopping refuses a material, and that
// its refusal names the material.
func stoppingRefusesNamed(
	t *testing.T, kind materials.Material, name, why string) {
	t.Helper()
	_, err := Stopping(kind, centimetre)
	if err == nil {
		t.Fatal(why)
	}
	if !strings.Contains(err.Error(), name) {
		t.Errorf("the refusal does not name the material: %v", err)
	}
}

// stoppingSteam is a substance with water's coefficients and no density.
func stoppingSteam(t *testing.T, water materials.Material) materials.Material {
	t.Helper()
	row := readSheet(t, stoppingFormula).row(t, stoppingFormula, "steam")
	h2o, err := materials.NewFormula(map[string]float64{
		"H": cell(t, row, "H"), "O": cell(t, row, "O")})
	if err != nil {
		t.Fatal(err)
	}
	steam, err := materials.NewSubstance("steam", h2o)
	if err != nil {
		t.Fatal(err)
	}
	steam, err = steam.Attenuating(water.Attenuations())
	if err != nil {
		t.Fatal(err)
	}
	return steam
}

// the refusals of Stopping, each by the material's name: no
// coefficients, no density, and a unit that is no length
func TestStoppingRefusesWhatItCannotDerive(t *testing.T) {
	gold, err := materials.ByName("gold")
	if err != nil {
		t.Fatal(err)
	}
	stoppingRefusesNamed(t, gold, "gold",
		"gold, with no coefficients cited, stops a beam")

	water, err := materials.ByName("water")
	if err != nil {
		t.Fatal(err)
	}
	// coefficients and no density
	stoppingRefusesNamed(t, stoppingSteam(t, water), "steam",
		"steam, with no density cited, stops a beam")

	for name, unit := range map[string]float64{
		"nought":        nought,
		"below nought":  -centimetre,
		"NaN":           math.NaN(),
		"past numbers":  math.Inf(unity),
		"below numbers": math.Inf(-unity),
	} {
		_, err := Stopping(water, unit)
		if err == nil {
			t.Errorf("a unit of %s: water stops a beam", name)
			continue
		}
		if !strings.Contains(err.Error(), "water") {
			t.Errorf("a unit of %s: the refusal does not name water: %v",
				name, err)
		}
	}
}

// stoppingLaw is what the scaling test holds fixed: the band, the material
// and the ranges and tolerances it draws and judges by.
type stoppingLaw struct {
	band            maths.Band
	aluminium       materials.Material
	shortest        float64
	longest         float64
	way             float64
	squareTolerance float64
	wayTolerance    float64
}

// newStoppingLaw is the law the scaling test holds, as the test's data
// gives it.
func newStoppingLaw(t *testing.T) stoppingLaw {
	t.Helper()
	aluminium, err := materials.ByName("aluminium")
	if err != nil {
		t.Fatal(err)
	}
	return stoppingLaw{
		band:            filmBand(t),
		aluminium:       aluminium,
		shortest:        picked(t, "stopping-unit-shortest"),
		longest:         picked(t, "stopping-unit-longest"),
		way:             picked(t, "stopping-way-longest"),
		squareTolerance: picked(t, "stopping-square-tolerance"),
		wayTolerance:    picked(t, "stopping-way-tolerance"),
	}
}

// through is the share one unit of aluminium passes of the law's band.
func (law stoppingLaw) through(draw *rapid.T, unit float64) float64 {
	stops, err := Stopping(law.aluminium, unit)
	if err != nil {
		draw.Fatal(err)
	}
	return stops.Through(law.band)
}

// check is the law for one draw of a unit and a way through.
func (law stoppingLaw) check(draw *rapid.T) {
	unit := rapid.Float64Range(law.shortest, law.longest).Draw(draw, "unit")
	way := rapid.Float64Range(nought, law.way).Draw(draw, "way in metres")

	one := law.through(draw, unit)
	two := law.through(draw, unit*pair)
	if math.Abs(two-one*one) > law.squareTolerance {
		draw.Fatalf("twice the unit passes %v, not the square of %v",
			two, one)
	}
	if !(one > nought && one < unity) {
		draw.Fatalf("a unit of %v m of aluminium passes %v", unit, one)
	}
	law.counted(draw, unit, way, one, two)
}

// counted checks that a way through, counted in either unit, passes what
// the material's rows give.
func (law stoppingLaw) counted(
	draw *rapid.T, unit, way, one, two float64) {
	byShort := math.Pow(one, way/unit)
	byLong := math.Pow(two, way/(unit*pair))
	if math.Abs(byShort-byLong) > law.wayTolerance {
		draw.Fatalf("%v m passes %v counted in short units, %v in long",
			way, byShort, byLong)
	}
	want, err := law.aluminium.Passes(materials.FilmEnergy, way)
	if err != nil {
		draw.Fatal(err)
	}
	if math.Abs(byShort-want) > law.wayTolerance {
		draw.Fatalf("%v m passes %v by the tracer's sum; the rows give %v",
			way, byShort, want)
	}
}

// The tracer raises one unit's share to the power of the way through,
// so the share must scale as Beer and Lambert's law does: a thing whose
// unit is twice as long passes, a unit, the square of what the shorter
// unit passes. The anti-test is its other half: the way through in
// metres decides the share, and the unit it is counted in does not.
func TestStoppingScalesAsBeerAndLamberts(t *testing.T) {
	law := newStoppingLaw(t)
	rapid.Check(t, law.check)
}

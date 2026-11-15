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
const centimetre = 0.01

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
	return maths.Band{Nanometres: stop.Wavelength()}
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
			t.Errorf("a centimetre of %s passes %v of the band; its rows give %v",
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
	others := map[string]maths.Band{
		"a soft x-ray":          {Nanometres: 1},
		"a gamma ray":           {Nanometres: 1e-4},
		"a radio wave":          {Nanometres: 1.26e8},
		"a photon a little off": {Nanometres: film.Nanometres * 1.001},
		"a photon a hair off":   {Nanometres: film.Nanometres * (1 + 1e-6)},
		"every band at once":    {},
	}
	for name, band := range others {
		if share := stops.Through(band); share != 0 {
			t.Errorf("%s: water passes %v with no row read there", name, share)
		}
	}
	for name, band := range map[string]maths.Band{
		"the film's band": film, "a soft x-ray": {Nanometres: 1}} {
		if gives := stops.Emits(band); gives != 0 {
			t.Errorf("%s: water gives off %v", name, gives)
		}
		if back := stops.Scatters(band); back != 0 {
			t.Errorf("%s: water sends back %v", name, back)
		}
	}
	if stops.Through(film) == 0 {
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
	near := maths.Band{Nanometres: film.Nanometres * (1 + 1e-12)}
	if stops.Through(near) != stops.Through(film) {
		t.Errorf("a band a part in a million million off passes %v, not %v",
			stops.Through(near), stops.Through(film))
	}
	far := maths.Band{Nanometres: film.Nanometres * (1 + 1e-3)}
	if got := stops.Through(far); got != 0 || !(stops.Through(film) > 0) {
		t.Errorf("a band a thousandth off passes %v, and the film's own %v",
			got, stops.Through(film))
	}
}

// the refusals of Stopping, each by the material's name: no
// coefficients, no density, and a unit that is no length
func TestStoppingRefusesWhatItCannotDerive(t *testing.T) {
	gold, err := materials.ByName("gold")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Stopping(gold, centimetre)
	if err == nil {
		t.Fatal("gold, with no coefficients cited, stops a beam")
	}
	if !strings.Contains(err.Error(), "gold") {
		t.Errorf("the refusal does not name the material: %v", err)
	}

	// coefficients and no density
	h2o, err := materials.NewFormula(map[string]float64{"H": 2, "O": 1})
	if err != nil {
		t.Fatal(err)
	}
	steam, err := materials.NewSubstance("steam", h2o)
	if err != nil {
		t.Fatal(err)
	}
	water, err := materials.ByName("water")
	if err != nil {
		t.Fatal(err)
	}
	steam, err = steam.Attenuating(water.Attenuations())
	if err != nil {
		t.Fatal(err)
	}
	_, err = Stopping(steam, centimetre)
	if err == nil {
		t.Fatal("steam, with no density cited, stops a beam")
	}
	if !strings.Contains(err.Error(), "steam") {
		t.Errorf("the refusal does not name the material: %v", err)
	}

	for name, unit := range map[string]float64{
		"nought":        0,
		"below nought":  -centimetre,
		"NaN":           math.NaN(),
		"past numbers":  math.Inf(1),
		"below numbers": math.Inf(-1),
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

// The tracer raises one unit's share to the power of the way through,
// so the share must scale as Beer and Lambert's law does: a thing whose
// unit is twice as long passes, a unit, the square of what the shorter
// unit passes. The anti-test is its other half: the way through in
// metres decides the share, and the unit it is counted in does not.
func TestStoppingScalesAsBeerAndLamberts(t *testing.T) {
	band := filmBand(t)
	aluminium, err := materials.ByName("aluminium")
	if err != nil {
		t.Fatal(err)
	}
	rapid.Check(t, func(draw *rapid.T) {
		unit := rapid.Float64Range(1e-4, 0.05).Draw(draw, "unit")
		way := rapid.Float64Range(0, 0.1).Draw(draw, "way in metres")

		short, err := Stopping(aluminium, unit)
		if err != nil {
			draw.Fatal(err)
		}
		long, err := Stopping(aluminium, unit*2)
		if err != nil {
			draw.Fatal(err)
		}
		one, two := short.Through(band), long.Through(band)
		if math.Abs(two-one*one) > 1e-12 {
			draw.Fatalf("twice the unit passes %v, not the square of %v",
				two, one)
		}
		if !(one > 0 && one < 1) {
			draw.Fatalf("a unit of %v m of aluminium passes %v", unit, one)
		}
		// the same way through, counted in either unit
		byShort := math.Pow(one, way/unit)
		byLong := math.Pow(two, way/(unit*2))
		if math.Abs(byShort-byLong) > 1e-9 {
			draw.Fatalf("%v m passes %v counted in short units, %v in long",
				way, byShort, byLong)
		}
		want, err := aluminium.Passes(materials.FilmEnergy, way)
		if err != nil {
			draw.Fatal(err)
		}
		if math.Abs(byShort-want) > 1e-9 {
			draw.Fatalf("%v m passes %v by the tracer's sum; the rows give %v",
				way, byShort, want)
		}
	})
}

package materials

import (
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/janearc/celestial/radiation"
)

// numbers are the cells of a row that are numbers, in the row's order,
// each read as the page writes it
func numbers(t *testing.T, row string) []float64 {
	t.Helper()
	var read []float64
	for cell := range strings.FieldsSeq(row) {
		if number, err := strconv.ParseFloat(cell, 64); err == nil {
			read = append(read, number)
		}
	}
	return read
}

// xrayRows is what each x-ray material was read from: its own page, the
// table its density is in, and the density's cell as the table writes
// it
var xrayRows = map[string]struct {
	page, table, cell string
}{
	"aluminium": {"ElemTab/z13.html", "tab1.html", "2.699E+00"},
	"lead":      {"ElemTab/z82.html", "tab1.html", "1.135E+01"},
	"water":     {"ComTab/water.html", "tab2.html", "1.000E+00"},
}

// Every x-ray material's stored numbers are the ones in the row it
// quotes: the energy and the coefficient are the row's first two
// numbers, exactly, and the density is the cell the table writes. That a
// row is quoted right against its page no test checks: celestial-data's
// nistxray adapter holds every page to its sum, which guards the page
// and not the copying. The rows were compared with the pages by hand,
// by the writer and again by a reviewer on 2026-11-12.
func TestTheXRayMaterialsAreTheirQuotedRows(t *testing.T) {
	for name, from := range xrayRows {
		kind, err := ByName(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		stop, there := kind.Attenuations().At(FilmEnergy)
		if !there {
			t.Errorf("%s has no coefficient at the film energy", name)
			continue
		}
		row := numbers(t, stop.Where().Row())
		if len(row) != 3 {
			t.Errorf("%s: its row %q is not three numbers",
				name, stop.Where().Row())
			continue
		}
		if stop.Energy() != row[0] {
			t.Errorf("%s: energy %v, its row has %v",
				name, stop.Energy(), row[0])
		}
		if stop.MuOverRho() != row[1] {
			t.Errorf("%s: mu over rho %v, its row has %v",
				name, stop.MuOverRho(), row[1])
		}
		if stop.Where().Release() != nist ||
			stop.Where().Paper() != hubbell {
			t.Errorf("%s: its coefficient is cited to %q, %q", name,
				stop.Where().Release(), stop.Where().Paper())
		}
		if !strings.Contains(stop.Where().Material(), from.page) {
			t.Errorf("%s: its coefficient is from %q, not %s",
				name, stop.Where().Material(), from.page)
		}

		dense := kind.DensityFrom()
		if !strings.Contains(dense.Row(), from.cell) {
			t.Errorf("%s: its density row %q has no cell %s",
				name, dense.Row(), from.cell)
		}
		cell, err := strconv.ParseFloat(from.cell, 64)
		if err != nil {
			t.Fatal(err)
		}
		if kind.Density() != cell {
			t.Errorf("%s: density %v, the table's cell is %v",
				name, kind.Density(), cell)
		}
		if dense.Release() != nist || dense.Paper() != hubbell {
			t.Errorf("%s: its density is cited to %q, %q", name,
				dense.Release(), dense.Paper())
		}
		if !strings.Contains(dense.Material(), from.table) {
			t.Errorf("%s: its density is from %q, not %s",
				name, dense.Material(), from.table)
		}
	}
}

// the anti-test of the one above: no x-ray material carries another's
// row, and none has a coefficient at an energy its row does not give
func TestTheXRayMaterialsAreNotOneAnothers(t *testing.T) {
	seen := map[string]string{}
	for name := range xrayRows {
		kind, err := ByName(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		stop, _ := kind.Attenuations().At(FilmEnergy)
		for _, row := range []string{stop.Where().Row(),
			kind.DensityFrom().Row()} {
			if other, taken := seen[row]; taken {
				t.Errorf("%s and %s quote one row: %q", name, other, row)
			}
			seen[row] = name
		}
		for _, energy := range []float64{0.05, 0.08, 60} {
			if got, there := kind.Attenuations().At(energy); there {
				t.Errorf("%s answers %v at %v MeV, an energy not quoted",
					name, got, energy)
			}
		}
	}
}

// The x-ray smoke's point, from the rows alone: through a centimetre at
// the film energy, lead stops the beam, aluminium passes part of it and
// water barely touches it. Each share is e to the minus mu over rho,
// times the density, times the centimetre, with the rows' own numbers.
func TestLeadStopsWhatAluminiumPartlyPassesAndWaterBarelyTouches(t *testing.T) {
	const centimetre = 0.01
	shares := map[string]float64{}
	for name := range xrayRows {
		kind, err := ByName(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		share, err := kind.Passes(FilmEnergy, centimetre)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		shares[name] = share
	}
	want := map[string]float64{
		"lead":      math.Exp(-5.021 * 11.35),
		"aluminium": math.Exp(-0.2778 * 2.699),
		"water":     math.Exp(-0.2059 * 1.000),
	}
	for name, share := range shares {
		if math.Abs(share-want[name]) > 1e-12 {
			t.Errorf("%s passes %v of the beam, its rows give %v",
				name, share, want[name])
		}
	}
	if !(shares["lead"] < 1e-20) {
		t.Errorf("a centimetre of lead passes %v: it does not stop the beam",
			shares["lead"])
	}
	if !(shares["aluminium"] > 0.4 && shares["aluminium"] < 0.55) {
		t.Errorf("a centimetre of aluminium passes %v, not about half",
			shares["aluminium"])
	}
	if !(shares["water"] > 0.8) {
		t.Errorf("a centimetre of water passes %v: it more than touches the beam",
			shares["water"])
	}
}

// the anti-test of the one above: the order is the materials', not the
// thickness's. A hundredth of the lead still passes less than a
// centimetre of water, and no material passes everything.
func TestTheXRayOrderIsTheMaterials(t *testing.T) {
	lead, err := ByName("lead")
	if err != nil {
		t.Fatal(err)
	}
	water, err := ByName("water")
	if err != nil {
		t.Fatal(err)
	}
	thin, err := lead.Passes(FilmEnergy, 0.001)
	if err != nil {
		t.Fatal(err)
	}
	thick, err := water.Passes(FilmEnergy, 0.01)
	if err != nil {
		t.Fatal(err)
	}
	if !(thin < thick) {
		t.Errorf("a millimetre of lead passes %v, a centimetre of water %v",
			thin, thick)
	}
	for name := range xrayRows {
		kind, err := ByName(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		share, err := kind.Passes(FilmEnergy, 0.0001)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !(share < 1) {
			t.Errorf("a tenth of a millimetre of %s passes everything", name)
		}
	}
}

// The film energy is an x-ray by celestial's ranges, and about 0.021 nm
// as its comment says.
func TestTheFilmEnergyIsAnXRay(t *testing.T) {
	lead, err := ByName("lead")
	if err != nil {
		t.Fatal(err)
	}
	stop, there := lead.Attenuations().At(FilmEnergy)
	if !there {
		t.Fatal("lead has no coefficient at the film energy")
	}
	if !radiation.XRay().Holds(stop.Wavelength()) {
		t.Errorf("the film energy, %v nm, is not held an x-ray",
			stop.Wavelength())
	}
	if math.Abs(stop.Wavelength()-0.021) > 0.0005 {
		t.Errorf("the film energy is %v nm, not about 0.021",
			stop.Wavelength())
	}
	if held := lead.Attenuations().Within(radiation.XRay()); len(held) != 1 {
		t.Errorf("lead's x-ray rows are %v, not its one", held)
	}
}

// the anti-test of the one above: it is not a gamma ray, and lead has
// no row there
func TestTheFilmEnergyIsNotAGammaRay(t *testing.T) {
	lead, err := ByName("lead")
	if err != nil {
		t.Fatal(err)
	}
	stop, _ := lead.Attenuations().At(FilmEnergy)
	if radiation.Gamma().Holds(stop.Wavelength()) {
		t.Errorf("the film energy, %v nm, is held a gamma ray",
			stop.Wavelength())
	}
	if held := lead.Attenuations().Within(radiation.Gamma()); len(held) != 0 {
		t.Errorf("lead answers gamma rows it was never given: %v", held)
	}
}

// Lead's atomic weight is its number, 82, over the ratio of number to
// mass in its row of NIST's Table 1, to the four figures kept.
func TestLeadsWeightIsItsNumberOverNISTsRatio(t *testing.T) {
	lead, err := ByName("lead")
	if err != nil {
		t.Fatal(err)
	}
	row := numbers(t, lead.DensityFrom().Row())
	if len(row) != 4 {
		t.Fatalf("lead's row %q is not four numbers",
			lead.DensityFrom().Row())
	}
	number, ratio := row[0], row[1]
	if number != 82 {
		t.Errorf("lead's row gives it the number %v", number)
	}
	if math.Abs(number/ratio-weights["Pb"]) > 0.05 {
		t.Errorf("lead weighs %v here; its row gives %v",
			weights["Pb"], number/ratio)
	}
	pb, err := NewFormula(map[string]float64{"Pb": 1})
	if err != nil {
		t.Fatal(err)
	}
	if pb.Mass() != weights["Pb"] {
		t.Errorf("a formula of one lead weighs %v, not %v",
			pb.Mass(), weights["Pb"])
	}
}

// the anti-test of the one above: lead is not gold, the heavy element
// that was here before it
func TestLeadsWeightIsNotAnothers(t *testing.T) {
	for symbol, weight := range weights {
		if symbol != "Pb" && weight == weights["Pb"] {
			t.Errorf("%s weighs what lead does, %v", symbol, weight)
		}
	}
	if !(weights["Pb"] > weights["Au"] && weights["Pb"] < weights["Th"]) {
		t.Errorf("lead, %v, does not sit between gold and thorium",
			weights["Pb"])
	}
}

// Lead and water say what they are and what they take from a beam, and
// carry no optics, for none is cited; aluminium keeps the optics it had
// before it was given its x-ray row.
func TestLeadAndWaterCarryNoOpticsAndAluminiumKeepsIts(t *testing.T) {
	for _, name := range []string{"lead", "water"} {
		kind, err := ByName(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if kind.Name() != name {
			t.Errorf("%s answers to the name %s", name, kind.Name())
		}
		for _, band := range []Band{Red, Green, Blue} {
			if kind.Optics().Has(band) {
				t.Errorf("%s has %s optics with none cited", name, band)
			}
		}
	}
	aluminium, err := ByName("aluminium")
	if err != nil {
		t.Fatal(err)
	}
	for _, band := range []Band{Red, Green, Blue} {
		if !aluminium.Optics().Has(band) {
			t.Errorf("aluminium lost its %s reading", band)
		}
	}
	if blue := aluminium.Optics().At(Blue).N(); blue != 0.64439 {
		t.Errorf("aluminium's blue n is %v, not the published 0.64439", blue)
	}
}

// the anti-test of the one above: the metals with no x-ray row cited
// carry none, and answer no share
func TestTheOtherMetalsCarryNoXRayRow(t *testing.T) {
	for _, name := range []string{"gold", "silver", "beryllium", "titanium"} {
		kind, err := ByName(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if _, there := kind.Attenuations().At(FilmEnergy); there {
			t.Errorf("%s has an x-ray row with none cited", name)
		}
		if kind.DensityFrom() != (Source{}) {
			t.Errorf("%s has a density source with none cited", name)
		}
		_, err = kind.Passes(FilmEnergy, 0.01)
		if err == nil {
			t.Errorf("%s passed a share of a beam with no row cited", name)
			continue
		}
		if !strings.Contains(err.Error(), name) {
			t.Errorf("%s: the refusal does not name it: %v", name, err)
		}
	}
}

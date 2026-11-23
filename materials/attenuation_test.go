package materials

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/celestial/radiation"
	"pgregory.net/rapid"
)

// stopping is a coefficient at an energy, read somewhere, for the tests
func stopping(t *testing.T, energy, muOverRho float64) Attenuation {
	t.Helper()
	stop, err := newAttenuation(energy, muOverRho, somewhere(t))
	if err != nil {
		t.Fatal(err)
	}
	return stop
}

// slab is iron with one coefficient at 60 keV, for the tests of what
// passes: the coefficient and the density are the tests' own, not a
// table's
func slab(t *testing.T, muOverRho, density float64) Material {
	t.Helper()
	stops, err := NewAttenuations(stopping(t, 0.06, muOverRho))
	if err != nil {
		t.Fatal(err)
	}
	kind, err := iron(t).atDensity(density, somewhere(t))
	if err != nil {
		t.Fatal(err)
	}
	kind, err = kind.Attenuating(stops)
	if err != nil {
		t.Fatal(err)
	}
	return kind
}

func TestAnAttenuationAnswersWhatItWasMadeOf(t *testing.T) {
	where := somewhere(t)
	stop, err := newAttenuation(0.06, 5.021, where)
	if err != nil {
		t.Fatal(err)
	}
	if stop.Energy() != 0.06 {
		t.Errorf("an energy of %v MeV, made at 0.06", stop.Energy())
	}
	if stop.MuOverRho() != 5.021 {
		t.Errorf("a coefficient of %v, made at 5.021", stop.MuOverRho())
	}
	if stop.Where() != where {
		t.Errorf("read from %v, made from %v", stop.Where(), where)
	}
}

// the anti-test of the one above: nothing nonsensical is made
func TestAnAttenuationRefusesNonsense(t *testing.T) {
	where := somewhere(t)
	refused := map[string][2]float64{
		"an energy of nought":         {0, 1},
		"an energy below nought":      {-0.06, 1},
		"an energy of NaN":            {math.NaN(), 1},
		"an energy past numbers":      {math.Inf(1), 1},
		"a coefficient of nought":     {0.06, 0},
		"a coefficient below nought":  {0.06, -5},
		"a coefficient of NaN":        {0.06, math.NaN()},
		"a coefficient past numbers":  {0.06, math.Inf(1)},
		"a coefficient below numbers": {0.06, math.Inf(-1)},
	}
	for name, bad := range refused {
		if _, err := newAttenuation(bad[0], bad[1], where); err == nil {
			t.Errorf("%s: an attenuation was made", name)
		}
	}
	if _, err := newAttenuation(0.06, 1, Source{}); err == nil {
		t.Error("an attenuation read nowhere was made")
	}
}

// A photon's wavelength comes from its energy through celestial, in
// electronvolts: turned back into an energy it is 60 keV again, and it
// is an x-ray.
func TestAnAttenuationsWavelengthIsItsEnergys(t *testing.T) {
	stop := stopping(t, 0.06, 1)
	back := radiation.Energy(stop.Wavelength())
	if math.Abs(back-60000) > 1e-6 {
		t.Errorf("the wavelength %v nm is a photon of %v eV, not 60000",
			stop.Wavelength(), back)
	}
	if !radiation.XRay().Holds(stop.Wavelength()) {
		t.Errorf("a 60 keV photon, %v nm, is not held an x-ray",
			stop.Wavelength())
	}
}

// the anti-test of the one above: the energy was not left in MeV, which
// would make a 60 keV photon tens of micrometres long, nor is the
// photon a gamma ray
func TestAnAttenuationsWavelengthIsNotItsEnergyInMeV(t *testing.T) {
	stop := stopping(t, 0.06, 1)
	if stop.Wavelength() == radiation.FromEnergy(0.06) {
		t.Errorf("the wavelength %v nm is 0.06 eV's, not 0.06 MeV's",
			stop.Wavelength())
	}
	if radiation.Gamma().Holds(stop.Wavelength()) {
		t.Errorf("a 60 keV photon, %v nm, is held a gamma ray",
			stop.Wavelength())
	}
	// a longer photon carries less: ten times the energy, a tenth the
	// wavelength
	hard := stopping(t, 0.6, 1)
	if !(hard.Wavelength() < stop.Wavelength()) {
		t.Errorf("600 keV is %v nm, no shorter than 60 keV's %v nm",
			hard.Wavelength(), stop.Wavelength())
	}
}

func TestAttenuationsAnswerAtAnEnergyAndWithinARange(t *testing.T) {
	soft, film := stopping(t, 0.01, 26.23), stopping(t, 0.06, 0.2778)
	stops, err := NewAttenuations(soft, film)
	if err != nil {
		t.Fatal(err)
	}
	got, there := stops.At(0.06)
	if !there || got != film {
		t.Errorf("at 0.06 MeV: %v, %v; the film row was given", got, there)
	}
	got, there = stops.At(0.01)
	if !there || got != soft {
		t.Errorf("at 0.01 MeV: %v, %v; the soft row was given", got, there)
	}
	held := stops.Within(radiation.XRay())
	if len(held) != 2 || held[0] != soft || held[1] != film {
		t.Errorf("the x-ray range holds %v, not both rows in order", held)
	}
}

// the anti-test of the one above: an energy no row was given at is not
// answered, nothing is interpolated between two rows, and a range that
// holds no row answers none
func TestAttenuationsAnswerNothingTheyWereNotGiven(t *testing.T) {
	stops, err := NewAttenuations(stopping(t, 0.01, 26.23),
		stopping(t, 0.06, 0.2778))
	if err != nil {
		t.Fatal(err)
	}
	for _, energy := range []float64{0.03, 0.0600001, 0, 60} {
		if got, there := stops.At(energy); there {
			t.Errorf("at %v MeV a row was answered: %v", energy, got)
		}
	}
	if held := stops.Within(radiation.Gamma()); len(held) != 0 {
		t.Errorf("the gamma range holds %v; no row is a gamma ray", held)
	}
	seen, err := radiation.NewRange(380, 750)
	if err != nil {
		t.Fatal(err)
	}
	if held := stops.Within(*seen); len(held) != 0 {
		t.Errorf("the light we see holds %v; no row is light", held)
	}
	// coefficients never made answer nothing at all
	var none Attenuations
	if got, there := none.At(0.06); there {
		t.Errorf("coefficients never made answered %v", got)
	}
	if held := none.Within(radiation.XRay()); len(held) != 0 {
		t.Errorf("coefficients never made hold %v", held)
	}
}

// the refusals of NewAttenuations: none, one never made, two at one
// energy
func TestAttenuationsRefuseNoneNeverMadeAndTwoAtOneEnergy(t *testing.T) {
	if _, err := NewAttenuations(); err == nil {
		t.Error("attenuations of no readings were made")
	}
	if _, err := NewAttenuations(Attenuation{}); err == nil {
		t.Error("attenuations of a reading never made were made")
	}
	film := stopping(t, 0.06, 0.2778)
	if _, err := NewAttenuations(film, Attenuation{}); err == nil {
		t.Error("attenuations with a reading never made were made")
	}
	_, err := NewAttenuations(film, stopping(t, 0.06, 5.021))
	if err == nil {
		t.Fatal("two attenuations at one energy were made")
	}
	if !strings.Contains(err.Error(), "0.06") {
		t.Errorf("the refusal does not name the energy: %v", err)
	}
}

// What NewAttenuations was given is its own afterwards: changing the
// caller's rows does not change what was made.
func TestAttenuationsKeepTheirOwnRows(t *testing.T) {
	rows := []Attenuation{stopping(t, 0.01, 26.23),
		stopping(t, 0.06, 0.2778)}
	stops, err := NewAttenuations(rows...)
	if err != nil {
		t.Fatal(err)
	}
	rows[1] = stopping(t, 0.08, 0.2018)
	if _, there := stops.At(0.06); !there {
		t.Error("the row at 0.06 MeV went when the caller's row changed")
	}
	if got, there := stops.At(0.08); there {
		t.Errorf("a row the caller wrote afterwards is answered: %v", got)
	}
}

func TestAllIsEveryRowInTheOrderGiven(t *testing.T) {
	soft, film := stopping(t, 0.01, 26.23), stopping(t, 0.06, 0.2778)
	stops, err := NewAttenuations(film, soft)
	if err != nil {
		t.Fatal(err)
	}
	all := stops.All()
	if len(all) != 2 || all[0] != film || all[1] != soft {
		t.Errorf("all the rows are %v, not the two in the order given", all)
	}
}

// the anti-test of the one above: what All hands out is the caller's to
// change without changing the coefficients, and coefficients never made
// have none
func TestAllHandsOutACopy(t *testing.T) {
	stops, err := NewAttenuations(stopping(t, 0.06, 0.2778))
	if err != nil {
		t.Fatal(err)
	}
	stops.All()[0] = stopping(t, 0.08, 0.2018)
	if _, there := stops.At(0.06); !there {
		t.Error("writing to what All handed out changed the coefficients")
	}
	var none Attenuations
	if all := none.All(); len(all) != 0 {
		t.Errorf("coefficients never made have rows: %v", all)
	}
}

func TestAMaterialAtADensityCarriesItAndItsSource(t *testing.T) {
	where, err := newSource("a table", "iron, a page", "a paper",
		"26 Fe Iron 7.874E+00")
	if err != nil {
		t.Fatal(err)
	}
	fe, err := NewFormula(map[string]float64{"Fe": 1})
	if err != nil {
		t.Fatal(err)
	}
	bare, err := NewSubstance("iron", fe)
	if err != nil {
		t.Fatal(err)
	}
	dense, err := bare.atDensity(7.874, where)
	if err != nil {
		t.Fatal(err)
	}
	if dense.Density() != 7.874 {
		t.Errorf("a density of %v, given 7.874", dense.Density())
	}
	if dense.DensityFrom() != where {
		t.Errorf("the density read from %v, given %v",
			dense.DensityFrom(), where)
	}
	if dense.Name() != "iron" {
		t.Errorf("the material answers to %q, not iron", dense.Name())
	}
	// the material it was made from is as it was
	if bare.Density() != 0 || bare.DensityFrom() != (Source{}) {
		t.Errorf("the material given a density changed: %v from %v",
			bare.Density(), bare.DensityFrom())
	}
}

// the anti-test of the one above: no density without a number and a
// source, and a density given with no source says it has none
func TestAtDensityRefusesNonsense(t *testing.T) {
	where := somewhere(t)
	for name, density := range map[string]float64{
		"nought":       0,
		"below nought": -7.874,
		"NaN":          math.NaN(),
		"past numbers": math.Inf(1),
	} {
		if _, err := iron(t).atDensity(density, where); err == nil {
			t.Errorf("a density of %s was given", name)
		}
	}
	_, err := iron(t).atDensity(7.874, Source{})
	if err == nil {
		t.Fatal("a density read nowhere was given")
	}
	if !strings.Contains(err.Error(), "iron") {
		t.Errorf("the refusal does not name the material: %v", err)
	}
	if _, err := (Material{}).atDensity(7.874, where); err == nil {
		t.Error("a material never made was given a density")
	}
	// iron of the tests was made at a density with no source
	if from := iron(t).DensityFrom(); from != (Source{}) {
		t.Errorf("a density given with no source says it is from %v", from)
	}
}

func TestAMaterialCarriesItsAttenuations(t *testing.T) {
	film := stopping(t, 0.06, 0.2778)
	stops, err := NewAttenuations(film)
	if err != nil {
		t.Fatal(err)
	}
	kind, err := iron(t).Attenuating(stops)
	if err != nil {
		t.Fatal(err)
	}
	if got, there := kind.Attenuations().At(0.06); !there || got != film {
		t.Errorf("the material answers %v, %v at 0.06 MeV", got, there)
	}
	if kind.Density() != 7.87 {
		t.Errorf("its density went to %v when it was given coefficients",
			kind.Density())
	}
}

// the anti-test of the one above: no coefficients, none carried
func TestAttenuatingRefusesNothing(t *testing.T) {
	_, err := iron(t).Attenuating(Attenuations{})
	if err == nil {
		t.Fatal("a material was given attenuations never made")
	}
	if !strings.Contains(err.Error(), "iron") {
		t.Errorf("the refusal does not name the material: %v", err)
	}
	stops, err := NewAttenuations(stopping(t, 0.06, 0.2778))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (Material{}).Attenuating(stops); err == nil {
		t.Error("a material never made was given attenuations")
	}
	if _, there := iron(t).Attenuations().At(0.06); there {
		t.Error("a material given no coefficients answers one")
	}
}

// What passes is e to the minus mu over rho, times the density, times
// the thickness in centimetres: held here to the sum done by hand for
// the tests' own numbers, 2 cm2/g at 5 g/cm3 through 1 mm, which is
// e to the minus one.
func TestWhatPassesIsDerived(t *testing.T) {
	kind := slab(t, 2, 5)
	share, err := kind.Passes(0.06, 0.001)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(share-math.Exp(-1)) > 1e-15 {
		t.Errorf("1 mm passes %v, not e to the minus one, %v",
			share, math.Exp(-1))
	}
	all, err := kind.Passes(0.06, 0)
	if err != nil {
		t.Fatal(err)
	}
	if all != 1 {
		t.Errorf("no thickness at all passes %v, not everything", all)
	}
}

// the anti-test of the one above: a thickness in metres is not read as
// centimetres, twice the density is not the same share, and what cannot
// be derived is refused, by the material's name
func TestPassesRefusesWhatItCannotDerive(t *testing.T) {
	kind := slab(t, 2, 5)
	share, err := kind.Passes(0.06, 0.001)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(share-math.Exp(-0.01)) < 1e-3 {
		t.Errorf("1 mm passes %v: the metres were read as centimetres",
			share)
	}
	denser, err := slab(t, 2, 10).Passes(0.06, 0.001)
	if err != nil {
		t.Fatal(err)
	}
	if !(denser < share) {
		t.Errorf("twice the density passes %v, no less than %v",
			denser, share)
	}
	refused := map[string][2]float64{
		"an energy with no row":     {0.08, 0.001},
		"a thickness below nought":  {0.06, -0.001},
		"a thickness of NaN":        {0.06, math.NaN()},
		"a thickness past numbers":  {0.06, math.Inf(1)},
		"a thickness below numbers": {0.06, math.Inf(-1)},
	}
	for name, bad := range refused {
		_, err := kind.Passes(bad[0], bad[1])
		if err == nil {
			t.Errorf("%s: a share was answered", name)
			continue
		}
		if !strings.Contains(err.Error(), "iron") {
			t.Errorf("%s: the refusal does not name the material: %v",
				name, err)
		}
	}
	// coefficients and no density: nothing to multiply by
	fe, err := NewFormula(map[string]float64{"Fe": 1})
	if err != nil {
		t.Fatal(err)
	}
	bare, err := NewSubstance("iron", fe)
	if err != nil {
		t.Fatal(err)
	}
	stops, err := NewAttenuations(stopping(t, 0.06, 2))
	if err != nil {
		t.Fatal(err)
	}
	light, err := bare.Attenuating(stops)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := light.Passes(0.06, 0.001); err == nil {
		t.Error("a material with no density cited passed a share")
	}
	// a density and no coefficients: nothing to take
	if _, err := iron(t).Passes(0.06, 0.001); err == nil {
		t.Error("a material with no coefficients passed a share")
	}
}

// Beer and Lambert's law, as a property: what passes two thicknesses,
// one after the other, is what passes the first times what passes the
// second; a share is never more than all nor less than none; and more
// of the material never passes more.
func TestPassingIsBeerAndLamberts(t *testing.T) {
	rapid.Check(t, func(draw *rapid.T) {
		muOverRho := rapid.Float64Range(1e-3, 1e3).Draw(draw, "mu over rho")
		density := rapid.Float64Range(1e-3, 25).Draw(draw, "density")
		first := rapid.Float64Range(0, 0.5).Draw(draw, "first")
		second := rapid.Float64Range(0, 0.5).Draw(draw, "second")

		where, err := newSource("a release", "a material", "a paper",
			"a row")
		if err != nil {
			draw.Fatal(err)
		}
		stop, err := newAttenuation(0.06, muOverRho, where)
		if err != nil {
			draw.Fatal(err)
		}
		stops, err := NewAttenuations(stop)
		if err != nil {
			draw.Fatal(err)
		}
		fe, err := NewFormula(map[string]float64{"Fe": 1})
		if err != nil {
			draw.Fatal(err)
		}
		kind, err := NewMaterial("iron", fe, density)
		if err != nil {
			draw.Fatal(err)
		}
		kind, err = kind.Attenuating(stops)
		if err != nil {
			draw.Fatal(err)
		}

		one, err := kind.Passes(0.06, first)
		if err != nil {
			draw.Fatal(err)
		}
		other, err := kind.Passes(0.06, second)
		if err != nil {
			draw.Fatal(err)
		}
		both, err := kind.Passes(0.06, first+second)
		if err != nil {
			draw.Fatal(err)
		}
		for name, share := range map[string]float64{
			"the first": one, "the second": other, "both": both} {
			if !(share >= 0 && share <= 1) {
				draw.Fatalf("%s passes a share of %v", name, share)
			}
		}
		// the sum of two thicknesses is rounded once before the
		// exponent is taken, so the two sides agree to parts in a
		// million, not to the last bit
		if math.Abs(both-one*other) > 1e-12+1e-6*both {
			draw.Fatalf("both pass %v; one after the other, %v",
				both, one*other)
		}
		if both > one || both > other {
			draw.Fatalf("both pass %v, more than one alone: %v, %v",
				both, one, other)
		}
	})
}

package materials

import (
	"math"
	"testing"
)

// somewhere is a source read from somewhere, for the tests
func somewhere(t *testing.T) Source {
	t.Helper()
	where, err := newSource("a release", "a material", "a paper", "a row")
	if err != nil {
		t.Fatal(err)
	}
	return where
}

func TestASourceRefusesANamelessPart(t *testing.T) {
	refused := map[string][4]string{
		"no release":  {"", "gold", "a paper", "a row"},
		"no material": {"a release", "", "a paper", "a row"},
		"no paper":    {"a release", "gold", "", "a row"},
		"no row":      {"a release", "gold", "a paper", ""},
	}
	for name, parts := range refused {
		if _, err := newSource(parts[0], parts[1], parts[2], parts[3]); err == nil {
			t.Errorf("%s: a source was made", name)
		}
	}
}

func TestAReadingRefusesNonsenseOptics(t *testing.T) {
	where := somewhere(t)
	refused := map[string][3]float64{
		"a wavelength of nought":    {0, 1, 1},
		"a wavelength below nought": {-617, 1, 1},
		"a wavelength of NaN":       {math.NaN(), 1, 1},
		"a wavelength past numbers": {math.Inf(1), 1, 1},
		"an n below nought":         {617, -0.1, 1},
		"an n of NaN":               {617, math.NaN(), 1},
		"a k below nought":          {617, 1, -1},
		"a k of NaN":                {617, 1, math.NaN()},
		"a k past numbers":          {617, 1, math.Inf(1)},
	}
	for name, bad := range refused {
		if _, err := newReading(bad[0], bad[1], bad[2], where); err == nil {
			t.Errorf("%s: a reading was made", name)
		}
	}
	if _, err := newReading(617, 1, 1, Source{}); err == nil {
		t.Error("a reading read nowhere was made")
	}
	// a k of nought is a clear material, and is not refused
	if _, err := newReading(617, 1, 0, where); err != nil {
		t.Errorf("a reading of k nought was refused: %v", err)
	}
}

func TestOpticsRefuseNoBandsAndAReadingNeverMade(t *testing.T) {
	if _, err := newOptics(nil); err == nil {
		t.Error("optics of no bands were made")
	}
	if _, err := newOptics(map[Band]Reading{Red: {}}); err == nil {
		t.Error("optics of a reading never made were made")
	}
	// and optics answer a reading never made for a band they lack
	read, err := newReading(616.8, 0.21, 3.272, somewhere(t))
	if err != nil {
		t.Fatal(err)
	}
	sight, err := newOptics(map[Band]Reading{Red: read})
	if err != nil {
		t.Fatal(err)
	}
	if sight.Has(Green) {
		t.Error("the optics say they have a green reading")
	}
	if green := sight.At(Green); green.wavelength != 0 || green.N() != 0 || green.K() != 0 {
		t.Errorf("a green reading never made answers %v", green)
	}
}

func TestAnOpticalMaterialRefusesNoNameNoFormulaNoOptics(t *testing.T) {
	fe, err := NewFormula(map[string]float64{"Fe": 1})
	if err != nil {
		t.Fatal(err)
	}
	read, err := newReading(617, 2.67, 3.72, somewhere(t))
	if err != nil {
		t.Fatal(err)
	}
	sight, err := newOptics(map[Band]Reading{Red: read})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newOpticalMaterial("", fe, sight); err == nil {
		t.Error("an optical material with no name was made")
	}
	if _, err := newOpticalMaterial("iron", formula{}, sight); err == nil {
		t.Error("an optical material of no formula was made")
	}
	if _, err := newOpticalMaterial("iron", fe, Optics{}); err == nil {
		t.Error("an optical material of no optics was made")
	}
}

func TestAMaterialMadeOfADensityHasNoOptics(t *testing.T) {
	made, err := NewMaterial("ice", water(t), 0.92)
	if err != nil {
		t.Fatal(err)
	}
	if made.Optics().Has(Red) {
		t.Error("ice made of a density alone says it has red optics")
	}
	if read := made.Optics().At(Red); read.N() != 0 || read.K() != 0 {
		t.Errorf("a reading never made answers %v", read)
	}
}

func TestAbsorptionIsFourPiKOverTheWavelength(t *testing.T) {
	// k 1 at a thousand nanometres takes four pi of it a metre
	read, err := newReading(1000, 1.5, 1, somewhere(t))
	if err != nil {
		t.Fatal(err)
	}
	// the arithmetic runs on variables, as the method's does, so it is
	// the same roundings and not a constant folded once
	wavelength, k := 1000.0, 1.0
	want := 4 * math.Pi * k / (wavelength * 1e-9)
	if got := read.Absorption(); got != want {
		t.Errorf("k 1 at 1000 nm absorbs %v a metre, not %v", got, want)
	}
	// and every committed metal's reading takes just what its own
	// stored k and wavelength say, four pi k over the wavelength
	for _, name := range []string{"gold", "silver", "beryllium", "titanium"} {
		made, err := ByName(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, band := range []Band{Red, Green, Blue} {
			read := made.Optics().At(band)
			want := 4 * math.Pi * read.K() / (read.Wavelength() * 1e-9)
			if got := read.Absorption(); got != want {
				t.Errorf("%s %s: absorbs %v a metre, not %v",
					name, band, got, want)
			}
		}
	}
}

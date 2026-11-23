package materials

import (
	"strings"
	"testing"
)

// whole is a page with every kind of row a page can hold, each sound:
// the pages below are this one with one thing wrong.
func whole() page {
	return page{
		parts: map[string]float64{"Fe": 1},
		sight: []row{{Red, 600, 1.5, 2.5, "odd", "a file", noble, "a row"}},
		dense: &densityRow{"a table", 7.87, "a row"},
		stops: []coefficientRow{{"a page", FilmEnergy, 1.2, "a row"}},
	}
}

// A sound page is made under the name it is asked for, with its
// formula, its optics, its density and its coefficient, each where its
// row says. The anti-test: with no rows of optics or of density it is
// the bare substance, with no optics, no density and no coefficient
// it was not given.
func TestASoundPageIsMadeWithEveryRowItHolds(t *testing.T) {
	made, err := whole().made("odd")
	if err != nil {
		t.Fatal(err)
	}
	if made.Name() != "odd" || made.Density() != 7.87 ||
		!made.Optics().Has(Red) || made.Optics().At(Red).N() != 1.5 {
		t.Errorf("the page made %q at density %v with red n %v",
			made.Name(), made.Density(), made.Optics().At(Red).N())
	}
	if stop, there := made.Attenuations().At(FilmEnergy); !there ||
		stop.MuOverRho() != 1.2 {
		t.Errorf("the page's coefficient is %v, there %v", stop.MuOverRho(), there)
	}
	bare, err := page{parts: map[string]float64{"Fe": 1}}.made("plain")
	if err != nil {
		t.Fatal(err)
	}
	if bare.Name() != "plain" || bare.Density() != 0 || bare.Optics().Has(Red) ||
		len(bare.Attenuations().All()) != 0 {
		t.Errorf("a page of a formula alone made %+v", bare)
	}
}

// A page with one thing wrong is refused, by what is wrong: nothing it
// is made of, no name, a row read from nowhere, a reading that is not
// one, a density of nought, a coefficient at no energy, a density with
// no coefficients, and two coefficients at one energy.
func TestAPageWithOneThingWrongIsRefusedByIt(t *testing.T) {
	wrong := map[string]struct {
		name  string
		spoil func(leaf *page)
		says  string
	}{
		"nothing it is made of": {"odd", func(leaf *page) {
			leaf.parts = nil
		}, "a formula of no elements"},
		"no name": {"", func(*page) {}, "no name"},
		"no name and no optics": {"", func(leaf *page) {
			leaf.sight = nil
		}, "no name"},
		"an optical row read from no row": {"odd", func(leaf *page) {
			leaf.sight[0].text = ""
		}, "a source with no row"},
		"an optical row at no wavelength": {"odd", func(leaf *page) {
			leaf.sight[0].wavelength = 0
		}, "a reading at wavelength"},
		"a density read from no row": {"odd", func(leaf *page) {
			leaf.dense.text = ""
		}, "a source with no row"},
		"a density of nought": {"odd", func(leaf *page) {
			leaf.dense.density = 0
		}, "odd at density 0"},
		"a coefficient read from no row": {"odd", func(leaf *page) {
			leaf.stops[0].text = ""
		}, "a source with no row"},
		"a coefficient at no energy": {"odd", func(leaf *page) {
			leaf.stops[0].energy = 0
		}, "an attenuation at energy"},
		"a density with no coefficients": {"odd", func(leaf *page) {
			leaf.stops = nil
		}, "attenuations of no readings"},
		"two coefficients at one energy": {"odd", func(leaf *page) {
			leaf.stops = append(leaf.stops, leaf.stops[0])
		}, "two attenuations"},
	}
	for what, each := range wrong {
		leaf := whole()
		each.spoil(&leaf)
		made, err := leaf.made(each.name)
		if err == nil || !strings.Contains(err.Error(), each.says) ||
			made.Name() != "" {
			t.Errorf("%s: made %q, %v; want it to say %q", what,
				made.Name(), err, each.says)
		}
	}
}

// Every page is made under its own name: five metals, one semiconductor,
// the yard's six and the x-ray smoke's two. The anti-tests: no name is in two books, and a name in
// none is refused by name.
func TestEveryPageIsMadeUnderItsOwnName(t *testing.T) {
	seen := map[string]bool{}
	for _, book := range []map[string]page{metals, alloys, semiconductors, yard, xrays} {
		for name := range book {
			if seen[name] {
				t.Errorf("%s is in two books", name)
			}
			seen[name] = true
			made, err := ByName(name)
			if err != nil || made.Name() != name {
				t.Errorf("%s: made %q, %v", name, made.Name(), err)
			}
		}
	}
	if len(seen) != 15 {
		t.Errorf("%d materials by name, want 15 including nickel-aluminium bronze", len(seen))
	}
	if made, err := ByName("osmium"); err == nil || made.Name() != "" ||
		!strings.Contains(err.Error(), `no material "osmium"`) {
		t.Errorf("osmium, in no book: made %q, %v", made.Name(), err)
	}
}

// A substance is refused with no name and with a formula never made.
func TestASubstanceRefusesNoNameAndNoFormula(t *testing.T) {
	if _, err := NewSubstance("", water(t)); err == nil ||
		!strings.Contains(err.Error(), "no name") {
		t.Errorf("a substance with no name: %v", err)
	}
	if _, err := NewSubstance("mist", formula{}); err == nil ||
		!strings.Contains(err.Error(), "mist has no formula") {
		t.Errorf("a substance of no formula: %v", err)
	}
	if made, err := NewSubstance("mist", water(t)); err != nil ||
		made.Name() != "mist" {
		t.Errorf("a substance of water: %q, %v", made.Name(), err)
	}
}

// A mix refuses a material never made among its kinds; and a grain
// gives back the mix it was made of.
func TestAMixRefusesAMaterialNeverMadeAndAGrainKeepsItsMix(t *testing.T) {
	if _, err := NewFractions([]Material{ice(t), {}},
		[]float64{0.5, 0.5}); err == nil ||
		!strings.Contains(err.Error(), "never made") {
		t.Errorf("a mix with a material never made: %v", err)
	}
	mix, err := NewFractions([]Material{ice(t), iron(t)}, []float64{0.25, 0.75})
	if err != nil {
		t.Fatal(err)
	}
	grain, err := NewParticle(2, mix)
	if err != nil {
		t.Fatal(err)
	}
	if got := grain.Made(); got.Of("ice") != 0.25 || got.Of("iron") != 0.75 {
		t.Errorf("the grain is %v ice and %v iron", got.Of("ice"), got.Of("iron"))
	}
}

// A band says its name. The anti-test: one that is not red, green or
// blue is given none of their names, and says its number.
func TestABandSaysItsName(t *testing.T) {
	names := map[Band]string{Red: "red", Green: "green", Blue: "blue",
		Band(7): "band 7"}
	for band, want := range names {
		if got := band.String(); got != want {
			t.Errorf("band %d says %q, want %q", int(band), got, want)
		}
	}
}

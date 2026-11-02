package materials

import (
	"math"
	"testing"
)

// water is ice's formula, H2O, for the tests that need a material
func water(t *testing.T) Formula {
	t.Helper()
	made, err := NewFormula(map[string]float64{"H": 2, "O": 1})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// ice is water ice at its bulk density, for the tests
func ice(t *testing.T) Material {
	t.Helper()
	made, err := NewMaterial("ice", water(t), 0.92)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// iron is iron at its bulk density, for the tests
func iron(t *testing.T) Material {
	t.Helper()
	fe, err := NewFormula(map[string]float64{"Fe": 1})
	if err != nil {
		t.Fatal(err)
	}
	made, err := NewMaterial("iron", fe, 7.87)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func TestAFormulaRefusesWhatIsNotAnElement(t *testing.T) {
	refused := map[string]map[string]float64{
		"nothing at all":   {},
		"an unknown atom":  {"Xx": 1},
		"a count of none":  {"H": 0},
		"a count below it": {"H": -2, "O": 1},
		"a count of NaN":   {"H": math.NaN()},
	}
	for name, parts := range refused {
		if _, err := NewFormula(parts); err == nil {
			t.Errorf("%s: a formula was made", name)
		}
	}
}

func TestAFormulaKnowsItsMass(t *testing.T) {
	// water is two hydrogens and an oxygen: 18.015 to three places
	if mass := water(t).Mass(); math.Abs(mass-18.015) > 0.001 {
		t.Errorf("water weighs %v, not 18.015", mass)
	}
}

func TestAMaterialRefusesANonsenseDensity(t *testing.T) {
	for _, density := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if _, err := NewMaterial("ice", water(t), density); err == nil {
			t.Errorf("a material of density %v was made", density)
		}
	}
	if _, err := NewMaterial("", water(t), 0.92); err == nil {
		t.Error("a material with no name was made")
	}
	if _, err := NewMaterial("ice", Formula{}, 0.92); err == nil {
		t.Error("a material of no formula was made")
	}
}

func TestFractionsMustSumToExactlyOne(t *testing.T) {
	both := []Material{ice(t), iron(t)}
	refused := map[string][]float64{
		"short of one":          {0.6, 0.399},
		"past one":              {0.6, 0.41},
		"a share below nought":  {1.2, -0.2},
		"a share that is NaN":   {math.NaN(), 1},
		"more shares than kind": {0.5, 0.25, 0.25},
	}
	for name, shares := range refused {
		if _, err := NewFractions(both, shares); err == nil {
			t.Errorf("%s: fractions were made", name)
		}
	}
	if _, err := NewFractions(nil, nil); err == nil {
		t.Error("fractions of nothing were made")
	}
	if _, err := NewFractions(both, []float64{0.6, 0.4}); err != nil {
		t.Errorf("fractions summing to one were refused: %v", err)
	}
}

func TestFractionsAnswerByMaterial(t *testing.T) {
	made, err := NewFractions([]Material{ice(t), iron(t)},
		[]float64{0.75, 0.25})
	if err != nil {
		t.Fatal(err)
	}
	if share := made.Of("iron"); share != 0.25 {
		t.Errorf("iron is %v of the mix, not 0.25", share)
	}
	if share := made.Of("flan"); share != 0 {
		t.Errorf("flan is %v of a mix without it, not nought", share)
	}
}

func TestAParticleRefusesNoSizeAndNoMakeup(t *testing.T) {
	made, err := NewFractions([]Material{ice(t)}, []float64{1})
	if err != nil {
		t.Fatal(err)
	}
	for _, size := range []float64{0, -0.1, math.NaN()} {
		if _, err := NewParticle(size, made); err == nil {
			t.Errorf("a particle of size %v was made", size)
		}
	}
	if _, err := NewParticle(0.1, Fractions{}); err == nil {
		t.Error("a particle made of nothing was made")
	}
	grain, err := NewParticle(0.1, made)
	if err != nil {
		t.Fatal(err)
	}
	if grain.Size() != 0.1 {
		t.Errorf("a grain of 0.1 says it is %v", grain.Size())
	}
}

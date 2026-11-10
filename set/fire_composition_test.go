package set_test

import (
	"github.com/janearc/slayer/set"
	"math"
	"strings"
	"testing"
)

func composedFire(t *testing.T) string {
	return strings.Replace(fireRecipeFixture(t), "edge-temperature: {value: 900, unit: K}", `edge-temperature: {value: 900, unit: K}
        intermittency-start: 0.06
        temperature-profile: hot-tongues
        composition:
          continuum: 0.7
          lines:
            - species: Na I
              wavelength: {value: 588.995, unit: nm}
              radiance: {value: 0.01, unit: W/m^2/sr}
              convention: air
              source: https://physics.nist.gov/PhysRefData/Handbook/Tables/sodiumtable2.htm`, 1)
}

func TestFireCompositionRetainsSourceUnitsAndShapeChoices(t *testing.T) {
	raw := composedFire(t)
	if !strings.Contains(raw, "composition:") {
		t.Fatal("fixture has no composition")
	}
	made, err := set.Read("composition.yaml", []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	recipe, ok := made.Recipe("wood fire")
	if !ok {
		t.Fatal("missing recipe")
	}
	spec, ok := recipe.Fire()
	if !ok {
		t.Fatal("missing fire")
	}
	if spec.Flame.Composition == nil || !spec.Flame.HotTongues || spec.Flame.Intermittency != .06 {
		t.Fatal("new controls lost")
	}
	lines := spec.Flame.Composition.Lines()
	if len(lines) != 1 || math.Abs(lines[0].Wavelength.InMetres()-588.995e-9) > 1e-20 || lines[0].Convention != "air" || lines[0].Radiance != .01 {
		t.Fatalf("line inputs changed: %+v", lines)
	}
}

func TestFireCompositionAndLightingRejectInvalidContracts(t *testing.T) {
	for _, change := range [][2]string{
		{"unit: nm", "unit: m"}, {"W/m^2/sr", "watts"}, {"convention: air", "convention: maybe"},
		{"continuum: 0.7", "continuum: -1"}, {"intermittency-start: 0.06", "intermittency-start: 1"},
		{"temperature-profile: hot-tongues", "temperature-profile: hotter"},
	} {
		raw := strings.Replace(composedFire(t), change[0], change[1], 1)
		if _, err := set.Read("bad-composition.yaml", []byte(raw)); err == nil {
			t.Fatalf("accepted %s", change[1])
		}
	}
	raw := strings.Replace(fireRecipeFixture(t), "  light: none", "  renderer: cpu\n  volume-lighting: cached\n  light: none", 1)
	if _, err := set.Read("bad-cache.yaml", []byte(raw)); err == nil {
		t.Fatal("CPU cache request silently accepted")
	}
}

package set_test

import (
	"math"
	"os"
	"strings"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/set"
)

func fireRecipeFixture(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("../doc/fire-study.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestFireRecipeResolvesPhysicalUnitsAndDoesNotAliasSmoke(t *testing.T) {
	made, err := set.Read("fire.yaml", []byte(fireRecipeFixture(t)))
	if err != nil {
		t.Fatal(err)
	}
	recipe, ok := made.Recipe("wood fire")
	if !ok || recipe.Kind() != "fire" || made.Things()[0].Kind() != "fire" {
		t.Fatal("fire recipe did not resolve")
	}
	spec, ok := recipe.Fire()
	if !ok || spec.Flame.Radius != 0.45 || spec.Flame.Height != 1.5 ||
		spec.Flame.Hottest != measure.Kelvin(1750) || spec.Smoke == nil ||
		spec.Smoke.Heat != 40 || spec.Smoke.Combustion != 21 || spec.LightCells != 3 {
		t.Fatalf("fire units or sampling changed: %+v; smoke %+v; hot %.17g", spec, spec.Smoke, spec.Flame.Hottest.InKelvin())
	}
	spec.Smoke.Heat = 999
	again, _ := recipe.Fire()
	if again.Smoke.Heat != 40 {
		t.Fatal("caller changed the recipe")
	}
}

func TestFireRecipeRejectsInvalidUnitsAndCombinations(t *testing.T) {
	for _, pair := range [][2]string{
		{"unit: K", "unit: C"},
		{"unit: kW", "unit: W"},
		{"temperature: {value: 1750", "temperature: {value: 850"},
		{"radius: {value: 0.45", "radius: {value: 0"},
		{"heat-release: {value: 40", "heat-release: {value: 0"},
		{"soot-yield: 0.0024", "soot-yield: 0"},
		{"light-cells: 3", "light-cells: 2"},
		{"albedo: 0.3", "albedo: 1.1"},
		{"air-density: {value: 1.2", "air-density: {value: -1"},
		{"height: {value: 10", "height: {value: 0.1"},
		{"      light-cells: 3", "      typo: 3"},
		{"        absorption: {value: 1, unit: m^-1}\n", ""},
	} {
		raw := strings.Replace(fireRecipeFixture(t), pair[0], pair[1], 1)
		if raw == fireRecipeFixture(t) {
			t.Fatal("ineffective rejection fixture")
		}
		if _, err := set.Read("bad-fire.yaml", []byte(raw)); err == nil {
			t.Fatalf("accepted %s", pair[1])
		}
	}
}

func TestFireRecipeCanOmitSmokeAndUsesOnlyItsOwnConstructor(t *testing.T) {
	raw := fireRecipeFixture(t)
	start, end := strings.Index(raw, "      smoke:"), strings.Index(raw, "  set:")
	raw = raw[:start] + raw[end:]
	made, err := set.Read("no-smoke.yaml", []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	recipe, _ := made.Recipe("wood fire")
	spec, ok := recipe.Fire()
	if !ok || spec.Smoke != nil {
		t.Fatal("absent smoke gained a default column")
	}
	other, err := set.Read("other.yaml", []byte(ecosystemFixture(t)))
	if err != nil {
		t.Fatal(err)
	}
	optical, _ := other.Recipe("ridge telescope")
	if _, ok := optical.Fire(); ok {
		t.Fatal("optical recipe became a fire")
	}
}

const fireLayoutBlock = "      layout:\n        pattern: cluster\n        count: 7\n        radius: {value: 0.55, unit: m}\n        height-variation: 0.3\n"

func TestFireLayoutRecipeCarriesItsExtentAndSampling(t *testing.T) {
	raw := strings.Replace(fireRecipeFixture(t), "      flame:", fireLayoutBlock+"      flame:", 1)
	made, err := set.Read("cluster.yaml", []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	recipe, _ := made.Recipe("wood fire")
	spec, ok := recipe.Fire()
	if !ok || spec.Layout.Pattern != "cluster" || spec.Layout.Count != 7 ||
		spec.Layout.Radius != measure.Metres(0.55) || spec.Layout.Variation != 0.3 {
		t.Fatal("layout data lost")
	}
	if spec.Smoke == nil || math.Abs(spec.Smoke.Base-2) > 1e-12 {
		t.Fatal("smoke base ignored its fuel footprint")
	}
}

func TestFireLayoutRecipeRejectsMixedOrUnknownDimensions(t *testing.T) {
	for _, change := range [][2]string{
		{"pattern: cluster", "pattern: circle-ish"},
		{"count: 7", "count: 33"},
		{"count: 7", "count: 1"},
		{"radius: {value: 0.55", "length: {value: 0.55"},
		{"height-variation: 0.3", "height-variation: 0.6"},
		{"radius: {value: 0.55, unit: m}", "radius: {value: 0.55, unit: deg}"},
	} {
		block := strings.Replace(fireLayoutBlock, change[0], change[1], 1)
		raw := strings.Replace(fireRecipeFixture(t), "      flame:", block+"      flame:", 1)
		if _, err := set.Read("bad-layout.yaml", []byte(raw)); err == nil {
			t.Fatal("invalid layout accepted")
		}
	}
}

func TestFireLineRecipeCarriesLengthWithoutInventingARadius(t *testing.T) {
	block := strings.Replace(fireLayoutBlock, "pattern: cluster", "pattern: line", 1)
	block = strings.Replace(block, "radius: {value: 0.55", "length: {value: 4", 1)
	raw := strings.Replace(fireRecipeFixture(t), "      flame:", block+"      flame:", 1)
	made, err := set.Read("line.yaml", []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	recipe, _ := made.Recipe("wood fire")
	spec, _ := recipe.Fire()
	if spec.Layout.Pattern != "line" || spec.Layout.Length != measure.Metres(4) ||
		spec.Layout.Radius != measure.Metres(0) || math.Abs(spec.Smoke.Base-4.9) > 1e-12 {
		t.Fatal("line recipe lost its source extent")
	}
}

func TestBothDocumentedFireStudiesRead(t *testing.T) {
	for _, path := range []string{"../doc/fire-study.yaml", "../doc/fire-cluster.yaml"} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = set.Read(path, raw); err != nil {
			t.Fatal(err)
		}
	}
}

package stage_test

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
)

func waterMaterialFixture(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("../set/testdata/built.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return strings.Replace(string(raw),
		"      grass:\n        colour: {srgb_linear: [0.05, 0.06, 0.04]}\n        source: eye\n",
		"      grass:\n        clear:\n          through: 1\n          index: 1.333\n          gloss: 1\n"+
			"          extinction-per-metre: [0.1, 0.2, 0.3]\n"+
			"          water-composition:\n"+
			"            salinity: {value: 12, source: salinity receipt, at: '2026-11-18T18:00:00Z', qualifier: provisional}\n"+
			"            suspended-sediment-mg-per-litre: {value: 30, source: sediment receipt, qualifier: estimated}\n"+
			"            turbidity-fnu: {value: 6.1, source: turbidity receipt}\n"+
			"        source: synthetic optical coefficients\n", 1)
}

func TestWaterCompositionSurvivesSetAndStage(t *testing.T) {
	for _, schema := range []string{set.EcosystemSchema,
		"https://github.com/janearc/slayer/schemas/ecosystem-v2.json"} {
		written := "$schema: " + schema + "\n" + waterMaterialFixture(t)
		read, err := set.Read("water.yaml", []byte(written))
		if err != nil {
			t.Fatal(err)
		}
		said, _ := read.World()
		world, err := stage.World(said)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, item := range world.Things {
			if item.Material.Name != "grass" {
				continue
			}
			material := item.Material
			salt, hasSalt := material.WaterComposition.Salinity()
			silt, hasSilt := material.WaterComposition.SuspendedSediment()
			turbidity, hasTurbidity := material.WaterComposition.Turbidity()
			mediumWater, mediumSource, hasMediumWater := material.ClearMedium.WaterComposition()
			if !hasSalt || salt.Value != 12 || salt.Source != "salinity receipt" ||
				salt.At != "2026-11-18T18:00:00Z" || salt.Qualifier != "provisional" ||
				!hasSilt || silt.Value != 30 || silt.Source != "sediment receipt" ||
				silt.Qualifier != "estimated" ||
				!hasTurbidity || turbidity.Value != 6.1 || turbidity.Source != "turbidity receipt" ||
				!hasMediumWater || mediumWater != material.WaterComposition ||
				mediumSource != "synthetic optical coefficients" ||
				material.Source != "synthetic optical coefficients" ||
				material.ClearMedium.Extinction() != maths.RGB(.1, .2, .3) {
				t.Fatalf("staged composition or optics differ: %+v", material)
			}
			found = true
			break
		}
		if !found {
			t.Fatal("missing water material")
		}
	}
}

func TestWaterCompositionRejectsMissingSourceAndEmptyComposition(t *testing.T) {
	for _, text := range []string{
		strings.Replace(waterMaterialFixture(t), "source: salinity receipt", "source: ''", 1),
		strings.Replace(waterMaterialFixture(t), "2026-11-18T18:00:00Z", "tomorrow", 1),
		strings.Replace(waterMaterialFixture(t),
			"          water-composition:\n            salinity: {value: 12, source: salinity receipt, at: '2026-11-18T18:00:00Z', qualifier: provisional}\n"+
				"            suspended-sediment-mg-per-litre: {value: 30, source: sediment receipt, qualifier: estimated}\n"+
				"            turbidity-fnu: {value: 6.1, source: turbidity receipt}\n",
			"          water-composition: {}\n", 1),
	} {
		if _, err := set.Read("bad-water.yaml", []byte(text)); err == nil {
			t.Fatal("accepted invalid water observation")
		}
	}
}

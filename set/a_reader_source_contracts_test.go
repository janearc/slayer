package set_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

func TestFrontFadesAndSingleTargetStandsKeepTheirLimits(t *testing.T) {
	for _, fade := range []struct {
		value string
		want  time.Duration
	}{{"0", 0}, {"4", 4 * time.Second}} {
		raw := legacyNumericFixture(frontFixture(t))
		raw = strings.Replace(raw, "fade-time: {value: 0.2", "fade-time: {value: "+fade.value, 1)
		raw = strings.Replace(raw, "count: 3", "count: 1", 1)
		made, err := set.Read("front-limit.yaml", []byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		front := made.FireFronts()[0]
		if front.FadeTime() != fade.want {
			t.Fatalf("fade %s became %v", fade.value, front.FadeTime())
		}
	}
}

func TestRecipeSparkAxesAndFuelFootprintSurviveParsing(t *testing.T) {
	raw := legacyNumericFixture(strings.Replace(fireRecipeFixture(t), "      flame:", sparkBlock+"      flame:", 1))
	made, err := set.Read("spark-axes.yaml", []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	recipe, ok := made.Recipe("wood fire")
	if !ok {
		t.Fatal("lost fire recipe")
	}
	spec, ok := recipe.Fire()
	if !ok || spec.Sparks == nil {
		t.Fatal("lost firebrands")
	}
	spark := spec.Sparks
	if spark.Velocity != maths.V(0, 9, 0) || spark.Air != maths.V(0, 8, 0) ||
		spark.Jitter != maths.V(2, 2, 2) || spark.Disk.InMetres() != .45 {
		t.Fatalf("spark transport or release footprint: %+v", spark)
	}
}

func TestRecipePrimaryAndInheritedComplexityAreExplicit(t *testing.T) {
	for _, primary := range []string{"segmented", "monolithic"} {
		raw := legacyNumericFixture(ecosystemFixture(t))
		raw = strings.Replace(raw, "primary: segmented", "primary: "+primary, 1)
		raw = strings.Replace(raw, "        complexity: clumps\n", "", 1)
		made, err := set.Read("recipe-controls.yaml", []byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		row, ok := made.Recipe("ridge telescope")
		if !ok || row.Optical().Segmented != (primary == "segmented") {
			t.Fatal("primary choice changed")
		}
		if made.Stands()[0].Complexity() != "quick" {
			t.Fatal("stand did not inherit scene complexity")
		}
		if !made.HasSeed() || made.Seed() != 11 {
			t.Fatal("explicit scene seed lost its presence")
		}
	}
}

func TestFireCannotAlsoUseTheLegacyAlightSwitch(t *testing.T) {
	raw := legacyNumericFixture(fireRecipeFixture(t))
	raw = strings.Replace(raw, "        recipe: wood fire", "        recipe: wood fire\n        alight: true", 1)
	if _, err := set.Read("double-fire.yaml", []byte(raw)); err == nil || !strings.Contains(err.Error(), "fire the fire is already alight") {
		t.Fatalf("double ignition: %v", err)
	}
}

func TestFrontsRefuseAlreadyLitTargetsAndNonfireRecipes(t *testing.T) {
	raw := legacyNumericFixture(frontFixture(t))
	for _, change := range []struct{ before, after, want string }{
		{"kind: coast-redwood, count: 3", "kind: coast-redwood, alight: 1, count: 3", "existing unlit non-fire stand"},
		{"kind: coast-redwood, count: 3", "recipe: wood fire, count: 3", "existing unlit non-fire stand"},
	} {
		edited := strings.Replace(raw, change.before, change.after, 1)
		if edited == raw {
			t.Fatal("ineffective stand refusal")
		}
		if _, err := set.Read("front-target.yaml", []byte(edited)); err == nil || !strings.Contains(err.Error(), change.want) {
			t.Fatalf("front target: %v", err)
		}
	}
	// Keep the original fire recipe for its placed object, but ask the front
	// for an existing tree recipe. A missing name is a different refusal.
	raw = strings.Replace(raw, "  recipes:\n", "  recipes:\n    tree recipe: {kind: coast-redwood}\n", 1)
	raw = strings.Replace(raw, "\n      recipe: wood fire\n", "\n      recipe: tree recipe\n", 1)
	if _, err := set.Read("front-recipe.yaml", []byte(raw)); err == nil || !strings.Contains(err.Error(), "needs a known fire recipe") {
		t.Fatalf("front tree recipe: %v", err)
	}
}

func TestNonOpticalRecipesRefuseEitherOpticalBlockOnItsOwn(t *testing.T) {
	original := legacyNumericFixture(ecosystemFixture(t))
	original = strings.Replace(original, "kind: optical-observatory", "kind: home", 1)
	enclosure := strings.Index(original, "      enclosure:")
	telescope := strings.Index(original, "      telescope:")
	next := strings.Index(original, "    redwoods:")
	if enclosure < 0 || telescope < enclosure || next < telescope {
		t.Fatal("missing optical blocks")
	}
	for _, raw := range []string{original[:enclosure] + original[telescope:], original[:telescope] + original[next:]} {
		if _, err := set.Read("non-optical.yaml", []byte(raw)); err == nil || !strings.Contains(err.Error(), "is not an optical observatory") {
			t.Fatalf("non-optical recipe: %v", err)
		}
	}
}

func TestFuelBedsRequireAnExistingFireRecipe(t *testing.T) {
	raw := legacyNumericFixture(bedFixture(t))
	raw = strings.Replace(raw, "  recipes:\n", "  recipes:\n    tree recipe: {kind: coast-redwood}\n", 1)
	raw = strings.Replace(raw, "      recipe: surface fire", "      recipe: tree recipe", 1)
	if _, err := set.Read("bed-tree-recipe.yaml", []byte(raw)); err == nil ||
		!strings.Contains(err.Error(), "fuel bed needs a single-patch fire recipe") {
		t.Fatalf("bed tree recipe: %v", err)
	}
}

func TestExplicitZeroSceneAndObjectSeedsRemainPresent(t *testing.T) {
	raw := legacyNumericFixture(ecosystemFixture(t))
	raw = strings.Replace(raw, "  seed: 11\n", "  seed: 0\n", 1)
	raw = strings.Replace(raw, "        seed: 5\n", "        seed: 0\n", 1)
	made, err := set.Read("zero-seed.yaml", []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if !made.HasSeed() || made.Seed() != 0 || made.Things()[0].Seed() != 0 {
		t.Fatal("explicit zero seed was lost")
	}
	raw = strings.Replace(raw, "  seed: 0\n", "", 1)
	made, err = set.Read("unsaid-seed.yaml", []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if made.HasSeed() || made.Seed() != 0 {
		t.Fatal("unspecified seed became explicit")
	}
}

func TestAStandCanLightEveryTreeButCannotHaveAnEmptyHeightBand(t *testing.T) {
	raw := legacyNumericFixture(ecosystemFixture(t))
	raw = strings.Replace(raw, "        count: 5000", "        count: 3\n        alight: 3", 1)
	made, err := set.Read("all-alight.yaml", []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if made.Stands()[0].Alight() != 3 {
		t.Fatal("stand lost its all-lit choice")
	}
	raw = strings.Replace(raw, "        above: 1150", "        above: 1150\n        below: 1150", 1)
	if _, err := set.Read("empty-band.yaml", []byte(raw)); err == nil || !strings.Contains(err.Error(), "which is nowhere") {
		t.Fatalf("empty height band: %v", err)
	}
}

func TestFrontCountAllowsEightAndRefusesNinePlans(t *testing.T) {
	raw := legacyNumericFixture(frontFixture(t))
	first := strings.Index(raw, "    through the grove:")
	last := strings.Index(raw, "  film:")
	if first < 0 || last < first {
		t.Fatal("front fixture has no isolated plan block")
	}
	entry := raw[first:last]
	for _, count := range []int{8, 9} {
		var plans strings.Builder
		for i := range count {
			plans.WriteString(strings.Replace(entry, "through the grove:", fmt.Sprintf("front %d:", i), 1))
		}
		edited := raw[:first] + plans.String() + raw[last:]
		made, err := set.Read("front-count.yaml", []byte(edited))
		if count == 8 {
			if err != nil || len(made.FireFronts()) != 8 {
				t.Fatalf("eight fronts: %v", err)
			}
			for i, front := range made.FireFronts() {
				if front.Name() != fmt.Sprintf("front %d", i) {
					t.Fatal("front order changed")
				}
			}
		} else if err == nil || !strings.Contains(err.Error(), "at most eight fire fronts") {
			t.Fatalf("nine fronts: %v", err)
		}
	}
}

func TestPositiveSharedSmokeRadiusCanBeLessThanOneMetre(t *testing.T) {
	raw := legacyNumericFixture(frontFixture(t))
	raw = strings.Replace(raw, "smoke-radius: {value: 8", "smoke-radius: {value: 0.5", 1)
	made, err := set.Read("small-shared-smoke.yaml", []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if made.FireFronts()[0].SmokeRadius().InMetres() != .5 {
		t.Fatal("shared smoke radius changed")
	}
}

func TestSparkRecipePropagatesTheCombinedLivingParticleBudget(t *testing.T) {
	block := strings.Replace(sparkBlock, "rate: {value: 100", "rate: {value: 10000", 1)
	block = strings.Replace(block, "lifetime: {value: 3", "lifetime: {value: 1", 1)
	raw := legacyNumericFixture(strings.Replace(fireRecipeFixture(t), "      flame:", block+"      flame:", 1))
	if _, err := set.Read("spark-budget.yaml", []byte(raw)); err == nil ||
		!strings.Contains(err.Error(), "embers: invalid source or too many living particles") {
		t.Fatalf("combined particle budget: %v", err)
	}
}

package set_test

import (
	"os"
	"testing"

	"github.com/janearc/slayer/set"
)

// A village is asked for by a recipe that says its parcel; no other recipe
// has one. Written 2026-11-20 beside the code and not yet run.

// villageSetFile is the village study set, from the set directory.
const villageSetFile = "../sets/village-block.yaml"

// TestAVillageRecipeSaysItsParcel checks that the study's village recipe
// is read as a village and made.
func TestAVillageRecipeSaysItsParcel(t *testing.T) {
	text, err := os.ReadFile(villageSetFile)
	if err != nil {
		t.Fatal(err)
	}
	made, err := set.Read(villageSetFile, text)
	if err != nil {
		t.Fatal(err)
	}
	recipe, found := made.Recipe("village")
	if !found || recipe.Kind() != "village" {
		t.Fatalf("the recipe is %v, found %v", recipe.Kind(), found)
	}
	if _, err := recipe.Construction(); err != nil {
		t.Error(err)
	}
}

// TestAParcelOnAnotherRecipeIsRefused is an anti-test: a lashed street's
// recipe that says a parcel is refused, not read and ignored.
func TestAParcelOnAnotherRecipeIsRefused(t *testing.T) {
	_, err := lashingsSetWith(t, "      density: 1.0\n"+
		"      parcel: {width: {value: 40, unit: m}}\n")
	if err == nil {
		t.Error("a lashed street with a parcel is not refused")
	}
}

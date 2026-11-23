package session

import (
	"slices"
	"testing"
)

func TestACraftInSpaceMayHaveARecipeOfTheSignsThatStandOnTheirOwn(t *testing.T) {
	for _, kind := range []string{"sign-roof", "sign-screen", "sign-blade",
		"sign-awning", "tower", "street"} {
		if !slices.Contains(recipeKinds, kind) {
			t.Errorf("a craft in space may not have a recipe of %s", kind)
		}
	}
	for _, kind := range []string{"frontage", "spruce", "sign-neon"} {
		if slices.Contains(recipeKinds, kind) {
			t.Errorf("a craft in space may have a recipe of %s", kind)
		}
	}
}

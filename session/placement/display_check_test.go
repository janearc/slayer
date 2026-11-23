package placement

import (
	"errors"
	"strings"
	"testing"
)

func TestCheckDisplayTakesASetSceneTakes(t *testing.T) {
	read := readSelection(t, recipeSource)
	if err := CheckDisplay(read); err != nil {
		t.Fatalf("a set the display takes was refused: %v", err)
	}
	if _, err := NewDisplayCache().Scene(read); err != nil {
		t.Fatalf("Scene refused the set CheckDisplay took: %v", err)
	}
}

func TestCheckDisplayRefusesWithoutASeedInSceneWords(t *testing.T) {
	read := readSelection(t, strings.Replace(recipeSource, "  seed: 19\n", "", 1))
	checked := CheckDisplay(read)
	if checked == nil {
		t.Fatal("a set with no seed was taken")
	}
	_, shown := NewDisplayCache().Scene(read)
	if shown == nil || shown.Error() != checked.Error() {
		t.Fatalf("check said %q, Scene said %v", checked, shown)
	}
	if !strings.HasSuffix(checked.Error(), "(no seed)") {
		t.Fatalf("the refusal does not say which term failed: %v", checked)
	}
}

func TestCheckDisplayRefusesABurningPlacementAndNamesIt(t *testing.T) {
	burning := strings.Replace(recipeSource, "      visitor: {kind: human-scale-2m,", "      visitor: {alight: true, kind: human-scale-2m,", 1)
	read := readSelection(t, burning)
	checked := CheckDisplay(read)
	var named *DisplayThingError
	if !errors.As(checked, &named) || named.Thing != "visitor" {
		t.Fatalf("the burning placement was not named: %v", checked)
	}
	if !errors.Is(checked, errBurning) {
		t.Fatalf("the refusal is not the burning refusal: %v", checked)
	}
	_, shown := NewDisplayCache().Scene(read)
	if shown == nil || shown.Error() != checked.Error() {
		t.Fatalf("check said %q, Scene said %v", checked, shown)
	}
}

func TestDisplayThingErrorSaysNameColonRefusal(t *testing.T) {
	refusal := &DisplayThingError{Thing: "tower", Err: errors.New("no mesh")}
	if refusal.Error() != "tower: no mesh" {
		t.Fatalf("got %q", refusal.Error())
	}
	if refusal.Unwrap().Error() != "no mesh" {
		t.Fatalf("the refusal alone was %q", refusal.Unwrap())
	}
}

func TestDisplayGateStopsABurningPlacementBeforeItsRecipe(t *testing.T) {
	burning := strings.Replace(recipeSource, "      first: {recipe: low,", "      first: {alight: true, recipe: low,", 1)
	read := readSelection(t, burning)
	for _, each := range read.Things() {
		if each.Name() != "first" {
			continue
		}
		if _, _, err := displayGate(read, each); !errors.Is(err, errBurning) {
			t.Fatalf("a burning recipe placement passed the gate: %v", err)
		}
	}
}

package main

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

func TestFireRecipeAndNamedPlacementKeepSourceOverflowRefusals(t *testing.T) {
	given := fireOnFixture(t)
	read := readBoundarySet(t, given.set)
	stage := staging{given: given, recipes: read, world: thing.NewWorld(), radii: map[string]float64{}}
	if _, err := stage.putFireRecipe("wood fire", 1, 0, maths.Vec{}, ""); err != nil {
		t.Fatal(err)
	}
	stage = staging{given: given, recipes: read, world: thing.NewWorld()}
	stage.given.exposure = math.MaxFloat64
	if _, err := stage.putFireRecipe("wood fire", 1, 0, maths.Vec{}, ""); err == nil || !strings.Contains(err.Error(), "emit: nonfinite volume source") {
		t.Fatalf("recipe lost source overflow: %v", err)
	}
	stage = staging{given: given, recipes: read, world: thing.NewWorld(), have: stock{}, places: map[string]maths.Vec{}, radii: map[string]float64{}}
	stage.given.exposure = math.MaxFloat64
	if err := stage.terrain(read); err != nil {
		t.Fatal(err)
	}
	if err := stage.stand(read); err == nil || !strings.Contains(err.Error(), "the tree: emit: nonfinite volume source") {
		t.Fatalf("named placement lost source overflow: %v", err)
	}
}

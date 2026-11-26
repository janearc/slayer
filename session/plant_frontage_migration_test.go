package session

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/set"
)

// A profile-backed frontage must use the original species' spacing data;
// otherwise it is treated like an unspaced terrace and changes the stand.
func TestMigratedPlantFrontagesKeepSpeciesSpacing(t *testing.T) {
	path := filepath.Join("..", "sets", "groningen.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	read, err := set.Read(path, raw)
	if err != nil {
		t.Fatal(err)
	}
	stage := testStaging()
	stage.recipes = read
	checked := 0
	for _, stand := range read.Stands() {
		recipe, ok := read.Recipe(stand.Recipe())
		if !ok || !recipe.IsPlant() {
			continue
		}
		actual, err := stage.fronted(stand)
		if err != nil {
			t.Fatal(err)
		}
		line, err := stage.frontageOf(stand)
		if err != nil {
			t.Fatal(err)
		}
		item, err := cities.ItemOf(recipe.PlantSpecies())
		if err != nil {
			t.Fatal(err)
		}
		if !item.Spaced {
			t.Fatalf("%s has no species spacing", recipe.PlantSpecies())
		}
		habit, err := cities.HabitNamed(stand.Habit())
		if err != nil {
			t.Fatal(err)
		}
		count, seed := stand.Count()
		want, err := habit.Sites(cities.OnFrontage(line), item, cities.Seed(seed), count)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(actual, want) {
			t.Fatalf("%s lost its species spacing", stand.Name())
		}
		checked++
	}
	if checked != 4 {
		t.Fatalf("checked %d plant frontages, want 4", checked)
	}
}

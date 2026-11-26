package session

import (
	"fmt"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

// coniferFixture makes the profile-backed recipe through the scene reader.
func coniferFixture(t *testing.T, species string) (set.Set, manufacture.Kind) {
	t.Helper()
	source := fmt.Sprintf(`scene:
  name: accounting
  recipes:
    plant: {plant: {family: conifer, file: ../manufacture/plants/conifers/conifer-profiles.yaml, species: %s}}
  set:
    things:
      specimen: {recipe: plant, seed: 7, east: 0, north: 0}
`, species)
	read, err := set.Read("../sets/accounting.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	recipe, ok := read.Recipe("plant")
	if !ok {
		t.Fatal("missing plant recipe")
	}
	kind, err := recipe.Construction()
	if err != nil {
		t.Fatal(err)
	}
	if kind == nil {
		t.Fatal("plant recipe has no constructor")
	}
	return read, *kind
}

func TestSceneRecipeKeepsItsExactPublicStoredFaces(t *testing.T) {
	for _, name := range []string{"can", "coast-redwood"} {
		t.Run(name, func(t *testing.T) {
			given := mustParse(t)
			var kind manufacture.Kind
			var read set.Set
			var err error
			category, recipe := name, ""
			if name == "coast-redwood" {
				read, kind = coniferFixture(t, name)
				category, recipe = "plant-profile", "plant"
			} else {
				kind, err = manufacture.KindNamed(name)
				if err != nil {
					t.Fatal(err)
				}
			}
			sown := manufacture.Sow(stir(given.seed, 7))
			object, err := manufacture.MakeWithComplexity(kind,
				sown, "quick")
			if err != nil {
				t.Fatal(err)
			}
			parts, err := object.Things(maths.Subspace{})
			if err != nil {
				t.Fatal(err)
			}
			stored, logical := publicGeometryCounts(t, parts)
			if !(stored > 1 && logical >= stored) {
				t.Fatal("reference lost its real stored and placed faces")
			}
			if name == "coast-redwood" && logical <= stored {
				t.Fatal("reference lost its actual repeated prototype savings")
			}
			stage := staging{given: given, world: thing.NewWorld(), have: stock{}, radii: map[string]float64{}, recipes: read}
			_, err = stage.putRecipe(category, recipe, "quick", 7, 0, maths.Vec{}, false, name)
			if err != nil || stage.made.things != 1 || stage.made.kept != stored || stage.made.faces != logical ||
				len(stage.world.Things) != len(parts) {
				t.Fatalf("actual geometry failed or lost counts: %v, %+v; public stored%d/logical%d", err, stage.made, stored, logical)
			}
		})
	}
}

func TestSceneRecipeReportsItsActualCarriedAndFireLights(t *testing.T) {
	for _, name := range []string{"spotlight", "spruce"} {
		t.Run(name, func(t *testing.T) {
			given := mustParse(t)
			stage := staging{given: given, world: thing.NewWorld(), have: stock{}, radii: map[string]float64{}}
			burning := name == "spruce"
			category, recipe := name, ""
			if burning {
				stage.recipes, _ = coniferFixture(t, name)
				category, recipe = "plant-profile", "plant"
			}
			_, err := stage.putRecipe(category, recipe, "quick", 7, 0, maths.Vec{}, burning, name)
			if err != nil {
				t.Fatal(err)
			}
			if len(stage.world.Lamps) == 0 || stage.made.lamps != len(stage.world.Lamps) {
				t.Fatalf("%s reported %d lamps for %d actual sources", name, stage.made.lamps, len(stage.world.Lamps))
			}
			if burning && (stage.made.burning != 1 || len(stage.world.Volumes) != 2) {
				t.Fatal("single requested burning specimen lost its actual flame or smoke")
			}
			if !burning && (stage.made.burning != 0 || len(stage.world.Volumes) != 0) {
				t.Fatal("carried-light control ignited a specimen")
			}
		})
	}
}

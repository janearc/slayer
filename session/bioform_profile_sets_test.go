package session

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

func readBioformProfileSet(t *testing.T, name string) set.Set {
	t.Helper()
	path := filepath.Join("..", "sets", "bioforms", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	read, err := set.Read(path, raw)
	if err != nil {
		t.Fatal(err)
	}
	return read
}

func TestBioformAnimalSetsReadTheirProfiles(t *testing.T) {
	for _, name := range []string{
		"fish-bones.yaml", "fish-natives.yaml", "fish-sea.yaml", "fish-skin.yaml",
		"birds-standing.yaml", "birds-gliding.yaml", "birds-honopu.yaml",
		"bird-grey-heron-standing.yaml", "bird-grey-heron-standing-quarter.yaml",
		"bird-grey-heron-wing.yaml", "bird-grey-heron-gliding.yaml",
		"bird-grey-heron-outline.yaml", "bird-grey-heron-outline-above.yaml",
		"bird-grey-heron-bones.yaml", "bird-great-frigatebird-gliding.yaml",
		"bird-white-tailed-tropicbird-gliding.yaml",
	} {
		t.Run(name, func(t *testing.T) {
			read := readBioformProfileSet(t, name)
			if len(read.Things()) == 0 {
				t.Fatal("no placements")
			}
			for _, each := range read.Things() {
				recipe, found := read.Recipe(each.Recipe())
				if !found || !recipe.IsAnimal() {
					t.Fatalf("%s does not use an animal profile", each.Name())
				}
			}
		})
	}
}

func TestMigratedBioformSchoolAndBirdStageTheirSpecifiedGeometry(t *testing.T) {
	for _, tc := range []struct {
		file, placement, species string
		position                 maths.Vec
		seed                     uint64
	}{
		{"fish-sea.yaml", "sardines", "pacific-sardine", maths.V(.5, -2.6, -2.5), 14},
		{"bird-grey-heron-standing.yaml", "bird", "grey-heron", maths.V(0, 0, 0), 7},
	} {
		t.Run(tc.placement, func(t *testing.T) {
			read := readBioformProfileSet(t, tc.file)
			var found bool
			for _, each := range read.Things() {
				if each.Name() != tc.placement {
					continue
				}
				found = true
				if each.Seed() != tc.seed {
					t.Fatalf("seed %d, want %d", each.Seed(), tc.seed)
				}
				recipe, ok := read.Recipe(each.Recipe())
				if !ok || recipe.AnimalSpecies() != tc.species {
					t.Fatalf("wrong species for %s", tc.placement)
				}
				actual, err := recipe.Construction()
				if err != nil {
					t.Fatal(err)
				}
				var expected manufacture.Kind
				if tc.placement == "sardines" {
					raw, err := os.ReadFile("../manufacture/animals/vertebrates/fishes/fish.yaml")
					if err != nil {
						t.Fatal(err)
					}
					book, err := manufacture.ReadFishProfiles("fish.yaml", raw)
					if err != nil {
						t.Fatal(err)
					}
					expected, err = book.NewFish(tc.species, manufacture.FishOptions{School: 500})
					if err != nil {
						t.Fatal(err)
					}
				} else {
					raw, err := os.ReadFile("../manufacture/animals/vertebrates/dinosaurs/theropods/birds/birds.yaml")
					if err != nil {
						t.Fatal(err)
					}
					book, err := manufacture.ReadBirdProfiles("birds.yaml", raw)
					if err != nil {
						t.Fatal(err)
					}
					expected, err = book.NewBird(tc.species, "standing")
					if err != nil {
						t.Fatal(err)
					}
				}
				got, err := manufacture.MakeWithComplexity(*actual, manufacture.Sow(tc.seed), "quick")
				if err != nil {
					t.Fatal(err)
				}
				want, err := manufacture.MakeWithComplexity(expected, manufacture.Sow(tc.seed), "quick")
				if err != nil {
					t.Fatal(err)
				}
				gotParts, err := got.Things(maths.Local())
				if err != nil {
					t.Fatal(err)
				}
				wantParts, err := want.Things(maths.Local())
				if err != nil {
					t.Fatal(err)
				}
				if got.Bounds() != want.Bounds() || !reflect.DeepEqual(gotParts, wantParts) {
					t.Fatal("recipe changed constructed geometry or material")
				}
			}
			if !found {
				t.Fatal("placement missing")
			}
			stage := testStaging()
			stage.given = mustParse(t)
			stage.groundlessAir, stage.recipes = true, read
			if err := stage.stand(read); err != nil {
				t.Fatal(err)
			}
			if stage.places[tc.placement] != tc.position || len(stage.world.Things) == 0 {
				t.Fatalf("failed to stage %s at %v", tc.placement, tc.position)
			}
		})
	}
}

package manufacture

import (
	"fmt"
	"github.com/janearc/slayer/scene/shape"
	"gopkg.in/yaml.v3"
	"testing"
)

// fishHistoricalRows maps old labels to constructors only inside tests.
func fishHistoricalRows(t *testing.T) map[string]maker {
	t.Helper()
	book, err := ReadFishProfiles("fish.yaml", fishRaw)
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Species map[string]struct{ Schools []int }
	}
	if err := yaml.Unmarshal(fishRaw, &document); err != nil {
		t.Fatal(err)
	}
	rows := map[string]maker{}
	for name, species := range document.Species {
		for _, each := range []struct {
			suffix  string
			options FishOptions
		}{{"", FishOptions{}}, {"-glide", FishOptions{Glide: true}}, {"-bones", FishOptions{Bones: true}}} {
			row, err := book.NewFish(name, each.options)
			if err != nil {
				t.Fatal(err)
			}
			rows[name+each.suffix] = row.how
		}
		for _, count := range species.Schools {
			row, err := book.NewFish(name, FishOptions{School: count})
			if err != nil {
				t.Fatal(err)
			}
			rows[fmt.Sprintf("%s-school-%d", name, count)] = row.how
		}
	}
	return rows
}

// TestFishKindsAreMade holds every fish kind to being made from a seed,
// as surfaces in looks.
func TestFishKindsAreMade(t *testing.T) {
	for name, row := range fishHistoricalRows(t) {
		made, err := row.made(sow(7))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(made.native) == 0 && len(made.groups) == 0 {
			t.Fatalf("%s made nothing", name)
		}
		for _, one := range made.native {
			if one.Material == nil {
				t.Fatalf("%s has a surface in no look", name)
			}
		}
	}
}

func TestFishSpeciesAreNotRegistryAliases(t *testing.T) {
	rows := table()
	for name := range fishHistoricalRows(t) {
		if _, ok := rows[name]; ok {
			t.Errorf("registered old fish alias %s", name)
		}
	}
	if _, err := KindNamed("pacific-sardine-school-500"); err == nil {
		t.Fatal("old fish kind still resolves")
	}
}

func fishStored(t *testing.T, count int) (int, int) {
	t.Helper()
	book, err := ReadFishProfiles("fish.yaml", fishRaw)
	if err != nil {
		t.Fatal(err)
	}
	row, err := book.NewFish("pacific-sardine", FishOptions{School: count})
	if err != nil {
		t.Fatal(err)
	}
	made, err := Make(row, Sow(9))
	if err != nil {
		t.Fatal(err)
	}
	meshes := map[shape.Shape]bool{}
	for _, group := range made.groups {
		for _, member := range group.Members() {
			meshes[member.Mesh] = true
		}
	}
	return made.faces(), len(meshes)
}

// TestFishSchoolDoesNotGrowItsMeshes holds a school of ten times the fish
// to the same stored faces and the same shared meshes.
func TestFishSchoolDoesNotGrowItsMeshes(t *testing.T) {
	few, fewMeshes := fishStored(t, 500)
	many, manyMeshes := fishStored(t, 5000)
	if few != many || fewMeshes != manyMeshes {
		t.Fatalf("500 fish store %d faces in %d meshes, 5000 store %d in "+
			"%d", few, fewMeshes, many, manyMeshes)
	}
}

func TestFishRefusesAnUnsownSeed(t *testing.T) {
	if _, err := (fishRow{}).made(seed{}); err == nil {
		t.Fatal("a fish grew from an unsown seed")
	}
}

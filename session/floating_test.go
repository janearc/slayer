package session

import (
	"testing"

	"github.com/janearc/slayer/set"
)

func TestAKindThatFloatsIsKnownByItsWaterline(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"sampan", "fishing-boat-at-anchor", "ferry"} {
		waterline, floats := afloatKind(name)
		if !floats || waterline != 0 {
			t.Errorf("%s: floats %v at waterline %v", name, floats, waterline)
		}
	}
	if waterline, floats := afloatKind("alfa"); !floats || waterline <= 0 {
		t.Errorf("the alfa floats with its keel below the waterline: %v, %v",
			floats, waterline)
	}
}

func TestAKindThatDoesNotFloatIsNotOnTheSea(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"spruce", "yurt", "pier", "sampan-2"} {
		if _, floats := afloatKind(name); floats {
			t.Errorf("%s does not float", name)
		}
	}
}

func TestEachBoatOfAStandIsDrawnAfresh(t *testing.T) {
	t.Parallel()
	var boats, trees set.Stand
	boats, trees = standOf(t, "sampan"), standOf(t, "spruce")
	seen := map[uint64]bool{}
	for index := range 24 {
		seen[standPick(boats, 5, index)] = true
	}
	if len(seen) != 24 {
		t.Errorf("24 boats were made from %d seeds", len(seen))
	}
	varied := map[uint64]bool{}
	for index := range 24 {
		varied[standPick(trees, 5, index)] = true
	}
	if len(varied) != standVariety {
		t.Errorf("24 trees were made from %d seeds, not the %d varieties",
			len(varied), standVariety)
	}
}

// standOf is a stand of a kind, read from a set's text.
func standOf(t *testing.T, kind string) set.Stand {
	t.Helper()
	selector, recipes := "kind: "+kind, ""
	if kind == "spruce" {
		selector = "recipe: specimen-spruce"
		recipes = "  recipes:\n    specimen-spruce: {plant: {family: conifer, file: ../manufacture/plants/conifers/conifer-profiles.yaml, species: spruce}}\n"
	}
	text := "scene:\n  name: a stand\n" + recipes + "  set:\n    stands:\n      those:\n" +
		"        " + selector + "\n        count: 2\n        seed: 1\n"
	read, err := set.Read("../sets/stand.yaml", []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	stands := read.Stands()
	if len(stands) != 1 {
		t.Fatalf("a set of one stand read %d", len(stands))
	}
	return stands[0]
}

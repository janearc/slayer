package manufacture

import (
	"slices"
	"testing"
)

func TestPlantBookSpeciesEnumeratesCheckedContents(t *testing.T) {
	for _, tc := range []struct{ family, species string }{
		{"seasonal", "valley-oak"}, {"conifer", "norway"},
		{"fern", "fern"}, {"cactus", "saguaro"},
	} {
		t.Run(tc.family, func(t *testing.T) {
			name, raw, err := EmbeddedPlantBook(tc.family)
			if err != nil {
				t.Fatal(err)
			}
			book, err := ReadPlantBook(tc.family, name, raw)
			if err != nil {
				t.Fatal(err)
			}
			names := book.Species()
			if !slices.IsSorted(names) || !slices.Contains(names, tc.species) {
				t.Fatalf("species: %v", names)
			}
			for _, species := range names {
				if !book.HasSpecies(species) {
					t.Fatalf("listed absent species %q", species)
				}
			}
			names[0] = "changed"
			if slices.Contains(book.Species(), "changed") {
				t.Fatal("caller changed checked book names")
			}
		})
	}
	var absent *PlantBook
	if len(absent.Species()) != 0 {
		t.Fatal("nil book has species")
	}
}

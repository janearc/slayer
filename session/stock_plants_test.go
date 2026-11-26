package session

import (
	"testing"

	"github.com/janearc/slayer/manufacture"
)

// The hillside selectors keep the checked family geometry and their seed path.
func TestHillsidePlantSelectorsUseCheckedBooks(t *testing.T) {
	given := mustParse(t)
	for _, tc := range []struct{ family, species string }{
		{"conifer", "spruce"}, {"conifer", "norway"}, {"fern", "fern"},
	} {
		t.Run(tc.species, func(t *testing.T) {
			book, err := hillsideBook(tc.family)
			if err != nil {
				t.Fatal(err)
			}
			row, err := book.New(manufacture.PlantOptions{Species: tc.species})
			if err != nil {
				t.Fatal(err)
			}
			want, err := manufacturedKind(given, row, 2, "quick")
			if err != nil {
				t.Fatal(err)
			}
			got, err := manufactured(given, tc.species, 2)
			if err != nil {
				t.Fatal(err)
			}
			if got.bounds != want.bounds || got.faces != want.faces || got.stored != want.stored || len(got.things) != len(want.things) {
				t.Fatalf("%s: got bounds %v, faces %d/%d and %d things; want %v, %d/%d and %d", tc.species, got.bounds, got.faces, got.stored, len(got.things), want.bounds, want.faces, want.stored, len(want.things))
			}
			for i := range got.things {
				if got.things[i].Shape.Bounds() != want.things[i].Shape.Bounds() || got.things[i].At(0) != want.things[i].At(0) {
					t.Fatalf("%s part %d changed shape or placement", tc.species, i)
				}
			}
		})
	}
	if _, err := manufactured(given, "no-such-hillside-species", 0); err == nil {
		t.Fatal("accepted an unknown selector")
	}
	if _, err := manufactured(given, "can", 0); err != nil {
		t.Fatalf("ordinary non-plant kind refused: %v", err)
	}
}

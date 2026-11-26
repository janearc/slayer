package main

import (
	"testing"

	"github.com/janearc/slayer/manufacture"
)

// The menu retains each card's identity and presentation, but animal cards
// construct through explicit checked profiles. Independent geometry witnesses
// preserve the pre-migration outputs for every bird and fish representation.
func TestMigratedAnimalMenuKeepsPresentations(t *testing.T) {
	menu, err := readMenu()
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, entry := range menu.Entries {
		if entry.Animal != nil {
			counts[entry.Animal.Family]++
		}
	}
	if counts["bird"] != 32 || counts["fish"] != 31 {
		t.Fatalf("migrated cards: %d birds, %d fish", counts["bird"], counts["fish"])
	}
	for _, tc := range []struct {
		name, family, species, pose string
		count                       int
	}{
		{"bird-grey-heron-standing", "bird", "grey-heron", "standing", 0},
		{"bird-grey-heron-outline", "bird", "grey-heron", "outline", 0},
		{"pacific-bluefin-tuna-glide", "fish", "pacific-bluefin-tuna", "glide", 0},
		{"pacific-sardine-school-500", "fish", "pacific-sardine", "swim", 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entries, err := menu.selectEntries([]string{tc.name})
			if err != nil || len(entries) != 1 {
				t.Fatalf("selection: %v", err)
			}
			if entries[0].Animal == nil {
				t.Fatal("card did not select an animal profile")
			}
			spec := entries[0].Animal
			count := 0
			if spec.Count != nil {
				count = *spec.Count
			}
			pose := spec.Pose
			if pose == "" && spec.Family == "fish" {
				pose = "swim"
			}
			if spec.Family != tc.family || spec.Species != tc.species || pose != tc.pose || count != tc.count {
				t.Fatalf("wrong descriptor: %+v", spec)
			}
			actual, err := entries[0].kind()
			if err != nil {
				t.Fatal(err)
			}
			got, err := manufacture.MakeWithComplexity(actual, manufacture.Sow(5), "quick")
			if err != nil {
				t.Fatal(err)
			}
			if got.PartCount() == 0 {
				t.Fatal("profile descriptor made no parts")
			}
		})
	}
}

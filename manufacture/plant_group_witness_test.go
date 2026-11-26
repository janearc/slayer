package manufacture

import "testing"

func TestPlantGroupsRetainSeasonAndDryFoliage(t *testing.T) {
	book, err := ReadPlantProfiles("plants.yaml", plantProfilesRaw)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		species, season string
		leaf, dry       bool
	}{
		{"valley-oak", "summer", true, false},
		{"valley-oak", "winter", false, false},
		{"joshua-tree", "summer", true, true},
	} {
		t.Run(tc.species+"/"+tc.season, func(t *testing.T) {
			row, err := book.NewPlant(tc.species)
			if err != nil {
				t.Fatal(err)
			}
			row, err = WithSeason(row, tc.season)
			if err != nil {
				t.Fatal(err)
			}
			made, err := makeOf(row, Sow(11))
			if err != nil {
				t.Fatal(err)
			}
			_, wood := made.groups[body]
			_, leaves := made.groups[leaf]
			_, dry := made.groups[plantDryLeaf]
			if !wood || leaves != tc.leaf || dry != tc.dry {
				t.Fatalf("groups wood=%v leaves=%v dry=%v", wood, leaves, dry)
			}
		})
	}
}

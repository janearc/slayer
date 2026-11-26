package main

import "testing"

// These profile-backed animals are catalogue specimens in their own right,
// rather than only the carved gargoyles made from their studies.
func TestCatalogueContainsLiveAllosaurusAndSquid(t *testing.T) {
	menu, err := readMenu()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []struct{ name, family string }{
		{"allosaurus", "theropod"},
		{"humboldt-squid", "squid"},
		{"orca", "whale"},
		{"pliosaur", "reptile"},
		{"purussaurus", "reptile"},
		{"hatzegopteryx", "reptile"},
	} {
		t.Run(want.name, func(t *testing.T) {
			rows, err := menu.selectEntries([]string{want.name})
			if err != nil || len(rows) != 1 {
				t.Fatalf("select %s: %v, %v", want.name, rows, err)
			}
			row := rows[0]
			if row.Animal == nil || row.Animal.Family != want.family || row.Animal.Species != want.name || row.source == "" {
				t.Fatalf("%s has no checked animal profile: %+v", want.name, row.Animal)
			}
			object, err := defaultPlan(t).construct(row, "quick")
			if err != nil || object == nil || object.faces == 0 {
				t.Fatalf("construct %s: %v, %v", want.name, object, err)
			}
		})
	}
}

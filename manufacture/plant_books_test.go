package manufacture

import "testing"

func TestPlantBooksResolveFamiliesAndIsolateEmbeddedBytes(t *testing.T) {
	for _, tc := range []struct {
		family, species, filename string
	}{
		{"seasonal", "valley-oak", "plant-profiles.yaml"},
		{"conifer", "coast-redwood", "conifer-profiles.yaml"},
		{"fern", "fern", "fern-profiles.yaml"},
		{"cactus", "saguaro", "cactus-profiles.yaml"},
	} {
		t.Run(tc.family, func(t *testing.T) {
			name, raw, err := EmbeddedPlantBook(tc.family)
			if err != nil || name != tc.filename || len(raw) == 0 {
				t.Fatalf("embedded book %q: %v", name, err)
			}
			book, err := ReadPlantBook(tc.family, name, raw)
			if err != nil {
				t.Fatal(err)
			}
			row, err := book.New(PlantOptions{Species: tc.species})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := MakeWithComplexity(row, Sow(5), "quick"); err != nil {
				t.Fatal(err)
			}
			first := raw[0]
			for i := range raw {
				raw[i] = 0
			}
			_, again, err := EmbeddedPlantBook(tc.family)
			if err != nil || again[0] != first {
				t.Fatal("embedded bytes were mutated")
			}
			if _, err := book.New(PlantOptions{Species: tc.species}); err != nil {
				t.Fatalf("source byte mutation changed checked book: %v", err)
			}
			if _, err := book.New(PlantOptions{Species: "absent"}); err == nil {
				t.Fatal("accepted absent species")
			}
			if _, err := ReadPlantBook(tc.family, name, []byte("bad: [")); err == nil {
				t.Fatal("accepted malformed book")
			}
		})
	}
	if _, _, err := EmbeddedPlantBook("unknown"); err == nil {
		t.Fatal("accepted unknown embedded family")
	}
	if _, err := ReadPlantBook("unknown", "bad", nil); err == nil {
		t.Fatal("accepted unknown family")
	}
	var absent *PlantBook
	if _, err := absent.New(PlantOptions{}); err == nil {
		t.Fatal("accepted nil book")
	}
}

func TestPlantBookOptionRulesAndShootForms(t *testing.T) {
	shoot, empty, whole, autumn, winter := "shoot", "", "whole", "autumn", "winter"
	for _, tc := range []struct {
		family, species        string
		season, representation *string
		valid                  bool
	}{
		{"seasonal", "valley-oak", nil, nil, true},
		{"seasonal", "valley-oak", &autumn, nil, true},
		{"seasonal", "valley-oak", &empty, nil, false},
		{"seasonal", "valley-oak", nil, &shoot, false},
		{"conifer", "coast-redwood", nil, &shoot, true},
		{"conifer", "giant-sequoia", nil, &shoot, true},
		{"conifer", "spruce", nil, &shoot, false},
		{"conifer", "coast-redwood", nil, &empty, false},
		{"conifer", "coast-redwood", nil, &whole, false},
		{"conifer", "coast-redwood", &winter, nil, false},
		{"conifer", "coast-redwood", &winter, &shoot, false},
		{"fern", "fern", &autumn, nil, false},
		{"cactus", "saguaro", nil, &shoot, false},
	} {
		name, raw, err := EmbeddedPlantBook(tc.family)
		if err != nil {
			t.Fatal(err)
		}
		book, err := ReadPlantBook(tc.family, name, raw)
		if err != nil {
			t.Fatal(err)
		}
		_, err = book.New(PlantOptions{Species: tc.species, Season: tc.season, Representation: tc.representation})
		if (err == nil) != tc.valid {
			t.Errorf("family %s species %s season %v representation %v: %v", tc.family, tc.species, tc.season, tc.representation, err)
		}
	}
}

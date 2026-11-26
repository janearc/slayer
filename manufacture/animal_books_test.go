package manufacture

import "testing"

func TestAnimalBookSharesFamilyOptionRules(t *testing.T) {
	name, raw, err := EmbeddedAnimalBook("fish")
	if err != nil {
		t.Fatal(err)
	}
	book, err := ReadAnimalBook("fish", name, raw)
	if err != nil {
		t.Fatal(err)
	}
	glide, standing, bones := "glide", "standing", "bones"
	zero, school := 0, 20
	seconds := 0.
	for _, tc := range []struct {
		options AnimalOptions
		valid   bool
	}{
		{AnimalOptions{Species: "pacific-sardine", Pose: &glide}, true},
		{AnimalOptions{Species: "pacific-sardine", Count: &school}, true},
		{AnimalOptions{Species: "pacific-sardine", Pose: &bones, Count: &zero}, true},
		{AnimalOptions{Species: "pacific-sardine", Pose: &bones, Count: &school}, false},
		{AnimalOptions{Species: "pacific-sardine", Pose: &standing}, false},
		{AnimalOptions{Species: "pacific-sardine", Seconds: &seconds}, false},
	} {
		_, err := book.New(tc.options)
		if (err == nil) != tc.valid {
			t.Errorf("%+v: %v", tc.options, err)
		}
	}
	before := raw[0]
	raw[0] ^= 0xff
	_, again, err := EmbeddedAnimalBook("fish")
	if err != nil || again[0] != before {
		t.Fatalf("embedded book was mutable: %v", err)
	}
}

func TestAnimalBookWhaleOptions(t *testing.T) {
	file, raw, err := EmbeddedAnimalBook("whale")
	if err != nil {
		t.Fatal(err)
	}
	book, err := ReadAnimalBook("whale", file, raw)
	if err != nil {
		t.Fatal(err)
	}
	swim, spy, bones, volumes, blank, study := "swim", "spy-hop", "bones", "volumes", "", "study"
	zero := 0
	for _, tc := range []struct {
		options          AnimalOptions
		valid, deforming bool
	}{
		{AnimalOptions{Species: "orca"}, true, true},
		{AnimalOptions{Species: "orca", Pose: &swim}, true, true},
		{AnimalOptions{Species: "orca", Pose: &spy}, true, false},
		{AnimalOptions{Species: "orca", Representation: &bones}, true, false},
		{AnimalOptions{Species: "orca", Pose: &spy, Representation: &bones}, false, false},
		{AnimalOptions{Species: "orca", Representation: &volumes}, false, false},
		{AnimalOptions{Species: "orca", Pose: &blank}, false, false},
		{AnimalOptions{Species: "orca", Representation: &blank}, false, false},
		{AnimalOptions{Species: "orca", Count: &zero}, false, false},
		{AnimalOptions{Species: "orca", Study: &study}, false, false},
	} {
		row, err := book.New(tc.options)
		if (err == nil) != tc.valid || (err == nil && Deforms(row) != tc.deforming) {
			t.Errorf("%+v: %v", tc.options, err)
		}
	}
}

func TestAnimalBookPinnipedStudiesAndRepresentations(t *testing.T) {
	name, raw, err := EmbeddedAnimalBook("pinniped")
	if err != nil {
		t.Fatal(err)
	}
	book, err := ReadAnimalBook("pinniped", name, raw)
	if err != nil {
		t.Fatal(err)
	}
	bones, volumes, skin, study := "bones", "volumes", "skin", "detailed-wet"
	lying, swimming, empty := "lying", "swimming", ""
	zero := 0
	for _, tc := range []struct {
		name    string
		options AnimalOptions
		valid   bool
	}{
		{"default", AnimalOptions{Species: "harbor-seal"}, true},
		{"skin", AnimalOptions{Species: "harbor-seal", Pose: &swimming, Representation: &skin}, true},
		{"harbor bones", AnimalOptions{Species: "harbor-seal", Representation: &bones}, true},
		{"walrus bones", AnimalOptions{Species: "walrus", Representation: &bones}, true},
		{"walrus volumes", AnimalOptions{Species: "walrus", Pose: &lying, Representation: &volumes}, true},
		{"study", AnimalOptions{Species: "harbor-seal", Study: &study}, true},
		{"study with pose", AnimalOptions{Species: "harbor-seal", Pose: &lying, Study: &study}, false},
		{"study with representation", AnimalOptions{Species: "harbor-seal", Representation: &skin, Study: &study}, false},
		{"empty study", AnimalOptions{Species: "harbor-seal", Study: &empty}, false},
		{"explicit empty pose", AnimalOptions{Species: "walrus", Pose: &empty, Representation: &bones}, false},
		{"empty representation", AnimalOptions{Species: "walrus", Representation: &empty}, false},
		{"bones with pose", AnimalOptions{Species: "walrus", Pose: &lying, Representation: &bones}, false},
		{"volumes swimming", AnimalOptions{Species: "walrus", Pose: &swimming, Representation: &volumes}, false},
		{"study with count", AnimalOptions{Species: "harbor-seal", Count: &zero, Study: &study}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := book.New(tc.options)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v: %v", tc.valid, err)
			}
		})
	}
	for _, family := range []string{"fish", "bird", "theropod"} {
		name, raw, err := EmbeddedAnimalBook(family)
		if err != nil {
			t.Fatal(err)
		}
		other, err := ReadAnimalBook(family, name, raw)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := other.New(AnimalOptions{Species: "any", Representation: &bones}); err == nil {
			t.Errorf("%s accepted pinniped representation", family)
		}
		if _, err := other.New(AnimalOptions{Species: "any", Study: &study}); err == nil {
			t.Errorf("%s accepted pinniped study", family)
		}
	}
}

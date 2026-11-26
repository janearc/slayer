package manufacture

import (
	"bytes"
	"testing"
)

func TestPinnipedRepresentationValidatesOptions(t *testing.T) {
	book, err := ReadPinnipedProfiles("pinnipeds.yaml", pinnipedProfilesRaw)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		species, pose, representation string
		valid                         bool
	}{
		{"harbor-seal", "", "bones", true},
		{"walrus", "", "bones", true},
		{"walrus", "", "volumes", true},
		{"walrus", "lying", "volumes", true},
		{"walrus", "swimming", "skin", true},
		{"harbor-seal", "swimming", "bones", false},
		{"walrus", "swimming", "bones", false},
		{"walrus", "swimming", "volumes", false},
		{"hawaiian-monk-seal", "", "bones", false},
		{"harbor-seal", "", "volumes", false},
		{"walrus", "", "unknown", false},
		{"absent", "", "bones", false},
	} {
		t.Run(tc.species+"/"+tc.pose+"/"+tc.representation, func(t *testing.T) {
			_, err := book.NewPinnipedRepresentation(tc.species, tc.pose, tc.representation)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v: %v", tc.valid, err)
			}
		})
	}
	if _, err := (*PinnipedProfiles)(nil).NewPinnipedRepresentation("walrus", "", "bones"); err == nil {
		t.Fatal("accepted nil book")
	}
}

func TestPinnipedRepresentationsUseTheirOwnAnatomy(t *testing.T) {
	old, err := ReadPinnipedProfiles("old.yaml", pinnipedProfilesRaw)
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(pinnipedProfilesRaw, []byte("thoracic: 14"), []byte("thoracic: 15"), 1)
	changed = bytes.Replace(changed, []byte(`thoracic: {basis: read, source: "Pierce, Clack and Hutchinson 2011, Journal of Anatomy 219:502, PMC3196755", value: 14}`), []byte("thoracic: {basis: picked}"), 1)
	changed = bytes.Replace(changed, []byte("cervical: 7"), []byte("cervical: 8"), 1)
	newBook, err := ReadPinnipedProfiles("new.yaml", changed)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ species, representation string }{
		{"harbor-seal", "bones"}, {"walrus", "bones"}, {"walrus", "volumes"},
	} {
		t.Run(tc.species+"/"+tc.representation, func(t *testing.T) {
			first, err := old.NewPinnipedRepresentation(tc.species, "", tc.representation)
			if err != nil {
				t.Fatal(err)
			}
			second, err := newBook.NewPinnipedRepresentation(tc.species, "", tc.representation)
			if err != nil {
				t.Fatal(err)
			}
			one, err := Make(first, Sow(5))
			if err != nil {
				t.Fatal(err)
			}
			two, err := Make(second, Sow(5))
			if err != nil {
				t.Fatal(err)
			}
			if referenceDigest(t, one) == referenceDigest(t, two) {
				t.Fatal("anatomy edit changed no representation geometry")
			}
		})
	}
}

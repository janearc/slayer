package manufacture

import (
	"strings"
	"testing"
)

func TestBirdProfilesExternalConstructorsMatchBaseline(t *testing.T) {
	book, err := ReadBirdProfiles("birds.yaml", birdRaw)
	if err != nil {
		t.Fatal(err)
	}
	for _, species := range book.book.Species() {
		for _, stance := range birdStances {
			row, err := book.NewBird(species, stance)
			if err != nil {
				t.Fatal(err)
			}
			made, err := row.how.made(sow(7))
			if err != nil {
				t.Fatal(err)
			}
			key := species + "/" + stance
			if got, want := birdAllosaurusWitness(made.native), birdBaselineWitness[key]; got != want {
				t.Errorf("%s: %s != %s", key, got, want)
			}
		}
	}
	if _, err := book.NewGreyHeron("standing"); err != nil {
		t.Fatal(err)
	}
	if _, err := book.NewBird("no-such-bird", "standing"); err == nil {
		t.Fatal("accepted absent species")
	}
	if _, err := book.NewBird("grey-heron", "running"); err == nil {
		t.Fatal("accepted absent stance")
	}
	if _, err := (*BirdProfiles)(nil).NewGreyHeron("standing"); err == nil {
		t.Fatal("accepted nil book")
	}
}

func TestBirdProfilesRejectUnsafeBooks(t *testing.T) {
	base := string(birdRaw)
	replacements := []struct{ name, from, to string }{
		{"null figure", "length: {least: 0.60, most: 0.79", "length: {least: null, most: 0.79"},
		{"missing number", "length: {least: 0.60, most: 0.79", "length: {most: 0.79"},
		{"nonfinite", "length: {least: 0.60, most: 0.79", "length: {least: .nan, most: 0.79"},
		{"excessive count", "cervicals: {least: 17, most: 17", "cervicals: {least: 100000, most: 100000"},
		{"fractional count", "cervicals: {least: 17, most: 17", "cervicals: {least: 17.5, most: 17.5"},
		{"bad colour", "colour: [0.78, 0.78, 0.76]", "colour: [.nan, 0.78, 0.76]"},
		{"null colour", "colour: [0.78, 0.78, 0.76]", "colour: [null, 0.78, 0.76]"},
		{"merge", "name: black-faced spoonbill", "<<: {name: black-faced spoonbill}"},
		{"unknown field", "name: black-faced spoonbill", "unknown: example\n    name: black-faced spoonbill"},
		{"unknown figure", "length: {least: 0.60, most: 0.79", "unknown: {least: 0.60, most: 0.79"},
		{"missing spatula", "spatula: {least: 0.26, most: 0.26", "spatula: {least: 0, most: 0.26"},
		{"missing plume length", "plume-length: {least: 0.15, most: 0.15", "plume-length: {least: 0, most: 0.15"},
		{"collapsed leg", "tarsus: {least: 0.11, most: 0.13", "tarsus: {least: 1e-300, most: 1e-300"},
		{"tiny head share", "head: {least: 0.10, most: 0.10", "head: {least: 0.00001, most: 0.10"},
	}
	for _, tc := range replacements {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(base, tc.from) {
				t.Fatal("fixture absent")
			}
			raw := strings.Replace(base, tc.from, tc.to, 1)
			if _, err := ReadBirdProfiles("birds.yaml", []byte(raw)); err == nil {
				t.Fatal("accepted invalid book")
			}
		})
	}
	for name, raw := range map[string]string{"second document": base + "\n---\nspecies: {}\n", "malformed trailing": base + "\n---\nspecies: [\n"} {
		t.Run(name, func(t *testing.T) {
			if _, err := ReadBirdProfiles("birds.yaml", []byte(raw)); err == nil {
				t.Fatal("accepted trailing document")
			}
		})
	}
	if _, err := ReadBirdProfiles("birds.yaml", make([]byte, (1<<20)+1)); err == nil {
		t.Fatal("accepted oversized book")
	}
}

func TestBirdProfilesAcceptMappingAliases(t *testing.T) {
	text := strings.Replace(string(birdRaw), "    figures:\n", "    figures: &first-figures\n", 1)
	text = strings.Replace(text, "    plumage:\n", "    plumage: &first-plumage\n", 1)
	egrets := strings.Index(text, "  little-egret:\n")
	if egrets < 0 {
		t.Fatal("missing egret fixture")
	}
	figures := egrets + strings.Index(text[egrets:], "    figures:\n")
	plumage := figures + strings.Index(text[figures:], "    plumage:\n")
	next := plumage + strings.Index(text[plumage:], "  grey-heron:\n")
	if figures < egrets || plumage < figures || next < plumage {
		t.Fatal("missing figure, plumage or species fixture")
	}
	text = text[:figures] + "    figures: *first-figures\n    plumage: *first-plumage\n" + text[next:]
	book, err := ReadBirdProfiles("aliases.yaml", []byte(text))
	if err != nil {
		t.Fatalf("mapping aliases: %v", err)
	}
	if _, err := book.NewLittleEgret("standing"); err != nil {
		t.Fatal(err)
	}
}

func TestBirdProfilesRetainTheirOwnData(t *testing.T) {
	original := append([]byte(nil), birdRaw...)
	a, err := ReadBirdProfiles("a.yaml", original)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(original), "length: {least: 0.60, most: 0.79", "length: {least: 0.70, most: 0.89", 1)
	if changed == string(original) {
		t.Fatal("fixture absent")
	}
	b, err := ReadBirdProfiles("b.yaml", []byte(changed))
	if err != nil {
		t.Fatal(err)
	}
	first, _ := a.NewBlackFacedSpoonbill("standing")
	second, _ := b.NewBlackFacedSpoonbill("standing")
	for i := range original {
		original[i] = 'x'
	}
	one, err := first.how.made(sow(7))
	if err != nil {
		t.Fatal(err)
	}
	two, err := second.how.made(sow(7))
	if err != nil {
		t.Fatal(err)
	}
	if birdAllosaurusWitness(one.native) == birdAllosaurusWitness(two.native) {
		t.Fatal("distinct books produced same geometry")
	}
	if got, want := birdAllosaurusWitness(one.native), birdBaselineWitness["black-faced-spoonbill/standing"]; got != want {
		t.Fatalf("original book changed: %s != %s", got, want)
	}
}

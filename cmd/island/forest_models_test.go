package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture"
)

func TestForestResolvesCheckedPlantsAndOtherKinds(t *testing.T) {
	books, err := forestBooks()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		family int
		name   string
	}{
		{0, "valley-oak"}, {1, "coast-redwood"}, {2, "fern"}, {3, "saguaro"},
	} {
		if !books[tc.family].book.HasSpecies(tc.name) {
			t.Fatalf("missing %s in checked family", tc.name)
		}
		want, err := books[tc.family].book.New(manufacture.PlantOptions{Species: tc.name})
		if err != nil {
			t.Fatal(err)
		}
		got, err := forestPlant(tc.name)
		if err != nil {
			t.Fatal(err)
		}
		detail := "hull"
		if tc.name == "fern" {
			detail = "quick"
		}
		wantMade, err := manufacture.MakeWithComplexity(want, manufacture.Sow(7), detail)
		if err != nil {
			t.Fatal(err)
		}
		gotMade, err := manufacture.MakeWithComplexity(got, manufacture.Sow(7), detail)
		if err != nil || gotMade.Bounds() != wantMade.Bounds() || gotMade.PartCount() != wantMade.PartCount() {
			t.Fatalf("%s: forest construction differs from checked book: %v", tc.name, err)
		}
	}
	if _, err := forestPlant("can"); err != nil {
		t.Fatalf("non-plant mixture kind was refused: %v", err)
	}
	if _, err := forestPlant("dragon"); err == nil {
		t.Fatal("unknown forest member accepted")
	}
}

func TestForestAndSitePrototypeKeysRemainDistinct(t *testing.T) {
	site := roomySite(t)
	ordinary := prototypeKey{kind: "can", detail: "quick", seed: 3}
	forest := ordinary
	forest.forest = true
	if _, err := site.load(ordinary); err != nil {
		t.Fatal(err)
	}
	if _, err := site.load(forest); err != nil {
		t.Fatal(err)
	}
	if len(site.store.values) != 2 {
		t.Fatal("forest and ordinary specimen shared a prototype key")
	}
}

func TestForestRetainsCheckedLooseShootSelectors(t *testing.T) {
	books, err := forestBooks()
	if err != nil {
		t.Fatal(err)
	}
	shoot := "shoot"
	for _, species := range []string{"coast-redwood", "giant-sequoia"} {
		want, err := books[1].book.New(manufacture.PlantOptions{Species: species, Representation: &shoot})
		if err != nil {
			t.Fatal(err)
		}
		got, err := forestPlant(species + "-shoot")
		if err != nil {
			t.Fatal(err)
		}
		wantMade, err := manufacture.MakeWithComplexity(want, manufacture.Sow(5), "quick")
		if err != nil {
			t.Fatal(err)
		}
		gotMade, err := manufacture.MakeWithComplexity(got, manufacture.Sow(5), "quick")
		if err != nil || gotMade.Bounds() != wantMade.Bounds() || gotMade.PartCount() != wantMade.PartCount() {
			t.Fatalf("%s shoot changed: %v", species, err)
		}
	}
}

func TestForestRefusesSpeciesAcrossTwoBooks(t *testing.T) {
	books, err := forestBooks()
	if err != nil {
		t.Fatal(err)
	}
	name, raw, err := manufacture.EmbeddedPlantBook("fern")
	if err != nil {
		t.Fatal(err)
	}
	duplicate := bytes.Replace(raw, []byte("kind: fern"), []byte("kind: valley-oak"), 1)
	if bytes.Equal(raw, duplicate) {
		t.Fatal("missing fern profile fixture")
	}
	book, err := manufacture.ReadPlantBook("fern", name, duplicate)
	if err != nil {
		t.Fatal(err)
	}
	mixed := append(append([]forestBook(nil), books...), forestBook{"fern", book})
	if _, err := forestPlantFrom(mixed, "valley-oak"); err == nil || !strings.Contains(err.Error(), "ambiguous forest plant") {
		t.Fatalf("duplicate species was not refused: %v", err)
	}
}

func TestForestShootRefusesMissingSpeciesWithoutFallback(t *testing.T) {
	name, raw, err := manufacture.EmbeddedPlantBook("conifer")
	if err != nil {
		t.Fatal(err)
	}
	missing := bytes.Replace(raw, []byte(`kind: "coast-redwood"`), []byte(`kind: "missing-redwood"`), 1)
	if bytes.Equal(raw, missing) {
		t.Fatal("missing conifer fixture")
	}
	book, err := manufacture.ReadPlantBook("conifer", name, missing)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := forestPlantFrom([]forestBook{{"conifer", book}}, "coast-redwood-shoot"); err == nil || !strings.Contains(err.Error(), "coast-redwood") {
		t.Fatalf("missing shoot species did not reach the book: %v", err)
	}
}

func TestForestRefusesShootSpeciesNameCollision(t *testing.T) {
	books, err := forestBooks()
	if err != nil {
		t.Fatal(err)
	}
	name, raw, err := manufacture.EmbeddedPlantBook("fern")
	if err != nil {
		t.Fatal(err)
	}
	collision := bytes.Replace(raw, []byte("kind: fern"), []byte("kind: coast-redwood-shoot"), 1)
	if bytes.Equal(raw, collision) {
		t.Fatal("missing fern fixture")
	}
	book, err := manufacture.ReadPlantBook("fern", name, collision)
	if err != nil {
		t.Fatal(err)
	}
	mixed := append(append([]forestBook(nil), books...), forestBook{"fern", book})
	if _, err := forestPlantFrom(mixed, "coast-redwood-shoot"); err == nil || !strings.Contains(err.Error(), "ambiguous forest plant") {
		t.Fatalf("shoot selector collision was not refused: %v", err)
	}
}

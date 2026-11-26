package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture"
)

const testAnimalEntry = `
  - kind: model-tuna
    group: Fish
    about: A profile-backed tuna.
    notes: A checked model book.
    details: [quick]
    animal: {family: fish, file: fish.yaml, species: pacific-bluefin-tuna, pose: glide}
`

func withAnimalMenu(t *testing.T) {
	t.Helper()
	before := menuRaw
	t.Cleanup(func() { menuRaw = before })
	menuRaw = append(bytes.Clone(menuRaw), testAnimalEntry...)
}

func writeEmbeddedModelBooks(t *testing.T, dir string) {
	t.Helper()
	for _, family := range []string{"fish", "bird", "pinniped", "theropod", "squid", "whale", "reptile", "snake", "sauropod"} {
		name, raw, err := manufacture.EmbeddedAnimalBook(family)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, family := range []string{"seasonal", "conifer", "fern", "cactus"} {
		name, raw, err := manufacture.EmbeddedPlantBook(family)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	name, raw, err := manufacture.EmbeddedAlgaBook("kelp")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), raw, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestCatalogueAnimalBooksAreFrozenAndDoNotFallBack(t *testing.T) {
	withAnimalMenu(t)
	embedded, err := readMenu()
	if err != nil {
		t.Fatal(err)
	}
	rows, err := embedded.selectEntries([]string{"model-tuna"})
	if err != nil || len(rows) != 1 || rows[0].source == "" {
		t.Fatalf("embedded profile: %v", err)
	}
	if _, err := manufacture.KindNamed("model-tuna"); err == nil {
		t.Fatal("catalogue ID became a registered alias")
	}

	dir := t.TempDir()
	writeEmbeddedModelBooks(t, dir)
	_, raw, err := manufacture.EmbeddedAnimalBook("fish")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "fish.yaml")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	first, err := parseMenuWithModels("menu.yaml", menuRaw, dir)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := first.selectEntries([]string{"model-tuna"})
	if err != nil {
		t.Fatal(err)
	}
	old := selected[0]
	changed := bytes.Replace(raw, []byte("length: {least: 1.5, most: 2.4"), []byte("length: {least: 1.8, most: 2.4"), 1)
	if bytes.Equal(raw, changed) {
		t.Fatal("missing length fixture")
	}
	if err := os.WriteFile(path, changed, 0600); err != nil {
		t.Fatal(err)
	}
	second, err := parseMenuWithModels("menu.yaml", menuRaw, dir)
	if err != nil {
		t.Fatal(err)
	}
	newRows, err := second.selectEntries([]string{"model-tuna"})
	if err != nil {
		t.Fatal(err)
	}
	if old.source == newRows[0].source {
		t.Fatal("model bytes did not change card identity")
	}
	a, err := old.kind()
	if err != nil {
		t.Fatal(err)
	}
	b, err := newRows[0].kind()
	if err != nil {
		t.Fatal(err)
	}
	one, err := manufacture.MakeWithComplexity(a, manufacture.Sow(5), "quick")
	if err != nil {
		t.Fatal(err)
	}
	two, err := manufacture.MakeWithComplexity(b, manufacture.Sow(5), "quick")
	if err != nil {
		t.Fatal(err)
	}
	if one.Bounds() == two.Bounds() {
		t.Fatal("checked snapshot ignored the new model")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := parseMenuWithModels("menu.yaml", menuRaw, dir); err == nil {
		t.Fatal("missing explicit model silently fell back")
	}
	if _, err := old.kind(); err != nil {
		t.Fatalf("previous checked snapshot changed: %v", err)
	}
	if err := os.WriteFile(path, changed, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "snake-profiles.yaml")); err != nil {
		t.Fatal(err)
	}
	if _, err := parseMenuWithModels("menu.yaml", menuRaw, dir); err == nil || !strings.Contains(err.Error(), "snake-profiles.yaml") {
		t.Fatalf("missing explicit snake book did not fail closed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "snake-profiles.yaml"), mustAnimalBook(t, "snake"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "sauropod-profiles.yaml")); err != nil {
		t.Fatal(err)
	}
	if _, err := parseMenuWithModels("menu.yaml", menuRaw, dir); err == nil || !strings.Contains(err.Error(), "sauropod-profiles.yaml") {
		t.Fatalf("missing explicit sauropod book did not fail closed: %v", err)
	}
}

func TestCatalogueAnimalResumeTracksModelAndDescriptor(t *testing.T) {
	withAnimalMenu(t)
	dir := t.TempDir()
	writeEmbeddedModelBooks(t, dir)
	_, raw, err := manufacture.EmbeddedAnimalBook("fish")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fish.yaml"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "book")
	run := func(draws int) {
		t.Helper()
		result := runDrawing(t, drawsEvery(t, draws, nil), "render", "model-tuna", "--models", dir, "--out", destination)
		if result.status != successfulExit {
			t.Fatalf("render: %s", result.wrong)
		}
	}
	run(1)
	first := cardOf(t, loadBook(t, destination), "model-tuna")
	if first.Source == "" || first.Settings == "" {
		t.Fatal("card lacks source or settings identity")
	}
	run(0)
	changed := bytes.Replace(raw, []byte("length: {least: 1.5, most: 2.4"), []byte("length: {least: 1.8, most: 2.4"), 1)
	if err := os.WriteFile(filepath.Join(dir, "fish.yaml"), changed, 0600); err != nil {
		t.Fatal(err)
	}
	run(1)
	second := cardOf(t, loadBook(t, destination), "model-tuna")
	if second.Source == first.Source || second.Settings != first.Settings {
		t.Fatal("model change did not invalidate only source identity")
	}
	menuRaw = bytes.Replace(menuRaw, []byte(testAnimalEntry), []byte(strings.Replace(testAnimalEntry, "pose: glide", "pose: swim", 1)), 1)
	run(1)
	third := cardOf(t, loadBook(t, destination), "model-tuna")
	if third.Source == second.Source {
		t.Fatal("pose did not change source identity")
	}
	// Old receipts without a source digest cannot certify a model-backed card.
	held := loadBook(t, destination)
	for i := range held.Cards {
		if held.Cards[i].Entry.Kind == "model-tuna" {
			held.Cards[i].Source = ""
		}
	}
	encoded, err := json.Marshal(held)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "catalogue.json"), encoded, 0600); err != nil {
		t.Fatal(err)
	}
	run(1)
}

func TestCatalogueAnimalRejectsUnsafePathsAndOptions(t *testing.T) {
	for _, spec := range []animalSpec{
		{Family: "fish", File: "../fish.yaml", Species: "pacific-bluefin-tuna"},
		{Family: "fish", File: "/tmp/fish.yaml", Species: "pacific-bluefin-tuna"},
		{Family: "fish", File: "other.yaml", Species: "pacific-bluefin-tuna"},
		{Family: "fish", File: "fish.yaml", Species: "pacific-bluefin-tuna", Pose: "standing"},
	} {
		row := entry{Kind: "example", Animal: &spec}
		if err := row.bindAnimal(&modelBooks{}); err == nil {
			t.Errorf("accepted %+v", spec)
		}
	}
	if _, _, err := manufacture.EmbeddedAnimalBook("unknown"); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Errorf("unknown family: %v", err)
	}
}

func TestCatalogueExplicitAnimalOverridesMatchingRegisteredKind(t *testing.T) {
	const name = "can"
	from := "  - kind: " + name + "\n"
	to := from + "    animal: {family: fish, file: fish.yaml, species: california-yellowtail, pose: glide}\n"
	raw := bytes.Replace(menuRaw, []byte(from), []byte(to), 1)
	if bytes.Equal(raw, menuRaw) {
		t.Fatal("missing registered kind fixture")
	}
	menu, err := parseMenu("menu.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := menu.selectEntries([]string{name})
	if err != nil || len(entries) != 1 {
		t.Fatalf("selected entries: %v", err)
	}
	profile, err := entries[0].kind()
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := manufacture.KindNamed(name)
	if err != nil {
		t.Fatal(err)
	}
	a, err := manufacture.MakeWithComplexity(profile, manufacture.Sow(5), "quick")
	if err != nil {
		t.Fatal(err)
	}
	b, err := manufacture.MakeWithComplexity(legacy, manufacture.Sow(5), "quick")
	if err != nil {
		t.Fatal(err)
	}
	if a.Bounds() == b.Bounds() {
		t.Fatal("registered kind overrode the explicit animal")
	}
	invalid := bytes.Replace(raw, []byte("species: california-yellowtail"), []byte("species: no-such-fish"), 1)
	if _, err := parseMenu("bad.yaml", invalid); err == nil {
		t.Fatal("invalid animal silently fell back to registered kind")
	}
}

func TestCataloguePinnipedRepresentationsAndStudies(t *testing.T) {
	bones, volumes, skin, study, badStudy := "bones", "volumes", "skin", "detailed-wet", "not-a-study"
	for _, tc := range []struct {
		name  string
		spec  animalSpec
		valid bool
	}{
		{"bones", animalSpec{Family: "pinniped", File: "pinniped_profiles.yaml", Species: "harbor-seal", Representation: &bones}, true},
		{"volumes", animalSpec{Family: "pinniped", File: "pinniped_profiles.yaml", Species: "walrus", Pose: "lying", Representation: &volumes}, true},
		{"study", animalSpec{Family: "pinniped", File: "pinniped_profiles.yaml", Species: "harbor-seal", Study: &study}, true},
		{"study and pose", animalSpec{Family: "pinniped", File: "pinniped_profiles.yaml", Species: "harbor-seal", Pose: "hauled", Study: &study}, false},
		{"study and representation", animalSpec{Family: "pinniped", File: "pinniped_profiles.yaml", Species: "harbor-seal", Representation: &skin, Study: &study}, false},
		{"missing study", animalSpec{Family: "pinniped", File: "pinniped_profiles.yaml", Species: "harbor-seal", Study: &badStudy}, false},
		{"bird with study", animalSpec{Family: "bird", File: "birds.yaml", Species: "grey-heron", Study: &study}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entry := entry{Kind: "sample", Animal: &tc.spec}
			err := entry.bindAnimal(&modelBooks{})
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v: %v", tc.valid, err)
			}
			if err == nil {
				if entry.source == "" {
					t.Fatal("missing model identity")
				}
				if _, err := entry.kind(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
	books := &modelBooks{}
	a := entry{Kind: "sample", Animal: &animalSpec{Family: "pinniped", File: "pinniped_profiles.yaml", Species: "harbor-seal", Representation: &bones}}
	b := entry{Kind: "sample", Animal: &animalSpec{Family: "pinniped", File: "pinniped_profiles.yaml", Species: "harbor-seal", Study: &study}}
	if err := a.bindAnimal(books); err != nil {
		t.Fatal(err)
	}
	if err := b.bindAnimal(books); err != nil {
		t.Fatal(err)
	}
	if a.source == b.source {
		t.Fatal("distinct options shared card identity")
	}
}

func TestCatalogueMigratedPinnipedCardsKeepTheirOptions(t *testing.T) {
	menu, err := readMenu()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ card, species, pose, representation, study string }{
		{"harbor-seal", "harbor-seal", "", "", "detailed-wet"},
		{"harbor-seal-dry", "harbor-seal", "", "", "detailed-dry"},
		{"harbor-seal-hauled", "harbor-seal", "hauled", "", ""},
		{"harbor-seal-swimming", "harbor-seal", "swimming", "", ""},
		{"harbor-seal-bones", "harbor-seal", "", "bones", ""},
		{"hawaiian-monk-seal-hauled", "hawaiian-monk-seal", "hauled", "", ""},
		{"bay-seal", "harbor-seal", "", "", "bay"},
		{"walrus-fresh-5", "walrus", "lying", "", ""},
		{"walrus-fresh-5-hauled-out", "walrus", "hauled-out", "", ""},
		{"walrus-fresh-5-swimming", "walrus", "swimming", "", ""},
		{"walrus-fresh-5-skeleton", "walrus", "", "bones", ""},
		{"walrus-fresh-5-volumes", "walrus", "", "volumes", ""},
	} {
		t.Run(tc.card, func(t *testing.T) {
			rows, err := menu.selectEntries([]string{tc.card})
			if err != nil || len(rows) != 1 {
				t.Fatalf("card: %v", err)
			}
			spec := rows[0].Animal
			if spec == nil || spec.Family != "pinniped" || spec.Species != tc.species || spec.Pose != tc.pose {
				t.Fatalf("wrong species or pose: %+v", spec)
			}
			value := func(s *string) string {
				if s == nil {
					return ""
				}
				return *s
			}
			if value(spec.Representation) != tc.representation || value(spec.Study) != tc.study {
				t.Fatalf("wrong representation or study: %+v", spec)
			}
			if _, err := rows[0].kind(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func mustAnimalBook(t *testing.T, family string) []byte {
	t.Helper()
	_, raw, err := manufacture.EmbeddedAnimalBook(family)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

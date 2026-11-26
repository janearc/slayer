package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/manufacture"
)

func TestCatalogueAlgaMenuSelectsSpeciesAndWater(t *testing.T) {
	menu, err := readMenu()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, condition string
		forest          bool
	}{
		{"giant-kelp", "moderate", false}, {"kelp-forest", "moderate", true},
		{"giant-kelp-slack", "slack", false}, {"giant-kelp-moderate", "moderate", false},
		{"giant-kelp-surge", "surge", false}, {"kelp-forest-slack", "slack", true},
		{"kelp-forest-moderate", "moderate", true}, {"kelp-forest-surge", "surge", true},
	} {
		rows, err := menu.selectEntries([]string{tc.name})
		if err != nil || len(rows) != 1 {
			t.Fatalf("%s: %v", tc.name, err)
		}
		got := rows[0].Alga
		if got == nil || got.Family != "kelp" || got.File != "kelp.yaml" ||
			got.Species != "macrocystis-pyrifera" || got.Condition != tc.condition || got.Forest != tc.forest ||
			rows[0].checked == nil || rows[0].source == "" {
			t.Fatalf("%s lost its checked alga recipe: %+v", tc.name, got)
		}
		if _, err := rows[0].kind(); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
	}
}

func TestCatalogueAlgaFreezesBooksAndIdentity(t *testing.T) {
	name, raw, err := manufacture.EmbeddedAlgaBook("kelp")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	spec := algaSpec{Family: "kelp", File: name, Species: "macrocystis-pyrifera", Condition: "moderate"}
	first := entry{Kind: "sample", Alga: &spec}
	if err := first.bindAlga(&modelBooks{root: dir}); err != nil {
		t.Fatal(err)
	}
	old, err := measureEntry(first)
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(raw, []byte("depth: {least: 12, most: 12"), []byte("depth: {least: 8, most: 8"), 1)
	if bytes.Equal(changed, raw) {
		t.Fatal("depth source missing")
	}
	if err := os.WriteFile(path, changed, 0600); err != nil {
		t.Fatal(err)
	}
	second := entry{Kind: "sample", Alga: &spec}
	if err := second.bindAlga(&modelBooks{root: dir}); err != nil {
		t.Fatal(err)
	}
	fresh, err := measureEntry(second)
	if err != nil {
		t.Fatal(err)
	}
	if old.Source == "" || old.Source == fresh.Source || old.Bounds == fresh.Bounds {
		t.Fatalf("changed book did not change identity and geometry: %+v %+v", old, fresh)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	again, err := measureEntry(first)
	if err != nil || again.Bounds != old.Bounds || again.Source != old.Source {
		t.Fatalf("frozen book changed after deletion: %+v %v", again, err)
	}
	if err := (&entry{Kind: "sample", Alga: &spec}).bindAlga(&modelBooks{root: dir}); err == nil {
		t.Fatal("missing explicit book fell back to embedded data")
	}
}

func TestCatalogueAlgaRefusesInvalidOptionsAndMixedDescriptors(t *testing.T) {
	for _, bad := range []algaSpec{
		{Family: "unknown", File: "kelp.yaml", Species: "macrocystis-pyrifera"},
		{Family: "kelp", File: "../kelp.yaml", Species: "macrocystis-pyrifera"},
		{Family: "kelp", File: "kelp.yaml", Species: "unknown"},
		{Family: "kelp", File: "kelp.yaml", Species: "macrocystis-pyrifera", Condition: "unknown"},
	} {
		if err := (&entry{Kind: "sample", Alga: &bad}).bindAlga(&modelBooks{}); err == nil {
			t.Errorf("accepted %+v", bad)
		}
	}
	for _, descriptor := range []string{
		"animal: {family: fish, file: fish.yaml, species: pacific-sardine}",
		"plant: {family: fern, file: fern-profiles.yaml, species: fern}",
	} {
		original := []byte("  - kind: giant-kelp\n")
		mixed := append(bytes.Clone(original), []byte("    "+descriptor+"\n")...)
		raw := bytes.Replace(menuRaw, original, mixed, 1)
		if bytes.Equal(raw, menuRaw) {
			t.Fatal("kelp card missing")
		}
		if _, err := parseMenu("menu.yaml", raw); err == nil {
			t.Errorf("accepted alga with %s", descriptor)
		}
	}
}

func TestCatalogueAlgaResumeTracksSourceAndCondition(t *testing.T) {
	previous := menuRaw
	t.Cleanup(func() { menuRaw = previous })
	dir := t.TempDir()
	writeEmbeddedModelBooks(t, dir)
	path := filepath.Join(dir, "kelp.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "book")
	run := func(draws int) {
		t.Helper()
		result := runDrawing(t, drawsEvery(t, draws, nil), "render", "giant-kelp", "--models", dir, "--out", destination)
		if result.status != successfulExit {
			t.Fatalf("render: %s", result.wrong)
		}
	}
	run(1)
	first := cardOf(t, loadBook(t, destination), "giant-kelp")
	run(0)
	changed := append(bytes.Clone(raw), []byte("\n# a distinct checked source\n")...)
	if err := os.WriteFile(path, changed, 0600); err != nil {
		t.Fatal(err)
	}
	run(1)
	second := cardOf(t, loadBook(t, destination), "giant-kelp")
	if second.Source == "" || second.Source == first.Source || second.Settings != first.Settings {
		t.Fatal("changed source did not invalidate the card")
	}
	old := []byte("  - kind: giant-kelp\n    alga: {family: kelp, file: kelp.yaml, species: macrocystis-pyrifera, condition: moderate}\n")
	new := bytes.Replace(old, []byte("condition: moderate"), []byte("condition: slack"), 1)
	menuRaw = bytes.Replace(menuRaw, old, new, 1)
	if bytes.Equal(menuRaw, previous) {
		t.Fatal("could not change the descriptor fixture")
	}
	run(1)
	third := cardOf(t, loadBook(t, destination), "giant-kelp")
	if third.Source == second.Source {
		t.Fatal("changed water condition did not invalidate the card")
	}
}

func TestInventoryReportsAlgaDescriptor(t *testing.T) {
	menu, err := readMenu()
	if err != nil {
		t.Fatal(err)
	}
	rows, err := menu.selectEntries([]string{"giant-kelp-slack"})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := writeInventory(&out, rows); err != nil {
		t.Fatal(err)
	}
	var data []struct {
		Alga   *algaSpec `json:"alga"`
		Source string    `json:"profile-identity-sha256"`
	}
	if err := json.Unmarshal(out.Bytes(), &data); err != nil || len(data) != 1 || data[0].Alga == nil || data[0].Alga.Condition != "slack" || data[0].Source == "" {
		t.Fatalf("missing alga inventory identity: %s (%v)", out.String(), err)
	}
}

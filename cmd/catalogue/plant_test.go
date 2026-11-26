package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

// shootFoliagePoints reads the actual shared foliage mesh, not the physical
// envelope, which intentionally need not change with needle width.
func shootFoliagePoints(t *testing.T, row manufacture.Kind) []maths.Vec {
	t.Helper()
	made, err := manufacture.MakeWithComplexity(row, manufacture.Sow(6), "quick")
	if err != nil {
		t.Fatal(err)
	}
	things, err := made.Things(maths.Local())
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range things {
		if one.Material == nil || one.Material.Name != "redwood foliage" {
			continue
		}
		group, ok := one.Shape.(*shape.MeshGroup)
		if !ok || group.MemberCount() == 0 {
			t.Fatal("shoot has no foliage mesh group")
		}
		leaves, ok := group.Member(0).Mesh.(*mesh.Mesh)
		if !ok {
			t.Fatal("shoot foliage is not a mesh")
		}
		return leaves.Points()
	}
	t.Fatal("shoot has no foliage")
	return nil
}

func TestCataloguePlantBooksFreezeGeometryAndSource(t *testing.T) {
	name, raw, err := manufacture.EmbeddedPlantBook("conifer")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	spec := plantSpec{Family: "conifer", File: name, Species: "coast-redwood"}
	shoot := "shoot"
	spec.Representation = &shoot
	first := entry{Kind: "sample-shoot", Plant: &spec}
	if err := first.bindPlant(&modelBooks{root: dir}); err != nil {
		t.Fatal(err)
	}
	old, err := first.kind()
	if err != nil {
		t.Fatal(err)
	}
	oldPoints := shootFoliagePoints(t, old)
	changed := bytes.Replace(raw, []byte("    width: 0.002\n"), []byte("    width: 0.003\n"), 1)
	if bytes.Equal(raw, changed) {
		t.Fatal("missing foliage fixture")
	}
	if err := os.WriteFile(path, changed, 0600); err != nil {
		t.Fatal(err)
	}
	second := entry{Kind: "sample-shoot", Plant: &spec}
	if err := second.bindPlant(&modelBooks{root: dir}); err != nil {
		t.Fatal(err)
	}
	fresh, err := second.kind()
	if err != nil {
		t.Fatal(err)
	}
	newPoints := shootFoliagePoints(t, fresh)
	if first.source == "" || first.source == second.source {
		t.Fatal("changed source did not change identity")
	}
	if reflect.DeepEqual(oldPoints, newPoints) {
		t.Fatal("changed foliage did not change mesh vertices")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	frozen, err := first.kind()
	if err != nil {
		t.Fatal(err)
	}
	if again := shootFoliagePoints(t, frozen); !reflect.DeepEqual(again, oldPoints) {
		t.Fatal("checked specimen changed after source deletion")
	}
	missing := entry{Kind: "sample-shoot", Plant: &spec}
	if err := missing.bindPlant(&modelBooks{root: dir}); err == nil {
		t.Fatal("missing explicit model fell back to embedded book")
	}
}

func TestCataloguePlantOptionsAndSourcesAreChecked(t *testing.T) {
	shoot, empty, winter := "shoot", "", "winter"
	for _, tc := range []struct {
		spec  plantSpec
		valid bool
	}{
		{plantSpec{Family: "conifer", File: "conifer-profiles.yaml", Species: "giant-sequoia", Representation: &shoot}, true},
		{plantSpec{Family: "seasonal", File: "plant-profiles.yaml", Species: "valley-oak", Season: &winter}, true},
		{plantSpec{Family: "conifer", File: "../conifer-profiles.yaml", Species: "coast-redwood"}, false},
		{plantSpec{Family: "conifer", File: "other.yaml", Species: "coast-redwood"}, false},
		{plantSpec{Family: "conifer", File: "conifer-profiles.yaml", Species: "spruce", Representation: &shoot}, false},
		{plantSpec{Family: "fern", File: "fern-profiles.yaml", Species: "fern", Season: &winter}, false},
		{plantSpec{Family: "fern", File: "fern-profiles.yaml", Species: "fern", Representation: &empty}, false},
	} {
		row := entry{Kind: "sample", Plant: &tc.spec}
		err := row.bindPlant(&modelBooks{})
		if (err == nil) != tc.valid {
			t.Errorf("%+v: %v", tc.spec, err)
		}
	}
}

func TestCataloguePlantDescriptorChangesIdentity(t *testing.T) {
	season := "winter"
	one := entry{Kind: "sample", Plant: &plantSpec{Family: "seasonal", File: "plant-profiles.yaml", Species: "valley-oak"}}
	two := entry{Kind: "sample", Plant: &plantSpec{Family: "seasonal", File: "plant-profiles.yaml", Species: "valley-oak", Season: &season}}
	books := &modelBooks{}
	if err := one.bindPlant(books); err != nil {
		t.Fatal(err)
	}
	if err := two.bindPlant(books); err != nil {
		t.Fatal(err)
	}
	if one.source == "" || two.source == "" || one.source == two.source {
		t.Fatal("season failed to change source identity")
	}
}

func TestMigratedPlantMenuKeepsSpeciesAndRepresentations(t *testing.T) {
	menu, err := readMenu()
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, row := range menu.Entries {
		if row.Plant == nil {
			continue
		}
		counts[row.Plant.Family]++
		if row.Plant.Species != strings.TrimSuffix(row.Kind, "-shoot") || row.checked == nil || row.source == "" {
			t.Fatalf("wrong or unchecked species on card %s: %+v", row.Kind, row.Plant)
		}
		files := map[string]string{"seasonal": "plant-profiles.yaml", "conifer": "conifer-profiles.yaml", "fern": "fern-profiles.yaml", "cactus": "cactus-profiles.yaml"}
		if row.Plant.File != files[row.Plant.Family] {
			t.Fatalf("wrong model book on %s", row.Kind)
		}
		shoot := strings.HasSuffix(row.Kind, "-shoot")
		if shoot != (row.Plant.Representation != nil && *row.Plant.Representation == "shoot") {
			t.Fatalf("wrong representation on %s", row.Kind)
		}
	}
	if counts["seasonal"] != 18 || counts["conifer"] != 6 || counts["fern"] != 1 || counts["cactus"] != 1 {
		t.Fatalf("wrong plant counts: %v", counts)
	}
	for _, tc := range []struct{ id, family, species, representation string }{
		{"valley-oak", "seasonal", "valley-oak", ""},
		{"fern", "fern", "fern", ""},
		{"saguaro", "cactus", "saguaro", ""},
		{"coast-redwood-shoot", "conifer", "coast-redwood", "shoot"},
		{"giant-sequoia-shoot", "conifer", "giant-sequoia", "shoot"},
	} {
		entries, err := menu.selectEntries([]string{tc.id})
		if err != nil || len(entries) != 1 {
			t.Fatalf("%s selection: %v", tc.id, err)
		}
		p := entries[0].Plant
		if p == nil {
			t.Fatalf("%s has no profile", tc.id)
		}
		representation := ""
		if p.Representation != nil {
			representation = *p.Representation
		}
		if p.Family != tc.family || p.Species != tc.species || representation != tc.representation {
			t.Fatalf("%s: %+v", tc.id, p)
		}
		row, err := entries[0].kind()
		if err != nil {
			t.Fatal(err)
		}
		made, err := manufacture.MakeWithComplexity(row, manufacture.Sow(5), "quick")
		if err != nil || made.PartCount() == 0 {
			t.Fatalf("%s did not build: %v", tc.id, err)
		}
	}
}

const testPlantEntry = `
  - kind: model-shoot
    group: Foliage
    about: A profile-backed conifer shoot.
    notes: A checked model book.
    details: [quick]
    plant: {family: conifer, file: conifer-profiles.yaml, species: coast-redwood, representation: shoot}
`

func TestCataloguePlantResumeTracksModelAndDescriptor(t *testing.T) {
	before := menuRaw
	t.Cleanup(func() { menuRaw = before })
	menuRaw = append(bytes.Clone(menuRaw), testPlantEntry...)
	dir := t.TempDir()
	writeEmbeddedModelBooks(t, dir)
	path := filepath.Join(dir, "conifer-profiles.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "book")
	run := func(draws int) {
		t.Helper()
		result := runDrawing(t, drawsEvery(t, draws, nil), "render", "model-shoot", "--models", dir, "--out", destination)
		if result.status != successfulExit {
			t.Fatalf("render: %s", result.wrong)
		}
	}
	run(1)
	first := cardOf(t, loadBook(t, destination), "model-shoot")
	if first.Source == "" {
		t.Fatal("missing model identity")
	}
	run(0)
	changed := bytes.Replace(raw, []byte("    width: 0.002\n"), []byte("    width: 0.003\n"), 1)
	if bytes.Equal(raw, changed) {
		t.Fatal("missing foliage fixture")
	}
	if err := os.WriteFile(path, changed, 0600); err != nil {
		t.Fatal(err)
	}
	run(1)
	second := cardOf(t, loadBook(t, destination), "model-shoot")
	if second.Source == first.Source || second.Settings != first.Settings {
		t.Fatal("model bytes did not invalidate source identity")
	}
	changedEntry := strings.Replace(testPlantEntry, "species: coast-redwood", "species: giant-sequoia", 1)
	menuRaw = bytes.Replace(menuRaw, []byte(testPlantEntry), []byte(changedEntry), 1)
	run(1)
	third := cardOf(t, loadBook(t, destination), "model-shoot")
	if third.Source == second.Source {
		t.Fatal("species did not invalidate source identity")
	}
}

func TestCatalogueRejectsMixedPlantAndAnimalDescriptors(t *testing.T) {
	original := "  - kind: valley-oak\n"
	mixed := original + "    animal: {family: fish, file: fish.yaml, species: pacific-sardine}\n"
	raw := bytes.Replace(menuRaw, []byte(original), []byte(mixed), 1)
	if bytes.Equal(raw, menuRaw) {
		t.Fatal("missing test card")
	}
	if _, err := parseMenu("menu.yaml", raw); err == nil {
		t.Fatal("accepted both animal and plant descriptors")
	}
}

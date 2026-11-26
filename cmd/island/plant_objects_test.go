package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture"
)

func plantObjectScene(objects string) string {
	return minimalScene + `site:
  at: {longitude-deg: -155.5, latitude-deg: 19.5}
  light-exposure: 1
  objects:
` + objects + `  budget: {cells: 10, candidates: 10, instances: 200000, prototype-faces: 2000000}
`
}

func plantObject(name, descriptor string) string {
	return fmt.Sprintf("    - {name: %s, %s, seed: 5, detail: hull, east-metres: 0, south-metres: 0, above-metres: 0, heading-deg: 0}\n", name, descriptor)
}

func TestPlantObjectsSnapshotSeparateProfilesAndReplay(t *testing.T) {
	dir := t.TempDir()
	_, original, err := manufacture.EmbeddedPlantBook("cactus")
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(original, []byte("height-minimum: 4.0"), []byte("height-minimum: 7.0"), 1)
	changed = bytes.Replace(changed, []byte("height-maximum: 9.0"), []byte("height-maximum: 12.0"), 1)
	if bytes.Equal(changed, original) {
		t.Fatal("profile fixture did not change")
	}
	first := writeFile(t, dir, "first.yaml", string(original))
	second := writeFile(t, dir, "second.yaml", string(changed))
	source := plantObjectScene(plantObject("a", "plant: {family: cactus, file: first.yaml, species: saguaro}") + plantObject("b", "plant: {family: cactus, file: second.yaml, species: saguaro}"))
	path := writeFile(t, dir, "scene.yaml", source)
	plan, err := readPlan(path)
	if err != nil {
		t.Fatal(err)
	}
	a, b := plan.Site.Objects[0].Plant, plan.Site.Objects[1].Plant
	if a.binding.identity == b.binding.identity || a.binding.source == b.binding.source {
		t.Fatal("distinct books share identity")
	}
	if len(plan.modelSources) != 2 || len(plan.modelSourceRecords()) != 2 {
		t.Fatal("both exact sources were not captured")
	}
	if err := os.WriteFile(first, changed, 0600); err != nil {
		t.Fatal(err)
	}
	changedPlan, err := readPlan(path)
	if err != nil {
		t.Fatal(err)
	}
	if changedPlan.Site.Objects[0].Plant.binding.identity == a.binding.identity {
		t.Fatal("a new read did not notice changed book bytes")
	}
	if err := os.Remove(second); err != nil {
		t.Fatal(err)
	}
	site := roomySite(t)
	site.plan = plan.Site
	one, err := site.load(prototypeKey{kind: a.Species, plant: a.binding.identity, seed: 5, detail: "hull"})
	if err != nil {
		t.Fatal(err)
	}
	two, err := site.load(prototypeKey{kind: b.Species, plant: b.binding.identity, seed: 5, detail: "hull"})
	if err != nil {
		t.Fatal(err)
	}
	if one.bounds == two.bounds || len(site.store.values) != 2 {
		t.Fatal("different source geometry shared a prototype")
	}
	if err := site.object(plan.Site.Objects[0]); err != nil {
		t.Fatal(err)
	}
	if len(site.record.Objects) != 1 || site.record.Objects[0].Kind != "" || site.record.Objects[0].Plant == nil || site.record.Objects[0].Plant.SHA256 != a.binding.source {
		t.Fatal("placed object did not record its profile source")
	}
	bundle := filepath.Join(t.TempDir(), "bundle")
	if err := os.Mkdir(bundle, 0755); err != nil {
		t.Fatal(err)
	}
	replay, err := plan.replayScene(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "scene.yaml"), replay, 0644); err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(t.TempDir(), "moved")
	if err := os.Rename(bundle, moved); err != nil {
		t.Fatal(err)
	}
	reloaded, err := readPlan(filepath.Join(moved, "scene.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []*objectPlant{a, b} {
		got := reloaded.Site.Objects[i].Plant
		if got.binding.identity != want.binding.identity || got.binding.source != want.binding.source {
			t.Fatalf("object %d changed identity on replay", i)
		}
	}
	if !bytes.Equal(reloaded.modelSources[a.binding.source], original) || !bytes.Equal(reloaded.modelSources[b.binding.source], changed) {
		t.Fatal("replay did not keep exact source bytes")
	}
	if err := os.WriteFile(filepath.Join(moved, modelFilename(a.binding.source)), changed, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := readPlan(filepath.Join(moved, "scene.yaml")); err == nil || !strings.Contains(err.Error(), "plant book hash changed") {
		t.Fatalf("tampered replay book: %v", err)
	}
	track, _ := loadSequence(t, sequenceText("segments", "linear", "scene.yaml"))
	frame, err := track.sample(plan, 1)
	if err != nil {
		t.Fatal(err)
	}
	if frame.Site.Objects[0].Plant.binding.identity != a.binding.identity {
		t.Fatal("sequence frame lost frozen profile")
	}
	frameDir := filepath.Join(t.TempDir(), "frame")
	if err := os.Mkdir(frameDir, 0755); err != nil {
		t.Fatal(err)
	}
	frameRaw, err := frame.replayScene(frameDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(frameDir, "scene.yaml"), frameRaw, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := readPlan(filepath.Join(frameDir, "scene.yaml")); err != nil {
		t.Fatalf("sequence frame replay: %v", err)
	}
}

func TestPlantOptionChangesPrototypeIdentity(t *testing.T) {
	dir := t.TempDir()
	_, raw, err := manufacture.EmbeddedPlantBook("seasonal")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "plants.yaml", string(raw))
	scene := plantObjectScene(plantObject("a", "plant: {family: seasonal, file: plants.yaml, species: valley-oak, season: autumn}") + plantObject("b", "plant: {family: seasonal, file: plants.yaml, species: valley-oak, season: winter}"))
	plan, err := readPlan(writeFile(t, dir, "scene.yaml", scene))
	if err != nil {
		t.Fatal(err)
	}
	if plan.Site.Objects[0].Plant.binding.identity == plan.Site.Objects[1].Plant.binding.identity {
		t.Fatal("different seasons shared a prototype identity")
	}
}

func TestPlantObjectsRefuseInvalidDescriptorsAndKeepKinds(t *testing.T) {
	dir := t.TempDir()
	_, raw, err := manufacture.EmbeddedPlantBook("cactus")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "cactus.yaml", string(raw))
	writeFile(t, dir, "bad.yaml", "profiles: [")
	writeFile(t, dir, "oversized.yaml", strings.Repeat("x", maxPlantBookBytes+1))
	for _, tc := range []struct {
		name, descriptor string
		valid            bool
	}{
		{"valid plant", "plant: {family: cactus, file: cactus.yaml, species: saguaro}", true},
		{"valid hash", fmt.Sprintf("plant: {family: cactus, file: cactus.yaml, species: saguaro, sha256: '%s'}", configHash(raw)), true},
		{"valid kind", "kind: can", true},
		{"old plant kind", "kind: saguaro", false},
		{"mixed", "kind: can, plant: {family: cactus, file: cactus.yaml, species: saguaro}", false},
		{"missing file", "plant: {family: cactus, file: missing.yaml, species: saguaro}", false},
		{"wrong family", "plant: {family: fern, file: cactus.yaml, species: saguaro}", false},
		{"wrong hash", "plant: {family: cactus, file: cactus.yaml, species: saguaro, sha256: '0000000000000000000000000000000000000000000000000000000000000000'}", false},
		{"missing species", "plant: {family: cactus, file: cactus.yaml, species: absent}", false},
		{"invalid season", "plant: {family: cactus, file: cactus.yaml, species: saguaro, season: winter}", false},
		{"invalid shoot", "plant: {family: cactus, file: cactus.yaml, species: saguaro, representation: shoot}", false},
		{"malformed book", "plant: {family: cactus, file: bad.yaml, species: saguaro}", false},
		{"oversized book", "plant: {family: cactus, file: oversized.yaml, species: saguaro}", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scene := plantObjectScene(plantObject("a", tc.descriptor))
			path := writeFile(t, dir, "test.yaml", scene)
			_, err := readPlan(path)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%t: %v", tc.valid, err)
			}
		})
	}
}

func TestPlantSnapshotDeduplicatesContentAcrossPaths(t *testing.T) {
	_, raw, err := manufacture.EmbeddedPlantBook("cactus")
	if err != nil {
		t.Fatal(err)
	}
	sources, total := map[string][]byte{}, 0
	first, hash, err := keepPlantSource(sources, bytes.Clone(raw), &total)
	if err != nil {
		t.Fatal(err)
	}
	other, otherHash, err := keepPlantSource(sources, bytes.Clone(raw), &total)
	if err != nil {
		t.Fatal(err)
	}
	if hash != otherHash || len(sources) != 1 || total != len(raw) || &first[0] != &other[0] {
		t.Fatal("identical books retained different source allocations")
	}
	dir := t.TempDir()
	writeFile(t, dir, "first.yaml", string(raw))
	writeFile(t, dir, "second.yaml", string(raw))
	scene := plantObjectScene(plantObject("a", "plant: {family: cactus, file: first.yaml, species: saguaro}") + plantObject("b", "plant: {family: cactus, file: second.yaml, species: saguaro}"))
	plan, err := readPlan(writeFile(t, dir, "scene.yaml", scene))
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.modelSources) != 1 || plan.Site.Objects[0].Plant.binding.identity != plan.Site.Objects[1].Plant.binding.identity {
		t.Fatal("identical source paths did not share model identity")
	}
}

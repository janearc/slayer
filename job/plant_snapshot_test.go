package job_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
)

func TestPlantJobFreezesExternalProfileAndChangesRenderKeys(t *testing.T) {
	dir := t.TempDir()
	raw, err := os.ReadFile("../manufacture/plants/seasonal/profiles.yaml")
	if err != nil {
		t.Fatal(err)
	}
	raw = bytes.Replace(raw, []byte("kind: valley-oak"), []byte("kind: meadow-oak"), 1)
	scene := `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: meadow
  setting: space
  light: none
  film: {width: 8, height: 8, rays: 1}
  cameras:
    far: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
  recipes:
    oak: {plant: {file: plants.yaml, species: meadow-oak}}
  set:
    things:
      tree: {recipe: oak, seed: 1, complexity: hull, east: 0, north: 0, height: {value: 0, unit: m}}
  world: {unit: 1}
`
	jobText := `"$schema": https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: scene.yaml}
film: {width: 8, height: 8, rays: 1}
outputs: {picture: out.png}
`
	for name, text := range map[string][]byte{"plants.yaml": raw, "scene.yaml": []byte(scene), "job.yaml": []byte(jobText)} {
		if err := os.WriteFile(filepath.Join(dir, name), text, 0600); err != nil {
			t.Fatal(err)
		}
	}
	path, asset := filepath.Join(dir, "job.yaml"), filepath.Join(dir, "plants.yaml")
	first, err := job.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	before := first.Canonical()
	firstBounds := frozenPlantBounds(t, first)
	originalScale := []byte("  scale:\n    minimum: 9\n    maximum: 16\n")
	largerScale := []byte("  scale:\n    minimum: 18\n    maximum: 32\n")
	scaleAt := bytes.Index(raw, originalScale)
	speciesAt := bytes.Index(raw, []byte("kind: meadow-oak"))
	if scaleAt < 0 || speciesAt < 0 || scaleAt > speciesAt {
		t.Fatal("expected meadow oak scale before its kind")
	}
	changed := bytes.Replace(raw, originalScale, largerScale, 1)
	if err := os.WriteFile(asset, changed, 0600); err != nil {
		t.Fatal(err)
	}
	second, err := job.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	secondBounds := frozenPlantBounds(t, second)
	if secondBounds == firstBounds {
		t.Fatal("geometry-changing profile edit did not change bounds")
	}
	if got := frozenPlantBounds(t, first); got != firstBounds {
		t.Fatalf("first job changed after edit: got %v want %v", got, firstBounds)
	}
	if first.Key() == second.Key() || first.PictureKey() == second.PictureKey() || first.LightKey() == second.LightKey() {
		t.Fatal("profile edit reused render keys")
	}
	if err := os.Remove(asset); err != nil {
		t.Fatal(err)
	}
	if got := frozenPlantBounds(t, first); got != firstBounds {
		t.Fatalf("first job changed after deleting profile: got %v want %v", got, firstBounds)
	}
	if got := frozenPlantBounds(t, second); got != secondBounds {
		t.Fatalf("second job changed after deleting profile: got %v want %v", got, secondBounds)
	}
	if !bytes.Equal(before, first.Canonical()) {
		t.Fatal("old snapshot changed")
	}
	if _, err := job.Read(path); err == nil || !strings.Contains(err.Error(), "plants.yaml") {
		t.Fatalf("missing profile: %v", err)
	}
}

// frozenPlantBounds constructs the checked recipe retained by the job, not
// one re-read from the profile file. It witnesses the geometry of that snapshot.
func frozenPlantBounds(t *testing.T, wanted job.Job) maths.Box {
	t.Helper()
	scene, ok := wanted.ImportedSet()
	if !ok {
		t.Fatal("plant set was not frozen")
	}
	recipe, ok := scene.Recipe("oak")
	if !ok {
		t.Fatal("frozen set has no oak recipe")
	}
	row, err := recipe.Construction()
	if err != nil {
		t.Fatal(err)
	}
	things := scene.Things()
	if len(things) != 1 {
		t.Fatalf("frozen set has %d things", len(things))
	}
	made, err := manufacture.MakeWithComplexity(*row, manufacture.Sow(1), things[0].Complexity())
	if err != nil {
		t.Fatal(err)
	}
	return made.Bounds()
}

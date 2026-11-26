package job_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
)

func TestAlgaJobFreezesProfileAndChangesRenderKeys(t *testing.T) {
	dir := t.TempDir()
	_, raw, err := manufacture.EmbeddedAlgaBook("kelp")
	if err != nil {
		t.Fatal(err)
	}
	scene := `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: kelp
  setting: space
  light: none
  film: {width: 8, height: 8, rays: 1}
  cameras:
    far: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
  recipes:
    specimen: {alga: {family: kelp, file: kelp.yaml, species: macrocystis-pyrifera, condition: moderate}}
  set:
    things:
      one: {recipe: specimen, seed: 1, complexity: quick, east: 0, north: 0, height: {value: 0, unit: m}}
  world: {unit: 1}
`
	jobText := `"$schema": https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: scene.yaml}
film: {width: 8, height: 8, rays: 1}
outputs: {picture: out.png}
`
	for name, data := range map[string][]byte{"kelp.yaml": raw, "scene.yaml": []byte(scene), "job.yaml": []byte(jobText)} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	path, asset := filepath.Join(dir, "job.yaml"), filepath.Join(dir, "kelp.yaml")
	first, err := job.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	before, firstKeys := frozenAlgaBounds(t, first), animalJobKeys(first)
	changed := bytes.Replace(raw, []byte("depth: {least: 12, most: 12"), []byte("depth: {least: 8, most: 8"), 1)
	if bytes.Equal(raw, changed) {
		t.Fatal("expected depth not present")
	}
	if err := os.WriteFile(asset, changed, 0600); err != nil {
		t.Fatal(err)
	}
	second, err := job.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	after, secondKeys := frozenAlgaBounds(t, second), animalJobKeys(second)
	if before == after {
		t.Fatal("profile depth did not change geometry")
	}
	for i := range firstKeys {
		if firstKeys[i] == secondKeys[i] {
			t.Fatalf("profile edit did not change key %d", i)
		}
	}
	if err := os.Remove(asset); err != nil {
		t.Fatal(err)
	}
	if frozenAlgaBounds(t, first) != before || frozenAlgaBounds(t, second) != after {
		t.Fatal("frozen profiles changed after source deletion")
	}
	if _, err := job.Read(path); err == nil {
		t.Fatal("accepted missing alga book")
	}
}

func frozenAlgaBounds(t *testing.T, wanted job.Job) maths.Box {
	t.Helper()
	scene, ok := wanted.ImportedSet()
	if !ok {
		t.Fatal("alga set was not frozen")
	}
	recipe, ok := scene.Recipe("specimen")
	if !ok || !recipe.IsAlga() {
		t.Fatal("alga recipe missing")
	}
	row, err := recipe.Construction()
	if err != nil {
		t.Fatal(err)
	}
	made, err := manufacture.Make(*row, manufacture.Sow(1))
	if err != nil {
		t.Fatal(err)
	}
	return made.Bounds()
}

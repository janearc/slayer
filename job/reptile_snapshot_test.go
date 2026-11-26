package job_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/manufacture"
)

func TestReptileJobSnapshotsAnatomyAndKeys(t *testing.T) {
	dir := t.TempDir()
	_, raw := manufacture.EmbeddedReptileBook()
	scene := `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: purussaurus
  setting: space
  light: none
  film: {width: 8, height: 8, rays: 1}
  cameras:
    far: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
  recipes:
    specimen: {animal: {family: reptile, file: reptiles.yaml, species: purussaurus}}
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
	for name, data := range map[string][]byte{"reptiles.yaml": raw, "scene.yaml": []byte(scene), "job.yaml": []byte(jobText)} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	path, asset := filepath.Join(dir, "job.yaml"), filepath.Join(dir, "reptiles.yaml")
	first, err := job.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	before, firstKeys := frozenAnimalGeometry(t, first), animalJobKeys(first)
	changed := bytes.Replace(raw, []byte("0.215, 0.24, 0.18, 0.34"), []byte("0.215, 0.28, 0.18, 0.34"), 1)
	if bytes.Equal(raw, changed) {
		t.Fatal("missing anatomy fixture")
	}
	if err := os.WriteFile(asset, changed, 0600); err != nil {
		t.Fatal(err)
	}
	second, err := job.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	after, secondKeys := frozenAnimalGeometry(t, second), animalJobKeys(second)
	if before == after {
		t.Fatal("anatomy edit did not change geometry")
	}
	for i := range firstKeys {
		if firstKeys[i] == secondKeys[i] {
			t.Fatalf("anatomy edit did not change key %d", i)
		}
	}
	if err := os.Remove(asset); err != nil {
		t.Fatal(err)
	}
	if frozenAnimalGeometry(t, first) != before || frozenAnimalGeometry(t, second) != after {
		t.Fatal("frozen anatomy changed")
	}
	if _, err := job.Read(path); err == nil {
		t.Fatal("missing reptile profile was accepted")
	}
}

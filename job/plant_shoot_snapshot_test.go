package job_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/job"
)

func TestConiferShootJobFreezesProfileAndChangesKeys(t *testing.T) {
	dir := t.TempDir()
	raw, err := os.ReadFile("../manufacture/plants/conifers/conifer-profiles.yaml")
	if err != nil {
		t.Fatal(err)
	}
	oldWidth, newWidth := []byte("width: 0.002\n"), []byte("width: 0.01\n")
	if bytes.Count(raw, oldWidth) != 1 {
		t.Fatal("expected one redwood needle width")
	}
	scene := `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: shoot
  setting: space
  light: none
  film: {width: 8, height: 8, rays: 1}
  cameras:
    far: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
  recipes:
    oak: {plant: {family: conifer, file: conifers.yaml, species: coast-redwood, representation: shoot}}
  set:
    things:
      shoot: {recipe: oak, seed: 1, complexity: hull, east: 0, north: 0, height: {value: 0, unit: m}}
  world: {unit: 1}
`
	jobText := `"$schema": https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: scene.yaml}
film: {width: 8, height: 8, rays: 1}
outputs: {picture: out.png}
`
	for name, data := range map[string][]byte{"conifers.yaml": raw, "scene.yaml": []byte(scene), "job.yaml": []byte(jobText)} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	path, source := filepath.Join(dir, "job.yaml"), filepath.Join(dir, "conifers.yaml")
	before, err := job.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	oldBounds := frozenPlantBounds(t, before)
	wholeScene := bytes.Replace([]byte(scene), []byte(", representation: shoot"), nil, 1)
	if bytes.Equal(wholeScene, []byte(scene)) {
		t.Fatal("shoot representation missing from fixture")
	}
	if err := os.WriteFile(filepath.Join(dir, "scene.yaml"), wholeScene, 0600); err != nil {
		t.Fatal(err)
	}
	whole, err := job.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if frozenPlantBounds(t, whole) == oldBounds {
		t.Fatal("whole tree and shoot have identical bounds")
	}
	if before.Key() == whole.Key() || before.PictureKey() == whole.PictureKey() || before.LightKey() == whole.LightKey() {
		t.Fatal("representation-only change reused job keys")
	}
	if err := os.WriteFile(filepath.Join(dir, "scene.yaml"), []byte(scene), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, bytes.Replace(raw, oldWidth, newWidth, 1), 0600); err != nil {
		t.Fatal(err)
	}
	after, err := job.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	newBounds := frozenPlantBounds(t, after)
	if oldBounds == newBounds {
		t.Fatal("edited shoot profile changed no geometry")
	}
	if before.Key() == after.Key() || before.PictureKey() == after.PictureKey() || before.LightKey() == after.LightKey() {
		t.Fatal("profile edit reused job keys")
	}
	if err := os.Remove(source); err != nil {
		t.Fatal(err)
	}
	if frozenPlantBounds(t, before) != oldBounds || frozenPlantBounds(t, after) != newBounds {
		t.Fatal("frozen shoot changed after deletion")
	}
	if _, err := job.Read(path); err == nil {
		t.Fatal("accepted missing book")
	}
}

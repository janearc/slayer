package job_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/manufacture"
)

func TestPlantFamilyJobSnapshotsGeometryAndKeys(t *testing.T) {
	for _, tc := range []struct{ family, species, detail, before, after string }{
		{"conifer", "spruce", "hull", "height: [4, 11]", "height: [4.1, 11.1]"},
		{"fern", "fern", "quick", "frond-length: {minimum: 0.4, maximum: 0.9}", "frond-length: {minimum: 0.5, maximum: 1.0}"},
		{"fern", "fern", "quick", "leaflet-reach: 0.22", "leaflet-reach: 0.3"},
		{"cactus", "saguaro", "hull", "height-minimum: 4.0", "height-minimum: 5.0"},
	} {
		t.Run(tc.family, func(t *testing.T) {
			dir := t.TempDir()
			_, raw, err := manufacture.EmbeddedPlantBook(tc.family)
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Count(raw, []byte(tc.before)) != 1 {
				t.Fatal("fixture dimension not unique")
			}
			scene := fmt.Sprintf(`$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: plant
  setting: space
  light: none
  film: {width: 8, height: 8, rays: 1}
  cameras:
    far: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
  recipes:
    oak: {plant: {family: %s, file: plant.yaml, species: %s}}
  set:
    things:
      tree: {recipe: oak, seed: 1, complexity: %s, east: 0, north: 0, height: {value: 0, unit: m}}
  world: {unit: 1}
`, tc.family, tc.species, tc.detail)
			jobText := `"$schema": https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: scene.yaml}
film: {width: 8, height: 8, rays: 1}
outputs: {picture: out.png}
`
			for name, content := range map[string][]byte{"plant.yaml": raw, "scene.yaml": []byte(scene), "job.yaml": []byte(jobText)} {
				if err := os.WriteFile(filepath.Join(dir, name), content, 0600); err != nil {
					t.Fatal(err)
				}
			}
			path, asset := filepath.Join(dir, "job.yaml"), filepath.Join(dir, "plant.yaml")
			old, err := job.Read(path)
			if err != nil {
				t.Fatal(err)
			}
			oldBounds, oldKeys := frozenPlantBounds(t, old), animalJobKeys(old)
			changed := bytes.Replace(raw, []byte(tc.before), []byte(tc.after), 1)
			if err := os.WriteFile(asset, changed, 0600); err != nil {
				t.Fatal(err)
			}
			newer, err := job.Read(path)
			if err != nil {
				t.Fatal(err)
			}
			newBounds, newKeys := frozenPlantBounds(t, newer), animalJobKeys(newer)
			if oldBounds == newBounds {
				t.Fatal("profile geometry edit changed no bounds")
			}
			if oldKeys[0] == newKeys[0] || oldKeys[1] == newKeys[1] || oldKeys[2] == newKeys[2] {
				t.Fatal("profile change did not invalidate all render keys")
			}
			if err := os.Remove(asset); err != nil {
				t.Fatal(err)
			}
			if frozenPlantBounds(t, old) != oldBounds || frozenPlantBounds(t, newer) != newBounds ||
				animalJobKeys(old) != oldKeys || animalJobKeys(newer) != newKeys {
				t.Fatal("frozen geometry or keys changed after profile deletion")
			}
			if _, err := job.Read(path); err == nil {
				t.Fatal("accepted missing profile")
			}
		})
	}
}

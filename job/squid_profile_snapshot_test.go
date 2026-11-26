package job_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/manufacture"
)

func TestSquidJobFreezesModelAndChangesRenderKeys(t *testing.T) {
	for _, tc := range []struct{ name, before, after string }{
		{"length", "mantle-metres: 0.8", "mantle-metres: 1.6"},
		{"anatomy", "mantle-radius: 0.255", "mantle-radius: 0.30"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, raw, err := manufacture.EmbeddedAnimalBook("squid")
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Count(raw, []byte(tc.before)) != 1 {
				t.Fatal("squid fixture absent or ambiguous")
			}
			dir := t.TempDir()
			scene := `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: squid
  setting: space
  light: none
  film: {width: 8, height: 8, rays: 1}
  cameras: {view: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}}
  recipes:
    specimen: {animal: {family: squid, file: squid.yaml, species: humboldt-squid, phase: 1.1}}
  set:
    things:
      squid: {recipe: specimen, seed: 3, complexity: quick, east: 0, north: 0, height: {value: 0, unit: m}}
  world: {unit: 1}
`
			jobText := `"$schema": https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: scene.yaml}
film: {width: 8, height: 8, rays: 1}
outputs: {picture: out.png}
`
			for name, data := range map[string][]byte{"scene.yaml": []byte(scene), "job.yaml": []byte(jobText), "squid.yaml": raw} {
				if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			path, asset := filepath.Join(dir, "job.yaml"), filepath.Join(dir, "squid.yaml")
			first, err := job.Read(path)
			if err != nil {
				t.Fatal(err)
			}
			before, firstKeys := frozenAnimalGeometry(t, first), animalJobKeys(first)
			changed := bytes.Replace(raw, []byte(tc.before), []byte(tc.after), 1)
			if err := os.WriteFile(asset, changed, 0600); err != nil {
				t.Fatal(err)
			}
			second, err := job.Read(path)
			if err != nil {
				t.Fatal(err)
			}
			after, secondKeys := frozenAnimalGeometry(t, second), animalJobKeys(second)
			if before == after {
				t.Fatal("profile change did not reach geometry")
			}
			for i := range firstKeys {
				if firstKeys[i] == secondKeys[i] {
					t.Fatalf("profile change did not change key %d", i)
				}
			}
			if err := os.Remove(asset); err != nil {
				t.Fatal(err)
			}
			if frozenAnimalGeometry(t, first) != before || frozenAnimalGeometry(t, second) != after ||
				animalJobKeys(first) != firstKeys || animalJobKeys(second) != secondKeys {
				t.Fatal("frozen geometry or keys changed after source deletion")
			}
			if _, err := job.Read(path); err == nil {
				t.Fatal("fresh job accepted a missing squid book")
			}
		})
	}
}

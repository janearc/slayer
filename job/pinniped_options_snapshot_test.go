package job_test

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
)

func TestPinnipedOptionsFreezeGeometryAndStudyAppearance(t *testing.T) {
	for _, tc := range []struct {
		name, species, option, old, replacement string
		study, walrus                           bool
	}{
		{"harbor bones", "harbor-seal", "representation: bones", "cervical: 7", "cervical: 8", false, false},
		{"walrus bones", "walrus", "representation: bones", "thoracic: 14", "thoracic: 15", false, true},
		{"walrus volumes", "walrus", "representation: volumes", "thoracic: 14", "thoracic: 15", false, true},
		{"detailed study", "harbor-seal", "study: detailed-wet", "wet: 0.85", "wet: 0.4", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			raw, err := os.ReadFile(filepath.Join("..", "manufacture", "animals", "vertebrates", "mammals", "pinnipeds", "profiles.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(raw, []byte(tc.old)) {
				t.Fatal("fixture absent")
			}
			changed := bytes.Replace(raw, []byte(tc.old), []byte(tc.replacement), 1)
			if tc.walrus {
				from := []byte(`thoracic: {basis: read, source: "Pierce, Clack and Hutchinson 2011, Journal of Anatomy 219:502, PMC3196755", value: 14}`)
				if !bytes.Contains(changed, from) {
					t.Fatal("evidence fixture absent")
				}
				changed = bytes.Replace(changed, from, []byte("thoracic: {basis: picked}"), 1)
			}
			scene := fmt.Sprintf(`$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: pinniped
  setting: space
  light: none
  seed: 19
  film: {width: 8, height: 8, rays: 1}
  cameras:
    far: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
  recipes:
    specimen: {animal: {family: pinniped, file: pinnipeds.yaml, species: %s, %s}}
  set:
    things:
      one: {recipe: specimen, seed: 1, complexity: quick, east: 0, north: 0, height: {value: 0, unit: m}}
  world: {unit: 1}
`, tc.species, tc.option)
			jobText := `"$schema": https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: scene.yaml}
film: {width: 8, height: 8, rays: 1}
outputs: {picture: out.png}
`
			for name, data := range map[string][]byte{"pinnipeds.yaml": raw, "scene.yaml": []byte(scene), "job.yaml": []byte(jobText)} {
				if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			path, asset := filepath.Join(dir, "job.yaml"), filepath.Join(dir, "pinnipeds.yaml")
			first, err := job.Read(path)
			if err != nil {
				t.Fatal(err)
			}
			before, firstKeys := pinnipedOptionWitness(t, first, tc.study), animalJobKeys(first)
			if err := os.WriteFile(asset, changed, 0600); err != nil {
				t.Fatal(err)
			}
			second, err := job.Read(path)
			if err != nil {
				t.Fatal(err)
			}
			after, secondKeys := pinnipedOptionWitness(t, second, tc.study), animalJobKeys(second)
			if before == after {
				t.Fatal("profile edit did not affect the selected representation")
			}
			for i := range firstKeys {
				if firstKeys[i] == secondKeys[i] {
					t.Fatalf("render key %d did not change", i)
				}
			}
			if err := os.Remove(asset); err != nil {
				t.Fatal(err)
			}
			if pinnipedOptionWitness(t, first, tc.study) != before || pinnipedOptionWitness(t, second, tc.study) != after {
				t.Fatal("frozen representation changed after source deletion")
			}
			if animalJobKeys(first) != firstKeys || animalJobKeys(second) != secondKeys {
				t.Fatal("frozen render keys changed")
			}
			if _, err := job.Read(path); err == nil {
				t.Fatal("accepted missing profile")
			}
		})
	}
}

func pinnipedOptionWitness(t *testing.T, wanted job.Job, study bool) string {
	t.Helper()
	if !study {
		return fmt.Sprintf("%x", frozenAnimalGeometry(t, wanted))
	}
	scene, ok := wanted.ImportedSet()
	if !ok {
		t.Fatal("set was not frozen")
	}
	recipe, ok := scene.Recipe("specimen")
	if !ok {
		t.Fatal("animal recipe missing")
	}
	row, err := recipe.Construction()
	if err != nil {
		t.Fatal(err)
	}
	made, err := manufacture.MakeWithComplexity(*row, manufacture.Sow(1), "quick")
	if err != nil {
		t.Fatal(err)
	}
	parts, err := made.Things(maths.Local())
	if err != nil || len(parts) == 0 {
		t.Fatalf("study parts: %v", err)
	}
	return fmt.Sprintf("%x", math.Float64bits(parts[0].Material.Gloss))
}

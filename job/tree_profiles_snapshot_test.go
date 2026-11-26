package job_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
)

func TestTreeProfileJobSnapshotsAndChangesAllKeys(t *testing.T) {
	raw, err := os.ReadFile("../manufacture/plants/seasonal/profiles.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"tower", "city-clustered-green-10000"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			pins := ""
			if kind == "tower" {
				pins = ", pins: {skin: concrete, planted: true, storeys: 30}"
			}
			scene := `"$schema": https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: tree models
  setting: space
  light: none
  film: {width: 8, height: 8, rays: 1}
  cameras: {view: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}}
  recipes:
    tree scene: {kind: ` + kind + `, tree-profiles: {file: trees.yaml}` + pins + `}
  set:
    things:
      object: {recipe: tree scene, seed: 3, complexity: quick, east: 0, north: 0, height: {value: 0, unit: m}}
  world: {unit: 1}
`
			jobText := `"$schema": https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: scene.yaml}
film: {width: 8, height: 8, rays: 1}
outputs: {picture: out.png}
`
			for name, data := range map[string][]byte{"scene.yaml": []byte(scene), "job.yaml": []byte(jobText), "trees.yaml": raw} {
				if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			path, asset := filepath.Join(dir, "job.yaml"), filepath.Join(dir, "trees.yaml")
			first, err := job.Read(path)
			if err != nil {
				t.Fatal(err)
			}
			frozen, ok := first.ImportedSet()
			if !ok {
				t.Fatal("job did not freeze the set")
			}
			recipe, ok := frozen.Recipe("tree scene")
			if !ok || !recipe.HasTreeProfiles() {
				t.Fatal("frozen book lost")
			}
			var oldColours map[maths.Light]bool
			if kind == "tower" {
				oldColours = frozenTreeColours(t, first)
			}
			// Change the green component for every seasonal species so any
			// tower choice is affected, without changing its seeded structure.
			green := regexp.MustCompile(`(?m)(  green:\n    r: )[^\n]+`)
			changed := green.ReplaceAll(raw, []byte("${1}0.91"))
			if bytes.Equal(raw, changed) {
				t.Fatal("missing foliage colour fixture")
			}
			if err := os.WriteFile(asset, changed, 0600); err != nil {
				t.Fatal(err)
			}
			second, err := job.Read(path)
			if err != nil {
				t.Fatal(err)
			}
			var newColours map[maths.Light]bool
			if kind == "tower" {
				newColours = frozenTreeColours(t, second)
				if reflect.DeepEqual(oldColours, newColours) {
					t.Fatal("real profile edit did not change tower foliage")
				}
			}
			if first.Key() == second.Key() || first.PictureKey() == second.PictureKey() || first.LightKey() == second.LightKey() {
				t.Fatal("changed tree book retained a render key")
			}
			if err := os.Remove(asset); err != nil {
				t.Fatal(err)
			}
			if _, err := recipe.Construction(); err != nil {
				t.Fatalf("frozen recipe reopened deleted file: %v", err)
			}
			if kind == "tower" {
				if got := frozenTreeColours(t, first); !reflect.DeepEqual(got, oldColours) {
					t.Fatal("old job foliage changed after deleting its book")
				}
				if got := frozenTreeColours(t, second); !reflect.DeepEqual(got, newColours) {
					t.Fatal("new job foliage changed after deleting its book")
				}
			}
			if _, err := job.Read(path); err == nil {
				t.Fatal("fresh job accepted missing book")
			}
		})
	}
}

// frozenTreeColours builds the actual checked tower retained by the job.
func frozenTreeColours(t *testing.T, wanted job.Job) map[maths.Light]bool {
	t.Helper()
	scene, ok := wanted.ImportedSet()
	if !ok {
		t.Fatal("tree set was not frozen")
	}
	recipe, ok := scene.Recipe("tree scene")
	if !ok {
		t.Fatal("tree recipe missing")
	}
	row, err := recipe.Construction()
	if err != nil {
		t.Fatal(err)
	}
	made, err := manufacture.MakeWithComplexity(*row, manufacture.Sow(3), "quick")
	if err != nil {
		t.Fatal(err)
	}
	parts, err := made.Things(maths.Local())
	if err != nil {
		t.Fatal(err)
	}
	colours := map[maths.Light]bool{}
	for _, part := range parts {
		if part.Material != nil && part.Material.Name == "broad foliage" {
			colours[part.Material.Colour] = true
		}
	}
	if len(colours) == 0 {
		t.Fatal("pinned tower produced no visible tree foliage")
	}
	return colours
}

package job_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/job"
)

func TestCompositionJobFreezesExternalBytesAndChangesAllRenderKeys(t *testing.T) {
	dir := t.TempDir()
	composition := `version: 1
name: post
parts:
  - {id: post, type: leg, at: [0, 0, 0], size: [2, 3, 2], ground: true}
`
	set := `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: post
  setting: space
  light: none
  film: {width: 64, height: 48, rays: 1, wide: 40, exposure: 1}
  cameras:
    far:
      location: {at: [0, -300, 2]}
      orientation: {look_at: [0, 0, 60]}
  recipes:
    post: {composition: {file: post.yaml}}
  set:
    things:
      post: {recipe: post, seed: 1, complexity: hull, east: 0, north: 0, height: {value: 0, unit: m}}
  world: {unit: 1}
`
	jobText := `"$schema": https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: set.yaml}
film: {width: 32, height: 24, rays: 1}
outputs: {picture: picture.png}
`
	for name, text := range map[string]string{"post.yaml": composition, "set.yaml": set, "job.yaml": jobText} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	path, asset := filepath.Join(dir, "job.yaml"), filepath.Join(dir, "post.yaml")
	first, err := job.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	before := first.Canonical()
	snapshot, ok := first.ImportedSet()
	if !ok {
		t.Fatal("composition set was not frozen")
	}
	sources, err := snapshot.CompositionSources()
	if err != nil || len(sources) != 1 {
		t.Fatalf("sources=%v error=%v", sources, err)
	}
	if err := os.WriteFile(asset, []byte(strings.Replace(composition, "[2, 3, 2]", "[2, 4, 2]", 1)), 0600); err != nil {
		t.Fatal(err)
	}
	second, err := job.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if first.Key() == second.Key() || first.PictureKey() == second.PictureKey() || first.LightKey() == second.LightKey() {
		t.Fatal("changed external file retained a render key")
	}
	if err := os.Remove(asset); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, first.Canonical()) {
		t.Fatal("old snapshot changed")
	}
	if _, err := job.Read(path); err == nil {
		t.Fatal("missing composition accepted")
	}
}

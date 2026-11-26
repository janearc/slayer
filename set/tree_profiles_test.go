package set_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

func treeProfileScene(schema, kind, extra string) string {
	source := `"$schema": https://github.com/janearc/slayer/schemas/` + schema + `
scene:
  name: trees
  setting: space
  light: none
  film: {width: 8, height: 8, rays: 1}
  cameras:
    view: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}
  recipes:
    trees:
      kind: ` + kind + `
      tree-profiles: {file: trees.yaml}
` + extra + `  set:
    things:
      object: {recipe: trees, seed: 3, complexity: quick, east: 0, north: 0, height: {value: 0, unit: m}}
  world: {unit: 1}
`
	if schema == "ecosystem-v1.json" {
		source = strings.Replace(source, "  setting: space\n", "", 1)
	}
	return source
}

func TestTreeProfileRecipesFreezeSourcesAndDigest(t *testing.T) {
	raw, err := os.ReadFile("../manufacture/plants/seasonal/profiles.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"tower", "city-clustered-green-10000", "city-clustered-night-10000"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			asset, path := filepath.Join(dir, "trees.yaml"), filepath.Join(dir, "scene.yaml")
			if err := os.WriteFile(asset, raw, 0600); err != nil {
				t.Fatal(err)
			}
			for _, schema := range []string{"ecosystem-v1.json", "ecosystem-v2.json"} {
				scene := treeProfileScene(schema, kind, "")
				read, has, err := set.ReadImported(path, []byte(scene))
				if err != nil || !has {
					t.Fatalf("%s: frozen=%v %v", schema, has, err)
				}
				r, ok := read.Recipe("trees")
				if !ok || !r.HasTreeProfiles() {
					t.Fatal("book was not attached")
				}
				sources, err := read.ProfileSources()
				if err != nil || len(sources) != 1 || sources[asset] == ([32]byte{}) {
					t.Fatalf("sources %v: %v", sources, err)
				}
				changed := append(bytes.Clone(raw), '\n')
				if err := os.WriteFile(asset, changed, 0600); err != nil {
					t.Fatal(err)
				}
				again, err := set.Read(path, []byte(scene))
				if err != nil {
					t.Fatal(err)
				}
				second, _ := again.Recipe("trees")
				if r.Digest() == second.Digest() {
					t.Fatal("changed source retained recipe digest")
				}
				if err := os.Remove(asset); err != nil {
					t.Fatal(err)
				}
				if _, err := r.Construction(); err != nil {
					t.Fatalf("frozen book was reopened: %v", err)
				}
				if _, has, err := set.ReadImported(path, []byte(scene)); !has || err == nil {
					t.Fatalf("missing book: has=%v err=%v", has, err)
				}
				if err := os.WriteFile(asset, raw, 0600); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestTreeProfileRecipesRefuseInvalidBooksAndContexts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scene.yaml")
	raw, err := os.ReadFile("../manufacture/plants/seasonal/profiles.yaml")
	if err != nil {
		t.Fatal(err)
	}
	incomplete := bytes.Replace(raw, []byte("kind: valley-oak"), []byte("kind: missing-valley-oak"), 1)
	if bytes.Equal(raw, incomplete) {
		t.Fatal("missing valley oak fixture")
	}
	if err := os.WriteFile(filepath.Join(dir, "trees.yaml"), incomplete, 0600); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"tower", "city-clustered-green-10000"} {
		if _, err := set.Read(path, []byte(treeProfileScene("ecosystem-v2.json", kind, ""))); err == nil {
			t.Fatalf("%s accepted an incomplete book", kind)
		}
	}
	for _, block := range []string{"tree-profiles: null", "tree-profiles: {}", "tree-profiles: {file: null}", "tree-profiles: {file: ''}"} {
		source := strings.Replace(treeProfileScene("ecosystem-v2.json", "tower", ""), "tree-profiles: {file: trees.yaml}", block, 1)
		if _, has, err := set.ReadImported(path, []byte(source)); !has || err == nil {
			t.Fatalf("%s: has=%v err=%v", block, has, err)
		}
	}
	source := treeProfileScene("ecosystem-v2.json", "boulder", "")
	if _, err := set.Read(path, []byte(source)); err == nil {
		t.Fatal("boulder accepted tree profiles")
	}
}

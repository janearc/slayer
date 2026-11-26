package set_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/set"
)

func squidSet(schema, options string) string {
	header := ""
	if schema != "" {
		header = `"$schema": https://github.com/janearc/slayer/schemas/` + schema + "\n"
	}
	return header + `scene:
  name: squid
  light: none
  film: {width: 8, height: 8, rays: 1}
  cameras: {view: {location: {at: [0, -20, 2]}, orientation: {look_at: [0, 0, 2]}}}
  recipes:
    squid: {animal: {family: squid, file: squid.yaml, species: humboldt-squid` + options + `}}
  set:
    things:
      squid: {recipe: squid, seed: 3, complexity: quick, east: 0, north: 0, height: {value: 0, unit: m}}
  world: {unit: 1}
`
}

func TestSquidSetReadsPhaseAndFreezesProfile(t *testing.T) {
	_, raw, err := manufacture.EmbeddedAnimalBook("squid")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path, asset := filepath.Join(dir, "scene.yaml"), filepath.Join(dir, "squid.yaml")
	for _, schema := range []string{"", "ecosystem-v1.json", "ecosystem-v2.json"} {
		t.Run(schema, func(t *testing.T) {
			if err := os.WriteFile(asset, raw, 0600); err != nil {
				t.Fatal(err)
			}
			source := squidSet(schema, ", phase: 1.1")
			read, has, err := set.ReadImported(path, []byte(source))
			if err != nil || !has {
				t.Fatalf("set snapshot: has=%v, %v", has, err)
			}
			recipe, found := read.Recipe("squid")
			if !found || !recipe.IsAnimal() || recipe.AnimalSpecies() != "humboldt-squid" {
				t.Fatal("squid identity lost")
			}
			sources, err := read.ProfileSources()
			if err != nil || len(sources) != 1 || sources[asset] == ([32]byte{}) {
				t.Fatalf("sources %v: %v", sources, err)
			}
			row, err := recipe.Construction()
			if err != nil {
				t.Fatal(err)
			}
			old, err := manufacture.Make(*row, manufacture.Sow(3))
			if err != nil {
				t.Fatal(err)
			}
			changed := bytes.Replace(raw, []byte("mantle-metres: 0.8"), []byte("mantle-metres: 1.6"), 1)
			if bytes.Equal(raw, changed) {
				t.Fatal("missing mantle fixture")
			}
			if err := os.WriteFile(asset, changed, 0600); err != nil {
				t.Fatal(err)
			}
			second, err := set.Read(path, []byte(source))
			if err != nil {
				t.Fatal(err)
			}
			other, _ := second.Recipe("squid")
			if recipe.Digest() == other.Digest() {
				t.Fatal("changed squid book retained recipe digest")
			}
			newRow, err := other.Construction()
			if err != nil {
				t.Fatal(err)
			}
			newMade, err := manufacture.Make(*newRow, manufacture.Sow(3))
			if err != nil || newMade.Bounds() == old.Bounds() {
				t.Fatalf("changed size lost: %v", err)
			}
			if err := os.Remove(asset); err != nil {
				t.Fatal(err)
			}
			frozen, err := manufacture.Make(*row, manufacture.Sow(3))
			if err != nil || frozen.Bounds() != old.Bounds() {
				t.Fatalf("frozen specimen changed after deletion: %v", err)
			}
			if _, has, err := set.ReadImported(path, []byte(source)); !has || err == nil {
				t.Fatalf("missing book has=%v err=%v", has, err)
			}
		})
	}
}

func TestSquidSetRejectsWrongPhaseAndOptions(t *testing.T) {
	_, raw, _ := manufacture.EmbeddedAnimalBook("squid")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "squid.yaml"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "scene.yaml")
	for _, schema := range []string{"", "ecosystem-v1.json", "ecosystem-v2.json"} {
		for _, option := range []string{", phase: null", ", phase: '0'", ", phase: .nan", ", phase: 100001", ", seconds: 0", ", count: 0", ", pose: swimming", ", representation: skin", ", study: bay"} {
			if _, err := set.Read(path, []byte(squidSet(schema, option))); err == nil {
				t.Errorf("%s accepted %s", schema, option)
			}
		}
		if _, err := set.Read(path, []byte(squidSet(schema, ", phase: 0"))); err != nil {
			t.Errorf("%s refused zero phase: %v", schema, err)
		}
		unknown := strings.Replace(squidSet(schema, ""), "humboldt-squid", "unknown-squid", 1)
		if _, err := set.Read(path, []byte(unknown)); err == nil {
			t.Errorf("%s accepted unknown species", schema)
		}
	}
}

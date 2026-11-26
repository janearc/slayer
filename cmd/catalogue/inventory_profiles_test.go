package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/manufacture"
)

func TestInventoryReportsCheckedProfileIdentity(t *testing.T) {
	menu, err := readMenu()
	if err != nil {
		t.Fatal(err)
	}
	entries, err := menu.selectEntries([]string{"fern", "pacific-sardine-glide", "human-scale-2m"})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := writeInventory(&output, entries); err != nil {
		t.Fatal(err)
	}
	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(output.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != len(entries) {
		t.Fatalf("got %d rows for %d entries", len(rows), len(entries))
	}
	for i, row := range rows {
		entry := entries[i]
		if entry.Animal == nil && entry.Plant == nil {
			if row["animal"] != nil || row["plant"] != nil || row["profile-identity-sha256"] != nil {
				t.Fatalf("ordinary row acquired profile metadata: %s", output.String())
			}
			continue
		}
		var source string
		if err := json.Unmarshal(row["profile-identity-sha256"], &source); err != nil || source == "" || source != entry.source {
			t.Fatalf("%s has wrong source identity: %q (%v)", entry.Kind, source, err)
		}
		if entry.Plant != nil {
			var spec plantSpec
			if err := json.Unmarshal(row["plant"], &spec); err != nil || spec != *entry.Plant || row["animal"] != nil {
				t.Fatalf("%s lost its plant descriptor: %s (%v)", entry.Kind, row["plant"], err)
			}
		} else {
			var spec animalSpec
			if err := json.Unmarshal(row["animal"], &spec); err != nil || spec.Family != "fish" || spec.Species != "pacific-sardine" || spec.Pose != "glide" || row["plant"] != nil {
				t.Fatalf("%s lost its animal descriptor: %s (%v)", entry.Kind, row["animal"], err)
			}
		}
	}
}

func TestInventoryProfileIdentityUsesFrozenExactBookBytes(t *testing.T) {
	name, raw, err := manufacture.EmbeddedPlantBook("fern")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	spec := plantSpec{Family: "fern", File: name, Species: "fern"}
	first := entry{Kind: "fern", Plant: &spec}
	if err := first.bindPlant(&modelBooks{root: dir}); err != nil {
		t.Fatal(err)
	}
	// A comment changes source bytes without changing the checked geometry.
	if err := os.WriteFile(path, append(bytes.Clone(raw), []byte("\n# second authored copy\n")...), 0600); err != nil {
		t.Fatal(err)
	}
	second := entry{Kind: "fern", Plant: &spec}
	if err := second.bindPlant(&modelBooks{root: dir}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	a, err := measureEntry(first)
	if err != nil {
		t.Fatal(err)
	}
	b, err := measureEntry(second)
	if err != nil {
		t.Fatal(err)
	}
	if a.Source == "" || a.Source == b.Source || a.Parts != b.Parts || a.Bounds != b.Bounds {
		t.Fatalf("source identity or frozen geometry was lost: %+v %+v", a, b)
	}
}

package main

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func TestCatalogueSauropodProfilesCanBeSelectedMeasuredAndConstructed(t *testing.T) {
	menu, err := readMenu()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"alamosaurus-sanjuanensis", "diplodocus-carnegii",
		"apatosaurus-louisae", "brachiosaurus-altithorax",
	}
	// Selection follows the catalogue order, not the order of the request.
	requested := slices.Clone(want)
	slices.Reverse(requested)
	entries, err := menu.selectEntries(requested)
	if err != nil || len(entries) != len(want) {
		t.Fatalf("select sauropods: %d entries: %v", len(entries), err)
	}
	var output bytes.Buffer
	if err := writeInventory(&output, entries); err != nil {
		t.Fatal(err)
	}
	var inventory []inventoryRow
	if err := json.Unmarshal(output.Bytes(), &inventory); err != nil {
		t.Fatal(err)
	}
	if len(inventory) != len(want) {
		t.Fatalf("inventory has %d rows, want %d", len(inventory), len(want))
	}
	plan := defaultPlan(t)
	for i, entry := range entries {
		if entry.Kind != want[i] || entry.Group != "Sauropods" || entry.Animal == nil ||
			entry.Animal.Family != "sauropod" || entry.Animal.File != "sauropod-profiles.yaml" ||
			entry.Animal.Species != want[i] || entry.source == "" || entry.checked == nil {
			t.Fatalf("wrong checked entry %d: %+v", i, entry)
		}
		row := inventory[i]
		if row.Kind != want[i] || row.Animal == nil || row.Animal.Species != want[i] ||
			row.Source != entry.source || row.Parts == 0 || row.Review != "pending" {
			t.Fatalf("wrong inventory row for %s: %+v", want[i], row)
		}
		object, err := plan.construct(entry, "quick")
		if err != nil || object == nil || object.faces == 0 {
			t.Fatalf("construct render selection %s: %v, %v", want[i], object, err)
		}
	}
}

func TestCatalogueSauropodRejectsUnsupportedOptions(t *testing.T) {
	original := "animal: {family: sauropod, file: sauropod-profiles.yaml, species: alamosaurus-sanjuanensis}"
	if !strings.Contains(string(menuRaw), original) {
		t.Fatal("missing sauropod fixture")
	}
	for _, option := range []string{"pose: swim", "count: 0", "seconds: 0", "phase: 0", "representation: skin", "study: default"} {
		t.Run(option, func(t *testing.T) {
			replacement := strings.TrimSuffix(original, "}") + ", " + option + "}"
			bad := strings.Replace(string(menuRaw), original, replacement, 1)
			if _, err := parseMenu("sauropod-option.yaml", []byte(bad)); err == nil {
				t.Fatalf("accepted unsupported sauropod option %s", option)
			}
		})
	}
	missing := strings.Replace(string(menuRaw), original, strings.Replace(original, "alamosaurus-sanjuanensis", "unknown-sauropod", 1), 1)
	if _, err := parseMenu("unknown-sauropod.yaml", []byte(missing)); err == nil {
		t.Fatal("accepted an unknown sauropod profile")
	}
}

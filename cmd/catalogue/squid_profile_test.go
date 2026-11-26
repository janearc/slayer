package main

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

func TestCatalogueSquidDescriptorBuildsCheckedPhase(t *testing.T) {
	zero, later := 0.0, 1.1
	books := &modelBooks{}
	first := entry{Kind: "squid-example", Animal: &animalSpec{Family: "squid", File: "squid-profiles.yaml", Species: "humboldt-squid", Phase: &zero}}
	second := entry{Kind: "squid-example", Animal: &animalSpec{Family: "squid", File: "squid-profiles.yaml", Species: "humboldt-squid", Phase: &later}}
	if err := first.bindAnimal(books); err != nil {
		t.Fatal(err)
	}
	if err := second.bindAnimal(books); err != nil {
		t.Fatal(err)
	}
	if first.source == "" || first.source == second.source {
		t.Fatal("phase did not change card source identity")
	}
	var specimens [][]float64
	for _, entry := range []entry{first, second} {
		row, err := entry.kind()
		if err != nil {
			t.Fatal(err)
		}
		made, err := manufacture.MakeWithComplexity(row, manufacture.Sow(3), "quick")
		if err != nil {
			t.Fatal(err)
		}
		parts, err := made.Things(maths.Local())
		if err != nil {
			t.Fatal(err)
		}
		skin, ok := parts[0].Shape.(*mesh.Mesh)
		if !ok {
			t.Fatalf("squid skin is %T", parts[0].Shape)
		}
		var coords []float64
		for _, p := range skin.Points() {
			coords = append(coords, p.X, p.Y, p.Z)
		}
		specimens = append(specimens, coords)
	}
	if len(specimens[0]) == 0 || len(specimens[0]) != len(specimens[1]) {
		t.Fatal("missing skin correspondence")
	}
	changed := false
	for i := range specimens[0] {
		if specimens[0][i] != specimens[1][i] {
			changed = true
			break
		}
	}
	if !changed {
		t.Fatal("phase did not change squid geometry")
	}
}

func TestCatalogueSquidDescriptorSchema(t *testing.T) {
	for _, tc := range []struct {
		options string
		valid   bool
	}{
		{"family: squid, file: squid-profiles.yaml, species: humboldt-squid", true},
		{"family: squid, file: squid-profiles.yaml, species: humboldt-squid, phase: 1.1", true},
		{"family: squid, file: squid-profiles.yaml, species: humboldt-squid, phase: null", false},
		{"family: squid, file: squid-profiles.yaml, species: humboldt-squid, seconds: 0", false},
		{"family: squid, file: squid-profiles.yaml, species: humboldt-squid, pose: swimming", false},
		{"family: fish, file: fish.yaml, species: pacific-sardine, phase: 0", false},
	} {
		card := fmt.Sprintf("\n  - kind: squid-test-card\n    group: test\n    about: A squid.\n    notes: A checked profile.\n    details: [quick]\n    animal: {%s}\n", tc.options)
		raw := append(bytes.Clone(menuRaw), []byte(card)...)
		_, err := parseMenu("menu.yaml", raw)
		if (err == nil) != tc.valid {
			t.Errorf("%s: valid=%v: %v", tc.options, tc.valid, err)
		}
	}
}

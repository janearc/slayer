package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// These are authored scene-replay choices, not measured tree or fire data.
// The independent uint64 C producer in testdata holds the established seed
// lanes for seed 7; explicit nonzero Y offsets also hold ignition placement.
func TestFrontPreparationKeepsTargetPositionsAndAuthoredSeedAssignments(t *testing.T) {
	given := frontOnFixture(t)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), "ignition-height: {value: 0, unit: m}",
		"ignition-height: {value: 3, unit: m}", 1)
	if text == string(raw) {
		t.Fatal("fixture lost its ignition-height setting")
	}
	if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	read := readBoundarySet(t, given.set)
	given.seed, given.exposure = 7, 1
	stage := staging{given: given, recipes: read, world: thing.NewWorld(),
		places: map[string]maths.Vec{"the tree": {}},
		fuels: map[string][]groveTree{"front fuel": {
			{at: maths.V(10, 4, 5)}, {at: maths.V(20, 9, 6)},
			{at: maths.V(30, 2, 7)}, {at: maths.V(40, 7, 8)},
		}}}
	if err := stage.prepareFronts(read); err != nil {
		t.Fatal(err)
	}
	if len(stage.fronts) != 1 {
		t.Fatalf("prepared %d fronts, want one", len(stage.fronts))
	}
	front := stage.fronts[0]
	want := map[maths.Vec]uint64{
		maths.V(10, 7, 5):  12918135221727111561,
		maths.V(20, 12, 6): 16731224329868871185,
		maths.V(30, 5, 7):  7862637804313477842,
		maths.V(40, 10, 8): 3467252261107883461,
	}
	seen := map[maths.Vec]bool{}
	for _, group := range front.plan.Groups() {
		for _, event := range group {
			seed, has := want[event.At]
			if !has || seen[event.At] || event.Seed != seed {
				t.Fatalf("target %v has seed %d, want %d; known=%v duplicate=%v",
					event.At, event.Seed, seed, has, seen[event.At])
			}
			seen[event.At] = true
		}
	}
	if len(seen) != len(want) {
		t.Fatalf("retained %d targets, want %d", len(seen), len(want))
	}
	seen = map[maths.Vec]bool{}
	for _, group := range front.groups {
		field := group.fire.field.(*flame.Field).Tick(3 * time.Second).(*flame.Field)
		for _, piece := range field.Pieces() {
			at := group.fire.place.Out(piece.At)
			seed, has := want[at]
			spec, _, _, _ := piece.Body.Samples()
			if !has || seen[at] || spec.Seed != seed {
				t.Fatalf("flame %v lost its planned seed %d: %d", at, seed, spec.Seed)
			}
			seen[at] = true
		}
	}
	if len(seen) != len(want) {
		t.Fatalf("prepared %d flames, want %d", len(seen), len(want))
	}
}

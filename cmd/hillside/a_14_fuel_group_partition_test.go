package main

import (
	"math"
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/scene/thing"
)

// Four 8-by-8 groups must cover a 16-by-16 bed without skipped or duplicate cells.
// Bind every actual rendered patch location to its retained solver-cell ID.
func TestFuelBedGroupsPartitionBothAxesAndBindEverySolverCell(t *testing.T) {
	given := fuelOnFixture(t)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), "      width: 8", "      width: 16", 1)
	text = strings.Replace(text, "      depth: 8", "      depth: 16", 1)
	if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	read := readBoundarySet(t, given.set)
	given.onGPU = false
	stage := staging{given: given, recipes: read, world: thing.NewWorld()}
	if err := stage.terrain(read); err != nil {
		t.Fatal(err)
	}
	if err := stage.prepareBeds(read); err != nil {
		t.Fatal(err)
	}
	defer (&liveScene{beds: stage.beds}).closeBeds()
	if len(stage.beds) != 1 || len(stage.beds[0].groups) != 4 {
		t.Fatal("bed did not retain four groups")
	}
	seen := map[int]bool{}
	for _, group := range stage.beds[0].groups {
		pieces := group.field.Pieces()
		if len(pieces) != 64 || len(group.indices) != 64 {
			t.Fatal("group lost its64-patch solver binding")
		}
		for i, piece := range pieces {
			point := group.anchor.Add(piece.At)
			east, north := stage.west+point.X, stage.north-point.Z
			x := int(math.Floor((east - 1046) / .5))
			z := int(math.Floor((8964 - north) / .5))
			if x < 0 || x >= 16 || z < 0 || z >= 16 {
				t.Fatalf("patch escaped bed at E%g N%g", east, north)
			}
			index := z*16 + x
			if group.indices[i] != index || seen[index] {
				t.Fatalf("patch(%d,%d) bound to cell%d, want unique%d", x, z, group.indices[i], index)
			}
			seen[index] = true
		}
	}
	if len(seen) != 256 {
		t.Fatalf("partition retained%d cells, want256", len(seen))
	}
}

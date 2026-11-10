package main

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/scene/biome"
	"github.com/janearc/slayer/scene/thing"
)

// Each region examines sixteen cells/candidates, but the shared scene allowance
// has thirty-one of each. An unreachable habitat band avoids constructing any
// tree and makes this a small preparation test, not a render.
const budgetRegion = `      seed: 1
      cell: {value: 50, unit: m}
      density: {value: 4, unit: ha^-1}
      variety: 1
      bounds: {west: 1000, east: 1100, north: 9000, south: 8900}
      above: 1000000000
      below: 10000000000
      mixture: {stones: 1}
`

func TestEcosystemStagingActuallySharesAndChargesItsRemainingWork(test *testing.T) {
	given := mustParse(test)
	block := "  recipes:\n    stones: {kind: cobble}\n" +
		"  ecosystems:\n    first:\n" + budgetRegion + "    second:\n" + budgetRegion +
		"  terrain:"
	given.set = onFixture(test, "  terrain:", block)
	read := readBoundarySet(test, given.set)
	stage := staging{given: given, recipes: read, world: thing.NewWorld()}
	if err := stage.terrain(read); err != nil {
		test.Fatal(err)
	}
	regions := read.Ecosystems()
	if len(regions) != 2 {
		test.Fatal("fixture lost its two regions")
	}
	query := biome.Rect{MinEast: 1000, MaxEast: 1100, MinSouth: -9000, MaxSouth: -8900}
	left := biome.Budget{Cells: 31, Candidates: 31}
	if err := stage.ecosystem(regions[0], query, &left); err != nil {
		test.Fatal(err)
	}
	if left.Cells != 15 || left.Candidates != 15 || stage.made.things != 0 {
		test.Fatal("edge and habitat rejects escaped shared accounting", left, stage.made)
	}
	err := stage.ecosystem(regions[1], query, &left)
	if err == nil || !strings.Contains(err.Error(), "15-cell work budget") {
		test.Fatal("the second region acquired a fresh allowance", left, err)
	}
}

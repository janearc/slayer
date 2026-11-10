package main

import (
	"math"
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/scene/thing"
)

// Decimal reference values come from the separate BigDecimal producer,
// using exact radius ratios from this authored 16-by-8 half-metre fixture.
// This holds initial conditions; it makes no experimental calibration claim.
func TestFuelBedIgnitionUsesCentredIsotropicDistancesAndKelvinExcess(t *testing.T) {
	given := fuelOnFixture(t)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), "      width: 8", "      width: 16", 1)
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
	if len(stage.beds) != 1 {
		t.Fatal("fixture lost bed")
	}
	state, err := stage.beds[0].model.At(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(state) != 128 {
		t.Fatalf("got%d cells, want128", len(state))
	}
	for _, sample := range []struct {
		x, z   int
		excess float64
	}{
		{7, 3, 857.1103199056587555}, {8, 4, 857.1103199056587555},
		{6, 3, 705.0397176547363831}, {7, 2, 705.0397176547363831},
		{9, 4, 705.0397176547363831}, {8, 5, 705.0397176547363831},
		{0, 0, 1.1195665912595677}, {15, 7, 1.1195665912595677},
	} {
		got := state[sample.z*16+sample.x].Excess
		if !(math.Abs(got-sample.excess) <= 1e-9) {
			t.Fatalf("cell(%d,%d) excess%.17gK, want%.17gK", sample.x, sample.z, got, sample.excess)
		}
	}
	for z := 0; z < 8; z++ {
		for x := 0; x < 16; x++ {
			cell := state[z*16+x]
			if cell.Fuel != 1 || !(cell.Excess >= 0) || math.IsInf(cell.Excess, 0) {
				t.Fatal("invalid initial fuel or excess")
			}
			horizontal := state[z*16+(15-x)].Excess
			vertical := state[(7-z)*16+x].Excess
			if !(math.Abs(cell.Excess-horizontal) <= 1e-9 && math.Abs(cell.Excess-vertical) <= 1e-9) {
				t.Fatalf("initial ignition is not centred at cell(%d,%d)", x, z)
			}
		}
	}
}

package main

import (
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/fuelbed"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// This is a renderer transfer contract, not a flame-height correlation.
// The authored curve reaches full strength at oxidation .35 and omits
// oxidation below .0007. Use actual solver cells and inspect actual fields.
func TestFuelBedFlamesFollowTheSolverAtTheAuthoredDisplayScale(t *testing.T) {
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
	given.onGPU, given.exposure = false, 1
	stage := staging{given: given, recipes: read, world: thing.NewWorld()}
	if err := stage.terrain(read); err != nil {
		t.Fatal(err)
	}
	if err := stage.prepareBeds(read); err != nil {
		t.Fatal(err)
	}
	defer (&liveScene{beds: stage.beds}).closeBeds()
	if len(stage.beds) != 1 || len(stage.beds[0].groups) != 2 {
		t.Fatal("fixture lost its two real flame groups")
	}
	bed := &stage.beds[0]
	clipped, partial, saturated := 0, 0, 0
	for _, at := range []time.Duration{0, 10 * time.Second} {
		state, err := bed.model.At(at.Seconds())
		if err != nil {
			t.Fatal(err)
		}
		type patch struct {
			fire flame.Fire
			at   maths.Vec
		}
		want := map[uint64]patch{}
		for _, group := range bed.groups {
			for i, piece := range group.field.Pieces() {
				fire, _, _, _ := piece.Body.Samples()
				reaction := fuelbed.Reaction(state[group.indices[i]], bed.spec.Activation)
				// These samples deliberately stay away from the exact cutoff;
				// this test is about the transfer across the whole fuel bed.
				if math.Abs(reaction-.0007) < 1e-12 {
					t.Fatal("fixture accidentally landed on the display boundary")
				}
				if reaction < .0007 {
					clipped++
					continue
				}
				strength := reaction * (20.0 / 7)
				if strength >= 1 {
					strength = 1
					saturated++
				} else {
					partial++
				}
				fire.Absorption *= strength
				fire.Height *= math.Pow(strength, 1.0/3)
				want[fire.Seed] = patch{fire: fire, at: group.anchor.Add(piece.At)}
			}
		}
		world := thing.NewWorld()
		if err := bed.addAt(world, &tally{}, at); err != nil {
			t.Fatal(err)
		}
		seen := map[uint64]bool{}
		for _, volume := range world.Volumes {
			field, ok := volume.Medium.(*flame.Field)
			if !ok || volume.Material == nil || volume.Material.Name != "surface flame" {
				t.Fatal("bed added a different medium instead of its flame field")
			}
			for _, piece := range field.Pieces() {
				fire, when, _, _ := piece.Body.Samples()
				expected, has := want[fire.Seed]
				if !has || seen[fire.Seed] {
					t.Fatalf("at %s: unexpected or duplicate active seed %d", at, fire.Seed)
				}
				seen[fire.Seed] = true
				if when != at || !(math.Abs(fire.Absorption-expected.fire.Absorption) <= 1e-12) ||
					!(math.Abs(fire.Height-expected.fire.Height) <= 1e-12) {
					t.Fatalf("at %s seed %d: flame scale/time differs from actual oxidation: got %g/%g/%s, want %g/%g/%s",
						at, fire.Seed, fire.Absorption, fire.Height, when, expected.fire.Absorption, expected.fire.Height, at)
				}
				if gap := volume.At(at).Out(piece.At).Sub(expected.at).Len(); !(gap <= 1e-10) {
					t.Fatalf("at %s seed %d: solver-bound flame moved %g m", at, fire.Seed, gap)
				}
				actual, reference := fire, expected.fire
				actual.Absorption, actual.Height = 0, 0
				reference.Absorption, reference.Height = 0, 0
				if actual != reference {
					t.Fatal("display scaling changed another physical or authored flame parameter")
				}
			}
		}
		if len(seen) != len(want) {
			t.Fatalf("at %s: drew %d solver-bound patches, want %d", at, len(seen), len(want))
		}
	}
	if clipped == 0 || partial == 0 || saturated == 0 {
		t.Fatalf("fixture lost a transfer regime: clipped %d, partial %d, saturated %d", clipped, partial, saturated)
	}
}

// Choose real checked fuel fractions that put a central solver cell just
// below, exactly at and just above the authored display cutoff. Searching
// the public reaction model selects inputs; it does not alter solver state.
func TestFuelBedKeepsTheInclusiveDisplayCutoffWithRealSolverCells(t *testing.T) {
	given := fuelOnFixture(t)
	given.onGPU, given.exposure = false, 1
	base, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	prepare := func(t *testing.T, text string) retainedBed {
		t.Helper()
		if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
		read := readBoundarySet(t, given.set)
		stage := staging{given: given, recipes: read, world: thing.NewWorld()}
		if err := stage.terrain(read); err != nil {
			t.Fatal(err)
		}
		if err := stage.prepareBeds(read); err != nil {
			t.Fatal(err)
		}
		if len(stage.beds) != 1 {
			t.Fatal("fixture lost its checked fuel bed")
		}
		t.Cleanup((&liveScene{beds: stage.beds}).closeBeds)
		return stage.beds[0]
	}
	control := prepare(t, string(base))
	initial, err := control.model.At(0)
	if err != nil {
		t.Fatal(err)
	}
	const index = 27
	cell := initial[index]
	if cell.Fuel != 1 {
		t.Fatal("reaction control lost its full fuel fraction")
	}
	unit := fuelbed.Reaction(cell, control.spec.Activation)
	if !(unit > .0007) {
		t.Fatal("central cell is too cold for the cutoff input search")
	}
	findFuel := func(t *testing.T, strength float64) float64 {
		t.Helper()
		low := strength * .35 / unit
		high := low
		for step := 0; step < 128; step++ {
			for _, candidate := range []float64{low, high} {
				probe := fuelbed.Cell{Excess: cell.Excess, Fuel: candidate}
				if fuelbed.Reaction(probe, control.spec.Activation)/.35 == strength {
					return candidate
				}
			}
			low, high = math.Nextafter(low, 0), math.Nextafter(high, 1)
		}
		t.Fatalf("no real fuel fraction found for exact display strength %.17g", strength)
		return 0
	}
	for _, sample := range []struct {
		name     string
		strength float64
		visible  bool
	}{
		{"below", math.Nextafter(.002, 0), false},
		{"at", .002, true},
		{"above", math.Nextafter(.002, 1), true},
	} {
		t.Run(sample.name, func(t *testing.T) {
			fuel := findFuel(t, sample.strength)
			text := strings.Replace(string(base), "      fuel: 1\n", fmt.Sprintf("      fuel: %.17g\n", fuel), 1)
			if text == string(base) {
				t.Fatal("fixture lost its fuel setting")
			}
			bed := prepare(t, text)
			state, err := bed.model.At(0)
			if err != nil {
				t.Fatal(err)
			}
			if state[index].Fuel != fuel || fuelbed.Reaction(state[index], bed.spec.Activation)/.35 != sample.strength {
				t.Fatal("actual checked solver did not retain the exact boundary input")
			}
			if len(bed.groups) != 1 || bed.groups[0].indices[index] != index {
				t.Fatal("fixture lost its central cell binding")
			}
			original, _, _, _ := bed.groups[0].field.Pieces()[index].Body.Samples()
			world := thing.NewWorld()
			if err := bed.addAt(world, &tally{}, 0); err != nil {
				t.Fatal(err)
			}
			found := 0
			for _, volume := range world.Volumes {
				field, ok := volume.Medium.(*flame.Field)
				if !ok {
					t.Fatal("bed added a different participating medium")
				}
				for _, piece := range field.Pieces() {
					fire, _, _, _ := piece.Body.Samples()
					if fire.Seed == original.Seed {
						found++
					}
				}
			}
			want := 0
			if sample.visible {
				want = 1
			}
			if found != want {
				t.Fatalf("strength %.17g: drew %d copies of the actual solver cell, want %d", sample.strength, found, want)
			}
		})
	}
}

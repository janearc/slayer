package main

import (
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/plume"
	"github.com/janearc/slayer/scene/thing"
)

// The independent C producer fixes this authored two-stage seed example.
// A public reference column also verifies that the seed changes the sampled
// smoke, rather than merely checking that two private integers agree.
func TestFireRecipeRetainsTheRequestedFlameAndSmokeSeed(t *testing.T) {
	given := fireOnFixture(t)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	night, err := os.ReadFile("../../doc/fire-night.yaml")
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Index(string(night), "      sparks:\n")
	if start < 0 {
		t.Fatal("night scene lost sparks")
	}
	end := strings.Index(string(night[start:]), "      light-cells:")
	if end < 0 {
		t.Fatal("night scene lost following light-cells")
	}
	text := strings.Replace(string(raw), "      flame:", string(night[start:start+end])+"      flame:", 1)
	if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	read := readBoundarySet(t, given.set)
	given.seed, given.exposure = 7, 1
	given.at = 1500 * time.Millisecond
	stage := staging{given: given, recipes: read, world: thing.NewWorld(), radii: map[string]float64{}}
	if _, err := stage.putFireRecipe("wood fire", 3, 0, maths.V(10, 2, 5), "fire"); err != nil {
		t.Fatal(err)
	}
	if len(stage.fires) != 1 {
		t.Fatal("fixture lost retained fire")
	}
	if len(stage.world.Lamps) == 0 || stage.made.lamps != len(stage.world.Lamps) {
		t.Fatalf("lamp tally %d differs from actual %d lamps", stage.made.lamps, len(stage.world.Lamps))
	}
	if len(stage.world.Things) == 0 || stage.made.sparks != len(stage.world.Things) {
		t.Fatalf("spark tally %d differs from actual %d particles", stage.made.sparks, len(stage.world.Things))
	}
	if stage.radii["fire"] != .45 {
		t.Fatalf("named fire lost its authored .45 m radius: %g", stage.radii["fire"])
	}
	body, has := stage.fires[0].field.(*flame.Body)
	if !has {
		t.Fatal("fixture did not retain a single flame body")
	}
	fire, _, _, _ := body.Samples()
	const expectedSeed uint64 = 14340413156236405150
	if fire.Seed != expectedSeed {
		t.Fatalf("flame seed%d, want%d", fire.Seed, expectedSeed)
	}
	row, has := read.Recipe("wood fire")
	if !has {
		t.Fatal("fixture lost recipe")
	}
	spec, has := row.Fire()
	if !has || spec.Smoke == nil {
		t.Fatal("fixture lost smoke")
	}
	smokeSpec := *spec.Smoke
	smokeSpec.Seed = expectedSeed
	reference, err := plume.New(smokeSpec)
	if err != nil {
		t.Fatal(err)
	}
	smokeSpec.Seed = 0
	unseeded, err := plume.New(smokeSpec)
	if err != nil {
		t.Fatal(err)
	}
	var actual *plume.Column
	for _, volume := range stage.world.Volumes {
		if column, has := volume.Medium.(*plume.Column); has {
			if actual != nil {
				t.Fatal("fixture has duplicate smoke columns")
			}
			actual = column
		}
	}
	if actual == nil {
		t.Fatal("fire recipe lost its actual smoke")
	}
	peak, seedGap := 0.0, 0.0
	for _, at := range []time.Duration{0, 700 * time.Millisecond, 2300 * time.Millisecond} {
		expected, plain, got := reference.Tick(at), unseeded.Tick(at), actual.Tick(at)
		for _, y := range []float64{1, 3, 7} {
			for _, x := range []float64{-.25, 0, .25} {
				for _, z := range []float64{-.25, 0, .25} {
					point := maths.V(x, y, z)
					want, value := expected.Density(point), got.Density(point)
					peak = math.Max(peak, want)
					seedGap = math.Max(seedGap, math.Abs(want-plain.Density(point)))
					if gap := math.Abs(want - value); !(gap <= 1e-12*math.Max(1e-30, want)) {
						t.Fatalf("seeded smoke at%s/%v: density%.17g, want%.17g", at, point, value, want)
					}
				}
			}
		}
	}
	if !(peak > 0 && seedGap > peak*1e-7) {
		t.Fatalf("fixture does not expose meaningful seed variation: gap%g peak%g", seedGap, peak)
	}
}

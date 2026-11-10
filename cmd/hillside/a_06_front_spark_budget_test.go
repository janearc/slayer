package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// An ordinary fire without a spark source reserves no particle slots.
func TestAbsentSparkSourceReservesNoParticleSlots(t *testing.T) {
	var source *retainedSparks
	if got := source.budget(); got != 0 {
		t.Fatalf("absent source reserves %d particle slots, want zero", got)
	}
}

// A checked front reserves four slots for its two-per-second, one-second
// source. Add a real static source to reach both sides of the scene limit.
func TestFrontSparkReservationKeepsTheExactAggregateLimit(t *testing.T) {
	given := frontOnFixture(t)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	block := `
      sparks:
        rate: {value: 2, unit: s^-1}
        lifetime: {value: 1, unit: s}
        radius: {value: 0.025, unit: m}
        release-height: {value: 3, unit: m}
        launch-speed: {value: 9, unit: m/s}
        spread: {value: 2, unit: m/s}
        updraft: {value: 8, unit: m/s}
        drag-time: {value: 0.8, unit: s}
        gravity: {value: 9.80665, unit: m/s^2}
        temperature: {value: 1650, unit: K}
        end-temperature: {value: 900, unit: K}
`
	text := strings.Replace(string(raw), "      flame:", strings.TrimPrefix(block, "\n")+"      flame:", 1)
	text = strings.Replace(text, "kind: fern, count: 4", "kind: fern, count: 1", 1)
	if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	read := readBoundarySet(t, given.set)
	row, has := read.Recipe("wood fire")
	if !has {
		t.Fatal("fixture lost fire recipe")
	}
	spec, has := row.Fire()
	if !has || spec.Sparks == nil || spec.Sparks.Rate != 2 || spec.Sparks.Life != time.Second {
		t.Fatal("fixture lost its two-per-second, one-second source")
	}
	given.exposure = 1
	for _, sample := range []struct {
		rate  float64
		total int
		okay  bool
	}{
		{8185, 8191, true}, {8186, 8192, true}, {8187, 8193, false},
	} {
		t.Run(fmt.Sprintf("%d total", sample.total), func(t *testing.T) {
			stage := staging{given: given, recipes: read, world: thing.NewWorld(),
				places: map[string]maths.Vec{"the tree": {}},
				fuels:  map[string][]groveTree{"front fuel": {{at: maths.Vec{}}}}}
			static := spec
			sparks := *spec.Sparks
			sparks.Rate = sample.rate
			static.Sparks = &sparks
			source, err := stage.prepareSparks(static, maths.Subspace{})
			if err != nil {
				t.Fatalf("valid individual static source: %v", err)
			}
			if source.budget() != sample.total-4 {
				t.Fatal("fixture did not reserve the expected static slots")
			}
			stage.fires = append(stage.fires, retainedFire{sparks: source})
			err = stage.prepareFronts(read)
			if sample.okay {
				if err != nil {
					t.Fatalf("%d sparks should fit: %v", sample.total, err)
				}
				if len(stage.fronts) != 1 || len(stage.fronts[0].sparks) != 1 {
					t.Fatal("front did not retain its actual source")
				}
			} else if err == nil || !strings.Contains(err.Error(), "front a passing front exceeds the budget of 8192 sparks") {
				t.Fatalf("%d sparks should exceed the scene limit: %v", sample.total, err)
			}
		})
	}
}

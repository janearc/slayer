package main

import (
	"strings"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/embers"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

// Two real sources each fit their own limit. Hold the documented scene
// budget independently, including acceptance at exactly 8192 sparks.
func TestStaticSparkBudgetAcceptsItsExactDocumentedLimit(t *testing.T) {
	for _, sample := range []struct {
		name string
		rate float64
		want int
		okay bool
	}{
		{"below", 4093, 8191, true},
		{"at", 4094, 8192, true},
		{"above", 4095, 8193, false},
	} {
		t.Run(sample.name, func(t *testing.T) {
			stage := &staging{given: asked{exposure: 1}, world: &thing.World{}}
			for _, rate := range []float64{4094, sample.rate} {
				spec := manufacture.BurnSpec{Sparks: &embers.Spec{
					Rate: rate, Life: time.Second, Drag: time.Second,
					Radius: measure.Metres(.02), Hot: measure.Kelvin(1700),
					Cool: measure.Kelvin(900),
				}}
				sparks, err := stage.prepareSparks(spec, maths.Subspace{})
				if err != nil {
					t.Fatalf("valid individual source: %v", err)
				}
				stage.fires = append(stage.fires, retainedFire{sparks: sparks})
			}
			count := stage.fires[0].sparks.budget() + stage.fires[1].sparks.budget()
			if count != sample.want {
				t.Fatalf("fixture reserves %d sparks, want %d", count, sample.want)
			}
			err := stage.prepareFronts(set.Set{})
			if sample.okay {
				if err != nil {
					t.Fatalf("%d sparks should fit the scene budget: %v", sample.want, err)
				}
			} else if err == nil || !strings.Contains(err.Error(),
				"scene exceeds the budget of 8192 sparks") {
				t.Fatalf("%d sparks should exceed the scene budget: %v", sample.want, err)
			}
		})
	}
}

package main

import (
	"os"
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

func TestFrontRefusesAggregateSparkBudgetFromRealYAML(t *testing.T) {
	given := frontOnFixture(t)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	block := `      sparks:
        rate: {value: 1000, unit: s^-1}
        lifetime: {value: 3, unit: s}
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
	text := strings.Replace(string(raw), "      flame:", block+"      flame:", 1)
	if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	// Every emitter is within its own 8192 limit. Four overlapping
	// targets plus the existing fire exceed the aggregate scene budget.
	if _, _, _, err := staged(given); err == nil || !strings.Contains(err.Error(), "front a passing front exceeds the budget of 8192 sparks") {
		t.Fatalf("aggregate front budget: %v", err)
	}
}

func TestStaticSourcesAlsoObserveTheAggregateSparkBudget(t *testing.T) {
	stage := &staging{given: asked{exposure: 1}, world: &thing.World{}}
	spec := manufacture.BurnSpec{Sparks: &embers.Spec{Rate: 1000, Life: 5 * time.Second, Drag: time.Second,
		Radius: measure.Metres(.02), Hot: measure.Kelvin(1700), Cool: measure.Kelvin(900)}}
	for range 2 {
		sparks, err := stage.prepareSparks(spec, maths.Subspace{})
		if err != nil {
			t.Fatal(err)
		}
		stage.fires = append(stage.fires, retainedFire{sparks: sparks})
	}
	if err := stage.prepareFronts(set.Set{}); err == nil || !strings.Contains(err.Error(), "scene exceeds the budget of 8192 sparks") {
		t.Fatalf("static aggregate budget: %v", err)
	}
}

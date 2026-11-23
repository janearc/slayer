package session

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

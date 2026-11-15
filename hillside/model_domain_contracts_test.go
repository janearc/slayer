package hillside

import (
	"errors"
	"testing"
	"time"

	"github.com/janearc/slayer/firefront"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/plume"
	"github.com/janearc/slayer/scene/thing"
)

// This is a retained-model unit boundary, not a scene-reader refusal.
func TestFrontSmokeRebuildPreservesAnInvalidSourceCause(t *testing.T) {
	plan, err := firefront.New(firefront.Spec{Direction: maths.XAxis, Speed: 1, Cell: 1, Duration: time.Second}, []firefront.Target{{}})
	if err != nil {
		t.Fatal(err)
	}
	invalid := plume.Fire{}
	_, cause := plume.New(invalid)
	if cause == nil {
		t.Fatal("invalid plume control accepted")
	}
	front := retainedFront{plan: plan, peakFuel: 1, smoke: &thing.Volume{}, smokeSpec: &invalid}
	if err := front.addAt(thing.NewWorld(), &tally{}, 0); !errors.Is(err, cause) {
		t.Fatalf("front lost plume cause: %v", err)
	}
}

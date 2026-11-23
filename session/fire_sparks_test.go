package session

import (
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/embers"
	"github.com/janearc/slayer/manufacture"

	"github.com/janearc/slayer/maths"
)

func TestFinishedFuelRetainsAirborneSparksWithoutNewBirths(t *testing.T) {
	stage := &staging{given: asked{exposure: 1}}
	spec := manufacture.BurnSpec{Sparks: &embers.Spec{Rate: 10, Life: 3 * time.Second, Drag: time.Second,
		Radius: measure.Metres(.02), Hot: measure.Kelvin(1700), Cool: measure.Kelvin(900)}}
	sparks, err := stage.prepareSparks(spec, maths.Subspace{})
	if err != nil {
		t.Fatal(err)
	}
	during := sparks.thingsIn(500*time.Millisecond, time.Second)
	airborne := sparks.thingsIn(2*time.Second, time.Second)
	dying := sparks.thingsIn(3500*time.Millisecond, time.Second)
	gone := sparks.thingsIn(4*time.Second, time.Second)
	if len(during) != 5 || len(airborne) != 10 || len(dying) != 5 || len(gone) != 0 {
		t.Fatalf("birth/death window lost: %d %d %d %d", len(during), len(airborne), len(dying), len(gone))
	}
}

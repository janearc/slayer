package session

import (
	"errors"
	"io"
	"math"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/globe"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// a bare hillside as asked, or the test fails.
func mustBare(t *testing.T, given asked) *thing.World {
	t.Helper()
	world, err := bare(given, 82.5)
	if err != nil {
		t.Fatal(err)
	}
	return world
}

// -earth stands the hillside on a globe of the earth's size and puts
// the sun and its fill at an endless distance, as bright on the ground
// as the lamps they replace: the earth moves the sun, not the day.
func TestOnTheEarthTheSunIsDistantAndAsBright(t *testing.T) {
	onGlobe := smallAsked()
	onGlobe.earth = true
	flat, round := mustBare(t, smallAsked()), mustBare(t, onGlobe)
	if len(round.Lamps) != len(flat.Lamps) {
		t.Fatalf("%d lamps on the earth, %d without", len(round.Lamps),
			len(flat.Lamps))
	}
	for i, lamp := range round.Lamps {
		if _, distant := lamp.(emit.Distant); !distant {
			t.Errorf("lamp %d on the earth is a %T", i, lamp)
		}
		_, _, near := flat.Lamps[i].Light(maths.Vec{}, maths.Up)
		_, _, far := lamp.Light(maths.Vec{}, maths.Up)
		if !(math.Abs(far.R/near.R-1) <= 1e-9) {
			t.Errorf("lamp %d lands %v on the earth, %v without", i, far,
				near)
		}
	}
	globes := 0
	for _, each := range round.Things {
		if _, isGlobe := each.Shape.(globe.Globe); isGlobe {
			globes++
		}
	}
	if globes != 1 {
		t.Errorf("the earth has %d globes under it", globes)
	}
}

// -sky black is no sky at all, as above the air; -sky painted is the
// usual one; anything else is refused.
func TestTheSkyIsAsChosen(t *testing.T) {
	black, painted := smallAsked(), smallAsked()
	black.sky, painted.sky = "black", "painted"
	if world := mustBare(t, black); world.Sky != nil {
		t.Errorf("a black sky is a %T", world.Sky)
	}
	if world := mustBare(t, painted); world.Sky == nil {
		t.Error("a painted sky is none")
	}
	if _, err := parse([]string{"-sky", "green"}, io.Discard); !errors.Is(
		err, errAsked) {
		t.Errorf("a green sky is taken: %v", err)
	}
}

// tenDegrees is how far below level a test tilts the eye, in degrees.
const tenDegrees = 10.0

// -tilt is how far below level the eye at -from looks: ten degrees
// from a hundred kilometres is at the horizon's foot.
func TestFromAHeightTheEyeLooksAsTiltedAsAsked(t *testing.T) {
	given := smallAsked()
	given.from, given.tilt = farHeight, tenDegrees
	eye := eyeFor(given, 82.5)
	want := -math.Sin(tenDegrees * math.Pi / 180)
	if !(math.Abs(eye.Forward().Y-want) <= 1e-9) {
		t.Errorf("tilted ten degrees the eye looks %v", eye.Forward())
	}
}

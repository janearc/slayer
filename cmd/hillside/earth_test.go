package main

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
func mustBare(t *testing.T, arguments ...string) *thing.World {
	t.Helper()
	world, err := bare(mustParse(t, arguments...), 82.5)
	if err != nil {
		t.Fatal(err)
	}
	return world
}

// -earth stands the hillside on a globe of the earth's size and puts
// the sun and its fill at an endless distance, as bright on the ground
// as the lamps they replace: the earth moves the sun, not the day.
func TestOnTheEarthTheSunIsDistantAndAsBright(t *testing.T) {
	flat, round := mustBare(t), mustBare(t, "-earth")
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
		if math.Abs(far.R/near.R-1) > 1e-9 {
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
	if world := mustBare(t, "-sky", "black"); world.Sky != nil {
		t.Errorf("a black sky is a %T", world.Sky)
	}
	if world := mustBare(t, "-sky", "painted"); world.Sky == nil {
		t.Error("a painted sky is none")
	}
	if _, err := parse([]string{"-sky", "green"}, io.Discard); !errors.Is(
		err, errAsked) {
		t.Errorf("a green sky is taken: %v", err)
	}
}

// -tilt is how far below level the eye at -from looks: ten degrees
// from a hundred kilometres is at the horizon's foot.
func TestFromAHeightTheEyeLooksAsTiltedAsAsked(t *testing.T) {
	eye := eyeFor(mustParse(t, "-from", "100000", "-tilt", "10"), 82.5)
	want := -math.Sin(10 * math.Pi / 180)
	if math.Abs(eye.Forward().Y-want) > 1e-9 {
		t.Errorf("tilted ten degrees the eye looks %v", eye.Forward())
	}
}

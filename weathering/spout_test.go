package weathering_test

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/weathering"
)

// the numbers the spout tests lay a streak out by, each a choice of the
// tests
const (
	// spoutStandsOut is how far, in metres, the test spout stands out of
	// its wall, as a gargoyle's mouth stands out of a cornice.
	spoutStandsOut = 3.0
	// spoutHeight is how high, in metres, the test spout is.
	spoutHeight = 10.0
	// streakLength is how long, in metres, the test streak runs.
	streakLength = 12.0
)

// spoutLayer is a layer with one spout, at the height and distance out
// from the wall at z of nought, with the one streak agent.
func spoutLayer() (weathering.Layer, weathering.Agent) {
	agent := weathering.Agent{Kind: weathering.Spout, Name: "spout",
		Strength: 1, Spacing: 1, Spread: 0.1, Length: streakLength,
		Level: 0.2, Grain: 1, Share: 1}
	layer := layerOf(weathering.NewShelter(testBlock), agent)
	layer.Spouts = []maths.Vec{maths.V(0, spoutHeight, spoutStandsOut)}
	return layer, agent
}

// streakAt is how much of the streak is on the wall at a place.
func streakAt(layer weathering.Layer, agent weathering.Agent,
	across, height float64) float64 {
	return layer.Amount(agent, &weathering.Site{
		Point: maths.V(across, height, 0), Normal: faceSouth, Name: "wall"})
}

func TestAStreakRunsDownTheWallBelowASpoutThatStandsOutFromIt(t *testing.T) {
	layer, agent := spoutLayer()
	if got := streakAt(layer, agent, 0, spoutHeight-2); !(got > 0) {
		t.Errorf("no streak two metres below the spout: %v", got)
	}
}

func TestTheStreakFadesAsItFallsAndStopsAtItsLength(t *testing.T) {
	layer, agent := spoutLayer()
	near := streakAt(layer, agent, 0, spoutHeight-1)
	far := streakAt(layer, agent, 0, spoutHeight-streakLength+1)
	if !(near > far) {
		t.Errorf("%v near and %v far: it does not fade", near, far)
	}
	if got := streakAt(layer, agent, 0, spoutHeight-streakLength-1); got != 0 {
		t.Errorf("%v beyond its length", got)
	}
}

func TestThereIsNoStreakAboveASpoutOrFarToItsSide(t *testing.T) {
	layer, agent := spoutLayer()
	if got := streakAt(layer, agent, 0, spoutHeight+1); got != 0 {
		t.Errorf("%v above the spout", got)
	}
	if got := streakAt(layer, agent, 5, spoutHeight-3); got != 0 {
		t.Errorf("%v five metres to the side", got)
	}
}

func TestAWallFacingAwayFromTheSpoutHasNoStreak(t *testing.T) {
	layer, agent := spoutLayer()
	got := layer.Amount(agent, &weathering.Site{
		Point:  maths.V(0, spoutHeight-2, 0),
		Normal: faceNorth, Name: "wall"})
	if got != 0 {
		t.Errorf("%v on the face turned from the spout", got)
	}
}

func TestAFloorIsNoWallForAStreak(t *testing.T) {
	layer, agent := spoutLayer()
	got := layer.Amount(agent, &weathering.Site{
		Point:  maths.V(0, spoutHeight-2, 1),
		Normal: faceUp, Name: "wall"})
	if got != 0 {
		t.Errorf("%v on a surface that faces up", got)
	}
}

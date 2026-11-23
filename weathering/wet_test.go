package weathering_test

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/weathering"
)

// the numbers the wet tests lay their agent out by, each a choice of the
// tests
const (
	// wetReach is how high the test splash goes, in metres.
	wetReach = 1.5
	// wetLow and wetHigh are a height low on a wall and one far above the
	// splash, in metres.
	wetLow  = 0.1
	wetHigh = 6.0
	// wetOpen is a place on the ground clear of the test ledge, in metres
	// along z.
	wetOpen = 5.0
	// wetUnder is a place on the ground under the test ledge.
	wetUnder = 0.2
)

// wetAgent is damp low on walls and on up faces, on whatever is named wall.
func wetAgent() weathering.Agent {
	return weathering.Agent{Kind: weathering.Wet, Name: "damp",
		On: []string{"wall"}, Strength: 0.5, Settle: 5, Reach: wetReach,
		Tilt: 0.5, Level: 0.3, Share: 1, Shelter: 3}
}

// wallSite is a site on a wall facing south, at a height.
func wallSite(height float64) *weathering.Site {
	return &weathering.Site{Point: maths.V(0, height, 0), Normal: faceSouth,
		Name: "wall"}
}

// the splash of a wet street is thickest at the ground and thins with
// height until the wall is dry.
func TestAWallIsDampestLowDown(t *testing.T) {
	agent := wetAgent()
	layer := layerOf(weathering.NewShelter(testBlock), agent)
	low, high := layer.Amount(agent, wallSite(wetLow)),
		layer.Amount(agent, wallSite(wetHigh))
	if !(low > 0) {
		t.Fatalf("the foot of the wall is dry: %v", low)
	}
	if !(low > 10*high) {
		t.Errorf("damp %v low down and %v high up", low, high)
	}
}

// an agent with no reach has no splash: its walls are dry, though a ledge
// open to the sky is not.
func TestAWetAgentWithNoReachLeavesWallsDry(t *testing.T) {
	agent := wetAgent()
	agent.Reach = 0
	layer := layerOf(weathering.NewShelter(testBlock), agent)
	if got := layer.Amount(agent, wallSite(wetLow)); got != 0 {
		t.Errorf("a wall with no splash is damp: %v", got)
	}
	ledge := &weathering.Site{Point: maths.V(0, 0.05, wetOpen),
		Normal: faceUp, Name: "wall"}
	if got := layer.Amount(agent, ledge); !(got > 0) {
		t.Errorf("rain left an open face dry: %v", got)
	}
}

// rain lands on a face that points up with open sky over it, and not on
// one under a ledge.
func TestRainWetsAnOpenFaceAndNotAShelteredOne(t *testing.T) {
	agent := wetAgent()
	layer := layerOf(shelterWithLedge("ledge"), agent)
	open := &weathering.Site{Point: maths.V(0, 0.05, wetOpen), Normal: faceUp,
		Name: "wall"}
	under := &weathering.Site{Point: maths.V(0, 0.05, wetUnder),
		Normal: faceUp, Name: "wall"}
	if !(layer.Amount(agent, open) > 0) {
		t.Error("an open face is dry")
	}
	if got := layer.Amount(agent, under); got != 0 {
		t.Errorf("a sheltered face is damp: %v", got)
	}
}

// damp lands only on what it is asked to: glass takes none.
func TestDampLandsOnlyOnWhatItIsAskedTo(t *testing.T) {
	agent := wetAgent()
	layer := layerOf(weathering.NewShelter(testBlock), agent)
	site := wallSite(wetLow)
	site.Name = "glass"
	if got := layer.Amount(agent, site); got != 0 {
		t.Errorf("glass is damp: %v", got)
	}
}

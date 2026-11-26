package cities

import (
	"math"
	"testing"
)

// hubSpan is the span, in metres, the hub bridge tests draw: the span of
// the bridge in the city by the bay.
const hubSpan = 700

// hubDrawn is a hub bridge of a seed drawn at the quick level.
func hubDrawn(t *testing.T, seed Seed) []Rendered {
	t.Helper()
	bridge, err := NewHubBridge(seed, Irregularity{Value: 0.5}, hubSpan)
	if err != nil {
		t.Fatal(err)
	}
	design, err := bridge.Design("quick", 0)
	if err != nil {
		t.Fatal(err)
	}
	drawn, err := design.Rendered("quick")
	if err != nil {
		t.Fatal(err)
	}
	return drawn
}

// A hub bridge is a tower whose body stands over a void on four legs, and
// two decks that run whole from one shore to the other through the void.
func TestAHubBridgeIsATowerOnFourLegsWithTheDecksThroughItsVoid(t *testing.T) {
	drawn := hubDrawn(t, 3)
	if legs := len(named(drawn, "tower leg")); legs != 4 {
		t.Errorf("%d legs, wanted 4", legs)
	}
	body, legs := named(drawn, "tower"), named(drawn, "tower leg")
	low, high := boundsOf(body)
	if math.Abs(low.Y-hubVoid) > 1e-6 || math.Abs(high.Y-hubTall) > 1e-6 {
		t.Errorf("the body stands from %v to %v m, wanted %v to %v", low.Y,
			high.Y, hubVoid, hubTall)
	}
	decks := named(drawn, "deck")
	if len(decks) != 2 {
		t.Fatalf("%d decks, wanted 2", len(decks))
	}
	for _, deck := range decks {
		low, high := boundsOf([]Rendered{deck})
		if math.Abs(high.X-low.X-hubSpan) > 1e-6 {
			t.Errorf("a deck runs %v m, wanted the whole span %v", high.X-low.X,
				float64(hubSpan))
		}
		if !(high.Y < hubVoid) {
			t.Errorf("a deck's top is %v m, over the void's %v", high.Y,
				float64(hubVoid))
		}
	}
	legLow, legHigh := boundsOf(legs)
	if legLow.Y != 0 || math.Abs(legHigh.Y-hubVoid) > 1e-6 {
		t.Errorf("the legs stand from %v to %v m", legLow.Y, legHigh.Y)
	}
}

// Anti: nothing of the bridge passes into a leg. Every storey and window of
// the arms lies across the span inside the legs' inner faces.
// anti-test of TestAHubBridgeIsATowerOnFourLegsWithTheDecksThroughItsVoid
func TestNothingOfAHubBridgePassesIntoALeg(t *testing.T) {
	drawn := hubDrawn(t, 3)
	inner := hubWide/2 - hubLeg
	for _, name := range []string{"arm storey", "window", "lit window",
		"deck"} {
		parts := named(drawn, name)
		if len(parts) == 0 {
			t.Errorf("no %q", name)
			continue
		}
		low, high := boundsOf(parts)
		if !(low.Z > -inner) || !(high.Z < inner) {
			t.Errorf("%q reaches %v to %v across, the legs' inner faces are "+
				"at %v", name, low.Z, high.Z, inner)
		}
	}
}

// A hub bridge's windows are lit by a draw that is the same for the same
// window every time, and lights about the share it says.
func TestAHubBridgesWindowsAreLitTheSameEveryTime(t *testing.T) {
	const bays = 4000
	lit := 0
	for bay := range bays {
		first := hubSpot(3, 1, 2, 1, bay)
		if again := hubSpot(3, 1, 2, 1, bay); again != first {
			t.Fatalf("bay %d drew %v then %v", bay, first, again)
		}
		if first < 0 || first >= 1 {
			t.Fatalf("bay %d drew %v, outside nought to one", bay, first)
		}
		if first < hubLitShare {
			lit++
		}
	}
	share := float64(lit) / bays
	if math.Abs(share-hubLitShare) > 0.03 {
		t.Errorf("%.3f of windows lit, wanted about %v", share, hubLitShare)
	}
}

// Anti: another seed, arm, storey or face lights other windows.
// anti-test of TestAHubBridgesWindowsAreLitTheSameEveryTime
func TestAnotherHubWindowDrawsOtherwise(t *testing.T) {
	base := hubSpot(3, 1, 2, 1, 7)
	for name, other := range map[string]float64{
		"seed":   hubSpot(4, 1, 2, 1, 7),
		"arm":    hubSpot(3, -1, 2, 1, 7),
		"storey": hubSpot(3, 1, 3, 1, 7),
		"face":   hubSpot(3, 1, 2, 2, 7),
		"bay":    hubSpot(3, 1, 2, 1, 8),
	} {
		if other == base {
			t.Errorf("another %s drew the same %v", name, base)
		}
	}
}

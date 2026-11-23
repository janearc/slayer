package manufacture

import (
	"errors"
	"math"
	"testing"
)

// epsilon is how near a bound must be to a figure to be it, in metres.
const epsilon = 1e-6

func TestTheWinnebagoKindIsOneGroupForEachOfFiveMaterials(t *testing.T) {
	made := mustMake(t, "winnebago", 1)
	if got := len(made.slots()); got != 5 {
		t.Errorf("%d materials, want body, cab, tyre, glazing and entry", got)
	}
	bounds := made.Bounds()
	if math.Abs(bounds.Min.Y) > epsilon || bounds.Max.Y < 3.3-epsilon {
		t.Errorf("the winnebago spans %v to %v up, want the ground to "+
			"its roof", bounds.Min.Y, bounds.Max.Y)
	}
}

func TestTheWinnebagosWheelsAreOneMeshPlacedSixTimes(t *testing.T) {
	made := mustMake(t, "winnebago", 1)
	most := 0
	for _, parts := range made.groupParts {
		most = max(most, parts)
	}
	if most != 7 {
		t.Errorf("the largest group has %d members, want the windows and "+
			"the windscreen, seven", most)
	}
	sixes := 0
	for _, parts := range made.groupParts {
		if parts == 6 {
			sixes++
		}
	}
	if sixes != 1 {
		t.Errorf("%d groups of six, want the one of wheels", sixes)
	}
}

func TestAWinnebagoNeedsASownSeed(t *testing.T) {
	if _, err := (winnebagoRow{}).made(seed{}); !errors.Is(err, errGood) {
		t.Errorf("an unsown seed made a winnebago: %v", err)
	}
}

func TestAWinnebagoIsMadeAtEveryDetailLevel(t *testing.T) {
	for _, level := range []string{"hull", "clumps", "quick", "demo"} {
		if _, err := (winnebagoRow{}).madeAt(sow(1), level); err != nil {
			t.Errorf("%s: %v", level, err)
		}
	}
}

func TestAWinnebagoAtNoDetailLevelIsRefused(t *testing.T) {
	if _, err := (winnebagoRow{}).madeAt(sow(1), "bespoke"); err == nil {
		t.Error("a winnebago was made at a level no data names")
	}
}

package terrain

import (
	"math"
	"testing"
)

// riseTolerance is how near a worked rise must come, in metres a metre;
// picked, far under any steepness a stand would say.
const riseTolerance = 1e-12

// planeWindow is three by three points a metre apart on a plane that
// rises east by one share of a metre in each metre and south by another.
func planeWindow(east, south float64) Window {
	window := Window{step: 1, across: 3, down: 3}
	for down := range 3 {
		for across := range 3 {
			window.heights = append(window.heights,
				east*float64(across)+south*float64(down))
		}
	}
	return window
}

func TestARiseIsTheTwoSlopesTakenTogether(t *testing.T) {
	window := planeWindow(0.3, 0.4)
	for _, point := range [][2]int{{1, 1}, {0, 0}, {2, 2}, {0, 2}} {
		got := window.Rise(point[0], point[1])
		if math.Abs(got-0.5) > riseTolerance {
			t.Errorf("at %v the rise is %v, not 0.5", point, got)
		}
	}
}

func TestARiseFallingAwayIsAsSteepAsOneClimbing(t *testing.T) {
	climbing := planeWindow(0.6, 0).Rise(1, 1)
	falling := planeWindow(-0.6, 0).Rise(1, 1)
	if math.Abs(climbing-falling) > riseTolerance {
		t.Errorf("ground climbing 0.6 rises %v and falling 0.6 rises %v",
			climbing, falling)
	}
}

func TestLevelGroundHasNoRise(t *testing.T) {
	if got := planeWindow(0, 0).Rise(1, 1); got != 0 {
		t.Errorf("level ground rises %v", got)
	}
}

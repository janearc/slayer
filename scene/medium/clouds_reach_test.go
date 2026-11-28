package medium

import (
	"testing"
)

// reachHigh and reachLow are how far up the layer, as shares of it, a test
// asks a patch whether it is there: near its top, and near its foot.
const (
	reachHigh    = 0.85
	reachLow     = 0.05
	reachSamples = 4000
)

// A patch is as tall as it is thick: where there is cloud near the top of
// the layer there is cloud at its foot, but a thin patch has none up there,
// so there are directions clouded low and clear high, and none the other
// way about.
func TestAThickerCloudReachesHigher(t *testing.T) {
	cloudy := cloudsLayer(t)
	span := cloudy.High - cloudy.Low
	atShare := func(direction float64, place int) float64 {
		at := scatter(t, place).Scale(cloudy.Radius + cloudy.Low + span*direction)
		return cloudy.Density(at)
	}
	lowOnly := 0
	for place := range reachSamples {
		high, low := atShare(reachHigh, place), atShare(reachLow, place)
		if high > nought && !(low > nought) {
			t.Fatalf("direction %d is clouded high, %v, and clear low", place,
				high)
		}
		if low > nought && high == nought {
			lowOnly++
		}
	}
	if lowOnly == 0 {
		t.Error("no cloud is thin enough to stop short of the top")
	}
}

// Anti-test: a layer with no cover has no cloud at any height, so the
// reach is no way round it.
func TestNoCoverMeansNoReach(t *testing.T) {
	bare := cloudsLayer(t)
	bare.Cover = nought
	span := bare.High - bare.Low
	for place := range reachSamples {
		at := scatter(t, place).Scale(bare.Radius + bare.Low + span*reachLow)
		if bare.Density(at) != nought {
			t.Fatalf("direction %d is clouded with no cover", place)
		}
	}
}

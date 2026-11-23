package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// A picture meter draws one ray a pixel and takes the log-average of the
// pixels' luminance. A pixel of haze is the light its steps sum, so the
// meter reads rightly only if the light a step picks is not far under the
// light of every lamp at that point, for most starts. These tests take a
// step's start as a pixel's, and read the log-average over many.

// meterFactor is how far the log-average of one-pick samples may be from
// the light of every lamp, either way: a factor of two, the least a
// meter's exposure is off by that anyone sees. picked.
const meterFactor = 2

// meterFloor is the least luminance a sample is read at, so that a sample
// of no light has a logarithm. It is the picture meter's own floor.
const meterFloor = 1e-6

// logAverageOf is the log-average luminance of the light a pick takes at a
// point, over splitStarts starts spread evenly from nothing to one.
func logAverageOf(point maths.Vec, pick func(start float64) []lit) float64 {
	sum := 0.0
	for each := range splitStarts {
		light := 0.0
		for _, one := range pick((float64(each) + 0.5) / splitStarts) {
			light += one.light.Luminance() * one.counts
		}
		sum += math.Log(math.Max(light, meterFloor))
	}
	return math.Exp(sum / splitStarts)
}

// TestTheSplitReadsAsBrightAsTheLampsToAOneRayMeter says that a near lamp
// among far ones, in a wide box that holds the point, does not starve the
// meter: the log-average of the light the split takes at the point, one
// start a pixel, is within meterFactor of the light of every lamp.
func TestTheSplitReadsAsBrightAsTheLampsToAOneRayMeter(t *testing.T) {
	scene := nearAmongFar(t)
	point := maths.V(0, 0, 0)
	got := logAverageOf(point, func(start float64) []lit {
		return scene.splitFrom(nil, point, maths.Vec{}, start,
			lampSplitMost)
	})
	every := luminanceAt(scene, point)
	if got < every/meterFactor || got > every*meterFactor {
		t.Errorf("the meter reads %g of lamps lighting %g", got, every)
	}
}

// TestAWalkByPowerAloneStarvesAOneRayMeterOfTheNearLamp is the control of
// the test above: the meter tells a pick that starves it. A walk by
// power alone (powerWalk, a fault made on purpose) takes the far bright
// lamp 300 starts in 303 and reads 3.4e-5 there, where every lamp
// together lights the origin 1.15e-3: the log-average of its samples is
// about a thirtieth of the lamps' light, far under meterFactor.
func TestAWalkByPowerAloneStarvesAOneRayMeterOfTheNearLamp(t *testing.T) {
	scene := nearAmongFar(t)
	point := maths.V(0, 0, 0)
	got := logAverageOf(point, func(start float64) []lit {
		return powerWalk(scene, point, start)
	})
	every := luminanceAt(scene, point)
	if !(got < every/meterFactor) {
		t.Errorf("the meter reads %g of lamps lighting %g", got, every)
	}
}

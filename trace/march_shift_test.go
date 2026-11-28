package trace

import (
	"math"
	"testing"
)

// The shift is a share of a step: from nought up to under one, over the
// whole step, and different for different fine pixels, so the steps of a
// coarse march do not line up across a picture as arcs.
func TestEveryPixelShiftsItsStepsItsOwnWay(t *testing.T) {
	seen := map[float64]bool{}
	var least, most = 1.0, 0.0
	for y := range uint32(64) {
		for x := range uint32(64) {
			share := MarchShift(x, y)
			if share < 0 || share >= 1 {
				t.Fatalf("pixel %d, %d shifted by %v, want from nought to "+
					"under one", x, y, share)
			}
			least, most = min(least, share), max(most, share)
			seen[share] = true
		}
	}
	if len(seen) < 4000 {
		t.Errorf("4096 pixels took only %d different shifts", len(seen))
	}
	if least > 0.01 || most < 0.99 {
		t.Errorf("the shifts span only %v to %v of a step", least, most)
	}
}

// A shift is a whole number over 2^24, which float32 holds as exactly as
// float64: what the Metal kernel can agree with to the last bit.
func TestAShiftIsExactInFloat32(t *testing.T) {
	for x := range uint32(50) {
		share := MarchShift(x, 7)
		if float64(float32(share)) != share {
			t.Fatalf("shift %v is not held exactly by a float32", share)
		}
	}
}

// Whatever its shift, a pixel's steps across a crossing stay inside it,
// each in its own cell.
func TestShiftedStepsStayInTheirCells(t *testing.T) {
	across := []crossing{{volume: &placedVolume{}, enter: 2, leave: 6}}
	for _, shift := range []float64{0, 0.25, 0.5, 0.999} {
		steps := stepping(across, 4)
		steps.shift = shift
		for cell := range 4 {
			stride, found := steps.nearest()
			if !found {
				t.Fatalf("shift %v: no step %d", shift, cell)
			}
			low, high := 2+float64(cell), 3+float64(cell)
			if stride.along < low || stride.along >= high {
				t.Errorf("shift %v step %d at %v, want in [%v, %v)",
					shift, cell, stride.along, low, high)
			}
		}
	}
}

// The march with its samples a share into each step is unbiased, for the
// shifts the draw uses: the shifts MarchShift gives over a grid of fine
// pixels, taken through the sum over the steps of an exponential, as the
// smoke tests march it, average to the integral, to within what so many
// samples leave: four of their own standard errors.
func TestTheShiftedMarchIsUnbiased(t *testing.T) {
	const (
		depth  = 1.4
		cuts   = 48
		side   = 512
		sigmas = 4
	)
	across := []crossing{{volume: &placedVolume{}, enter: 0, leave: depth}}
	exact := (1 - math.Exp(-2*depth)) / 2
	var sum, squares, worst float64
	for y := range uint32(side) {
		for x := range uint32(side) {
			steps := stepping(across, cuts)
			steps.shift = MarchShift(x, y)
			total := 0.0
			for range cuts {
				stride, _ := steps.nearest()
				total += math.Exp(-2*stride.along) * stride.length
			}
			sum += total
			squares += total * total
			worst = math.Max(worst, math.Abs(total-exact)/exact)
		}
	}
	count := float64(side * side)
	mean := sum / count
	spread := math.Sqrt((squares/count - mean*mean) / count)
	lean := (mean - exact) / exact
	t.Logf("lean %.3g of the integral, standard error %.3g, worst single "+
		"shift %.3g", lean, spread/exact, worst)
	if math.Abs(mean-exact) > sigmas*spread {
		t.Errorf("the mean over %d pixels' shifts is %v, the integral %v: "+
			"a lean of %.3g, more than %d standard errors (%.3g)",
			side*side, mean, exact, lean, sigmas, spread/exact)
	}
}

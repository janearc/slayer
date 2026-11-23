package colour_test

import (
	"math"
	"math/rand"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/colour"
	"github.com/janearc/slayer/ninjafilm/data"
)

// oldBloom is the bloom as it was before infinite light was handled,
// written out so that the finite path can be held to it bit for bit.
func oldBloom(lights []maths.Light, across, down int, exposure,
	radius float64) []maths.Light {
	divisor := data.Picked("bloom-spill-divisor")
	over := make([]maths.Light, len(lights))
	found := false
	for index, light := range lights {
		light = light.Scale(exposure)
		if luminance := light.Luminance(); luminance > whole {
			over[index] = light.Scale((luminance - whole) / luminance / divisor)
			found = true
		}
	}
	if !found {
		return lights
	}
	reachSigmas := data.Picked("bloom-reach-sigmas")
	reach := int(math.Ceil(reachSigmas * radius))
	weights := make([]float64, pair*reach+whole)
	var sum float64
	for index := range weights {
		offset := float64(index - reach)
		weights[index] = math.Exp(-offset * offset / (pair * radius * radius))
		if offset == none {
			weights[index] = whole
		}
		sum += weights[index]
	}
	for index := range weights {
		weights[index] /= sum
	}
	spread := func(in []maths.Light, vertical bool) []maths.Light {
		limit, stride := across, whole
		if vertical {
			limit, stride = down, across
		}
		blurred := make([]maths.Light, len(in))
		for index := range in {
			at := index % across
			if vertical {
				at = index / across
			}
			var sum maths.Light
			for step, weight := range weights {
				along := at + step - reach
				if along >= none && along < limit {
					sum = sum.Add(in[index+(along-at)*stride].Scale(weight))
				}
			}
			blurred[index] = sum
		}
		return blurred
	}
	glow := spread(spread(over, false), true)
	out := make([]maths.Light, len(lights))
	for index, light := range lights {
		out[index] = light.Add(glow[index].Scale(whole / exposure))
	}
	return out
}

// On finite light the bloom is bit for bit the one written out above,
// over seeded random frames at several exposures and radii.
func TestTheBloomOfFiniteLightIsUnchangedBitForBit(t *testing.T) {
	rng := rand.New(rand.NewSource(whole))
	for n := 0; n < 300; n++ {
		across, down := 3+rng.Intn(10), 3+rng.Intn(10)
		lights := make([]maths.Light, across*down)
		for index := range lights {
			lights[index] = maths.RGB(rng.Float64()*4, rng.Float64()*4,
				rng.Float64()*4)
		}
		exposure := []float64{0.3, 1, 1.7, 3}[n%4]
		radius := []float64{0.5, 1, 2}[n%3]
		got := colour.Bloom(lights, across, down, exposure, radius)
		want := oldBloom(lights, across, down, exposure, radius)
		for index := range got {
			g, w := got[index], want[index]
			if math.Float64bits(g.R) != math.Float64bits(w.R) ||
				math.Float64bits(g.G) != math.Float64bits(w.G) ||
				math.Float64bits(g.B) != math.Float64bits(w.B) {
				t.Fatalf("frame %d (%dx%d, exposure %v, radius %v) pixel %d: got %v, want %v",
					n, across, down, exposure, radius, index, g, w)
			}
		}
	}
}

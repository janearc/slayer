package trace

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// lampsAsked bounds sampled nonambient lamps at a point in larger scenes.
// Scenes at or below this total lamp count evaluate every lamp.
const lampsAsked = 8

// lit holds a selected lamp's index, direction, distance, unoccluded light, and
// sampling weight.
type lit struct {
	index  int
	toward maths.Vec
	far    float64
	light  maths.Light
	counts float64
}

// ambient reports whether this lamp uses the zero-direction ambient convention.
func (one lit) ambient() bool { return (reached{toward: one.toward}).ambient() }

// asked selects lamps and assigns sampling weights. Small scenes evaluate every
// lamp. Larger scenes always retain ambient lamps and make lampsAsked weighted
// selections among the remaining positive contributions.
//
// Systematic samples are spaced across cumulative worth from a deterministic
// position-derived offset. Brighter contributions are selected more often. Each
// selection is weighted by total worth divided by its own worth and the number
// of samples, accounting for repeated selections.
//
// Without occlusion, sampled worth sums to the full worth. With occlusion, only
// selected lamps trace shadows; the weighted estimate is correct on average
// over sampling offsets.
func (scene staged) asked(point, facing maths.Vec,
	worth func(toward maths.Vec, light maths.Light) float64) []lit {
	every := len(scene.lamps) <= lampsAsked
	var kept, rest []lit
	var worths []float64
	total := 0.0
	for i, lamp := range scene.lamps {
		toward, far, light := lamp.Light(point, facing)
		one := lit{index: i, toward: toward, far: far, light: light,
			counts: 1}
		if every || one.ambient() {
			kept = append(kept, one)
			continue
		}
		if each := worth(toward, light); each > 0 {
			rest, worths = append(rest, one), append(worths, each)
			total += each
		}
	}
	if every || total == 0 {
		return kept
	}
	return append(kept, picked(rest, worths, total, placeShare(point))...)
}

// picked performs lampsAsked systematic selections over cumulative worths,
// using start as the fractional offset within the first interval. Repeated
// selections combine into one lamp entry with the corresponding weight.
func picked(lamps []lit, worths []float64, total, start float64) []lit {
	step := total / lampsAsked
	next, laid, taken := start*step, 0.0, 0
	var made []lit
	for i, one := range lamps {
		laid += worths[i]
		picks := 0
		for next < laid && taken < lampsAsked {
			picks, taken = picks+1, taken+1
			next += step
		}
		if picks > 0 {
			one.counts = float64(picks) * step / worths[i]
			made = append(made, one)
		}
	}
	return made
}

// byLight uses luminance as worth for an isotropic volume sample.
func byLight(_ maths.Vec, light maths.Light) float64 {
	return light.Luminance()
}

// shadowed applies a selected lamp's sampling weight and transmission. Ambient
// contributions remain unshadowed, matching reaching.
func (scene staged) shadowed(one lit, point maths.Vec) maths.Light {
	light := one.light.Scale(one.counts)
	if light == (maths.Light{}) || one.ambient() {
		return light
	}
	return light.Times(scene.transmit(point, one.toward, one.far))
}

// placeShare hashes a point's coordinate bits to a reproducible fraction in [0,
// 1), using the SplitMix64 finalizer (Steele, Lea, and Flood, 2014).
func placeShare(point maths.Vec) float64 {
	stirred := uint64(0x9e3779b97f4a7c15)
	for _, part := range []float64{point.X, point.Y, point.Z} {
		stirred ^= math.Float64bits(part)
		stirred ^= stirred >> 30
		stirred *= 0xbf58476d1ce4e5b9
		stirred ^= stirred >> 27
		stirred *= 0x94d049bb133111eb
		stirred ^= stirred >> 31
	}
	return float64(stirred>>11) / (1 << 53)
}

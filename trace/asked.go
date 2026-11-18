package trace

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// lampsEvery is the most lamps a scene may have and still light every
// lamp exactly at every point. A street of six lamps under a sun, a
// flood and a flare is nine, and lighting each exactly keeps the fast
// light store, which keeps a grid a lamp, in step with the tracer.
const lampsEvery = 16

// lampsAsked is how many picks a point makes among its nonambient
// lamps when the scene has more than lampsEvery: past that, lighting
// every lamp at every point costs more than the sampling noise it saves.
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

// asked selects lamps and assigns sampling weights. Scenes of lampsEvery
// lamps or fewer evaluate every ambient lamp and every lamp that sends
// light to the point; one that sends none adds nothing to any sum. Larger
// scenes retain ambient lamps and pick lampsAsked among the rest by worth.
//
// Systematic samples are spaced across cumulative worth from a deterministic
// position-derived offset. Brighter contributions are selected more often. Each
// selection is weighted by total worth divided by its own worth and the number
// of samples, accounting for repeated selections.
//
// room is the caller's scratch: the result is built in it from the start,
// so a scene of lampsEvery lamps or fewer allocates nothing when room has
// that capacity. The result shares room's memory and is good until the
// caller next uses room. A room must belong to one goroutine.
//
// Without occlusion, sampled worth sums to the full worth. With occlusion, only
// selected lamps trace shadows; the weighted estimate is correct on average
// over sampling offsets.
func (scene *staged) asked(room []lit, point, facing maths.Vec,
	worth func(toward maths.Vec, light maths.Light) float64) []lit {
	return scene.askedExcept(room, point, facing, worth, -1)
}

// askedExcept is asked without one exact lamp index. The ocean mean-plane
// source path uses it to keep the transmitted sun out of legacy straight-ray
// lighting without allocating a filtered scene for each surface.
func (scene *staged) askedExcept(room []lit, point, facing maths.Vec,
	worth func(toward maths.Vec, light maths.Light) float64,
	skip int) []lit {
	every := len(scene.lamps) <= lampsEvery
	kept := room[:0]
	var rest []lit
	var worths []float64
	total := 0.0
	for i, lamp := range scene.lamps {
		if i == skip {
			continue
		}
		toward, far, light := lamp.Light(point, facing)
		one := lit{index: i, toward: toward, far: far, light: light,
			counts: 1}
		if every || one.ambient() {
			if one.ambient() || light != (maths.Light{}) {
				kept = append(kept, one)
			}
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
func (scene *staged) shadowed(one lit, point maths.Vec) maths.Light {
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

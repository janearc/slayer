package trace

import (
	"math"
	"sync"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
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

// lampsMarched is how many picks a step of a march through a medium
// makes, where asked would make lampsAsked, among the lamps that are not
// distant and not in a lamp tree: a ray through haze takes a hundred
// steps and more, and its steps' picks are summed along it, so eight a
// step cost eight shadow rays a step for little less noise. picked: one,
// as a march is usually lit.
const lampsMarched = 1

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

// marched is asked for a step of a march through a medium: lampsMarched
// picks, where asked makes lampsAsked. A scene of lampsEvery lamps or
// fewer is lit lamp by lamp as asked lights it. A scene with a lamp tree
// takes every distant lamp, a sun, at every step, picks lampsMarched among
// the other lamps outside the tree, and splits the tree's lamps into at
// most lampSplitMost nodes, a pick from each (see splitFrom).
func (scene *staged) marched(room []lit, point, facing maths.Vec,
	worth func(toward maths.Vec, light maths.Light) float64) []lit {
	return scene.askedPicks(room, point, facing, worth, -1, lampsMarched,
		lampSplitMost)
}

// askedExcept is asked without one exact lamp index. The ocean mean-plane
// source path uses it to keep the transmitted sun out of legacy straight-ray
// lighting without allocating a filtered scene for each surface.
func (scene *staged) askedExcept(room []lit, point, facing maths.Vec,
	worth func(toward maths.Vec, light maths.Light) float64,
	skip int) []lit {
	return scene.askedPicks(room, point, facing, worth, skip, lampsAsked, 0)
}

// askedPicks is askedExcept making a given number of picks where a scene
// has more lamps than lampsEvery. splits, where a scene has a lamp tree,
// is how many nodes at most its lamps are split into; nought walks the
// tree from its root as a surface does.
func (scene *staged) askedPicks(room []lit, point, facing maths.Vec,
	worth func(toward maths.Vec, light maths.Light) float64,
	skip, picks, splits int) []lit {
	if scene.tree != nil {
		return scene.askedTreed(room, point, facing, worth, skip, picks,
			splits)
	}
	every := len(scene.lamps) <= lampsEvery
	kept := room[:0]
	var rest []lit
	var worths []float64
	var scratch *askedRoom
	if !every {
		scratch = askedRooms.Get().(*askedRoom)
		rest, worths = scratch.rest[:0], scratch.worths[:0]
	}
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
	if every {
		return kept
	}
	if total > 0 {
		kept = picked(kept, rest, worths, total, placeShare(point), picks)
	}
	// picked copies what it takes, so the scratch goes back grown
	scratch.rest, scratch.worths = rest, worths
	askedRooms.Put(scratch)
	return kept
}

// askedTreed is askedPicks for a scene with a lamp tree. The lamps not
// in the tree are asked as before: every ambient one, and of the rest
// that send the point light, all of them when they are picks or fewer,
// and picks of them by worth when more. Then picks walks down the tree
// pick among the standing lamps, so a point never prices them all.
//
// With splits above nought, as at a step through a medium, a distant lamp
// is taken every time, at a count of one, and not picked among the rest by
// worth: the sun is the most of the light, and its worth beside a beam's
// is no guide to how often it is wanted.
//
// The standing lamps are then not walked to from the root but split into
// at most splits nodes, a pick from each, by splitFrom.
func (scene *staged) askedTreed(room []lit, point, facing maths.Vec,
	worth func(toward maths.Vec, light maths.Light) float64,
	skip, picks, splits int) []lit {
	kept := room[:0]
	scratch := askedRooms.Get().(*askedRoom)
	rest, worths := scratch.rest[:0], scratch.worths[:0]
	total := 0.0
	for _, i := range scene.others {
		if i == skip {
			continue
		}
		toward, far, light := scene.lamps[i].Light(point, facing)
		one := lit{index: i, toward: toward, far: far, light: light,
			counts: 1}
		if one.ambient() {
			kept = append(kept, one)
			continue
		}
		if splits > 0 && math.IsInf(far, 1) {
			if light != (maths.Light{}) {
				kept = append(kept, one)
			}
			continue
		}
		if each := worth(toward, light); each > 0 {
			rest, worths = append(rest, one), append(worths, each)
			total += each
		}
	}
	start := placeShare(point)
	if len(rest) <= picks {
		kept = append(kept, rest...)
	} else if total > 0 {
		kept = picked(kept, rest, worths, total, start, picks)
	}
	// picked copies what it takes, so the scratch goes back grown
	scratch.rest, scratch.worths = rest, worths
	askedRooms.Put(scratch)
	if splits > 0 {
		return scene.splitFrom(kept, point, facing, start, splits)
	}
	return scene.pickedFrom(kept, point, facing, start, skip, picks)
}

// askedRoom is the scratch askedExcept weighs a large scene's lamps in:
// the lamps that send light, and what each is worth.
type askedRoom struct {
	rest   []lit
	worths []float64
}

// askedRooms keeps askedRoom scratch between the points of a draw. A
// sync.Pool holds one a processor, so the worker running there takes
// back the scratch it last put, already grown to the scene's lamps, and
// a shaded point allocates none: before, a street of 250 lamps grew
// both by append at every point, 116 GiB over one CPU picture.
var askedRooms = sync.Pool{New: func() any { return new(askedRoom) }}

// picked performs asks systematic selections over cumulative worths,
// using start as the fractional offset within the first interval. Repeated
// selections combine into one lamp entry with the corresponding weight.
// The picks are appended to into, the caller's room, so that a point
// asking from a room with space for them allocates nothing.
func picked(into, lamps []lit, worths []float64, total,
	start float64, asks int) []lit {
	step := total / float64(asks)
	next, laid, taken := start*step, 0.0, 0
	made := into
	for i, one := range lamps {
		laid += worths[i]
		picks := 0
		for next < laid && taken < asks {
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
	stirred := uint64(noise.Golden)
	for _, part := range []float64{point.X, point.Y, point.Z} {
		stirred = noise.Mix(stirred ^ math.Float64bits(part))
	}
	return noise.Unit(stirred)
}

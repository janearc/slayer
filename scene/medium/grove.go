package medium

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/maths"
)

// bucket is where a canopy index sits: its place along x, then along z.
type bucket [bucketAxes]int

// the numbers of a grove, true by definition: a bucket has an index on two
// axes, and the signs of the infinities a box starts from.
const (
	bucketAxes = 2  // defined: x and z
	signPlus   = 1  // defined: the sign of positive infinity
	signMinus  = -1 // defined: the sign of negative infinity
)

// groveFadeFrom is where, as a share of a canopy's radius, a faded grove's
// mist begins to thin toward the canopy's edge. It is chosen, in
// choices.yaml.
var groveFadeFrom = choice("grove-fade-from")

// Canopy is a specimen's physical footprint, independent of mesh detail.
type Canopy struct {
	At             maths.Vec
	Radius, Height float64
}

// GroveWeather prescribes an appearance, not a water or condensation budget.
// Extinction is per metre; Depth is metres above ground; Coverage blends a
// clear field with seeded patches. Zero coverage is clear; one is filled.
//
// Intermediate coverage is an authored noise threshold, not measured area.
// Wind advects those patches in m/s.
//
// See doc/model/vegetation/redwood-atmosphere.md for the evidence and its
// limits.
type GroveWeather struct {
	Extinction, Depth, Coverage float64
	// PatchScale separates weather patch size from layer depth.
	// Zero takes Depth as the size of the patches.
	PatchScale  float64
	GroundFloor float64
	Seed        uint64
	Wind        maths.Vec
	Fade        bool
	// Asymmetry is the Henyey-Greenstein asymmetry the field scatters
	// with; nought is even every way.
	Asymmetry float64
	// Edge is how far under its top the field holds its whole density
	// before it thins to nothing at the top, as air mixed up to a lid
	// does. Nought thins it all the way from the bottom.
	Edge float64
}

// Grove holds one bounded field for a group. Its horizontal index is in
// buckets as wide as the widest canopy, so a density query looks only at the
// trees near it, however many trees there are and however deep the mist.
type Grove struct {
	weather  GroveWeather
	ground   func(x, z float64) float64
	box      maths.Box
	canopies map[bucket][]Canopy
	cell     float64
	seconds  float64
}

// NewGrove binds prescribed mist to the union of the group's canopies. Ground
// must be safe for concurrent reads; the GPU samples it when baking its grid.
// Canopy bucket coordinates must fit the host index range.
func NewGrove(weather GroveWeather, trees []Canopy,
	ground func(x, z float64) float64) (*Grove, error) {
	if ground == nil || len(trees) == weatherZero || !weather.validMist() {
		return nil, fmt.Errorf("medium: invalid grove weather or empty canopy")
	}
	grove := &Grove{weather: weather, ground: ground,
		canopies: map[bucket][]Canopy{}, box: maths.Box{
			Min: maths.V(math.Inf(signPlus), math.Inf(signPlus),
				math.Inf(signPlus)),
			Max: maths.V(math.Inf(signMinus), math.Inf(signMinus),
				math.Inf(signMinus))}}
	for _, tree := range trees {
		if !weatherFinite(tree.At.Len()) || !weatherFinite(tree.Radius) ||
			tree.Radius <= weatherZero || !weatherFinite(tree.Height) ||
			tree.Height <= weatherZero || weather.GroundFloor > tree.At.Y {
			return nil, fmt.Errorf("medium: invalid grove canopy")
		}
		grove.cell = math.Max(grove.cell, tree.Radius)
	}
	for _, tree := range trees {
		lo := tree.At.Sub(maths.V(tree.Radius, nought, tree.Radius))
		hi := tree.At.Add(maths.V(tree.Radius, tree.Height, tree.Radius))
		if !grove.indexable(lo.X) || !grove.indexable(hi.X) ||
			!grove.indexable(lo.Z) || !grove.indexable(hi.Z) {
			return nil, fmt.Errorf("medium: grove canopy exceeds the bucket index range")
		}
		grove.box.Min = maths.V(math.Min(grove.box.Min.X, lo.X),
			math.Min(grove.box.Min.Y, weather.GroundFloor), math.Min(grove.box.Min.Z, lo.Z))
		grove.box.Max = maths.V(math.Max(grove.box.Max.X, hi.X),
			math.Max(grove.box.Max.Y, hi.Y), math.Max(grove.box.Max.Z, hi.Z))
		for x := grove.index(lo.X); x <= grove.index(hi.X); x++ {
			for z := grove.index(lo.Z); z <= grove.index(hi.Z); z++ {
				key := bucket{x, z}
				grove.canopies[key] = append(grove.canopies[key], tree)
			}
		}
	}
	return grove, nil
}

// indexable prevents float-to-int saturation and loop-counter overflow.
// float64(MaxInt) rounds upward on 64-bit hosts, so the upper bound is strict.
func (grove *Grove) indexable(value float64) bool {
	index := math.Floor(value / grove.cell)
	return index >= float64(math.MinInt) && index < float64(math.MaxInt)
}

// index assigns a horizontal coordinate to a canopy bucket.
func (grove *Grove) index(value float64) int {
	return int(math.Floor(value / grove.cell))
}

// Bounds encloses the prescribed ground floor and every canopy.
func (grove *Grove) Bounds() maths.Box { return grove.box }

// Asymmetry is the Henyey-Greenstein asymmetry the field scatters with:
// nought scatters evenly every way.
func (grove *Grove) Asymmetry() float64 { return grove.weather.Asymmetry }

// Finest requests enough grid cells to retain the prescribed layer. The GPU's
// memory cap still applies, so a large, thin grove is approximate.
func (grove *Grove) Finest() float64 {
	return grove.weather.Depth / mistTuning.samples
}

// Tick shares the immutable footprint and advances only the weather phase.
func (grove *Grove) Tick(at time.Duration) Medium {
	ticked := *grove
	ticked.seconds = at.Seconds()
	return &ticked
}

// nearby is the canopies whose footprints reach the bucket that holds at.
func (grove *Grove) nearby(at maths.Vec) []Canopy {
	return grove.canopies[bucket{grove.index(at.X), grove.index(at.Z)}]
}

// Density fades above the terrain, within the union of canopy footprints.
func (grove *Grove) Density(at maths.Vec) float64 {
	box := grove.box
	if at.X < box.Min.X || at.X > box.Max.X || at.Y < box.Min.Y ||
		at.Y > box.Max.Y || at.Z < box.Min.Z || at.Z > box.Max.Z {
		return nought
	}
	below := at.Y - grove.ground(at.X, at.Z)
	if below <= nought || below >= grove.weather.Depth {
		return nought
	}
	var weight float64
	for _, tree := range grove.nearby(at) {
		distance := math.Hypot(at.X-tree.At.X, at.Z-tree.At.Z)
		if distance >= tree.Radius || at.Y > tree.At.Y+tree.Height {
			continue
		}
		fade := float64(unity)
		if grove.weather.Fade {
			fade = unity - maths.Smooth(groveFadeFrom, unity,
				distance/tree.Radius)
		}
		weight = math.Max(weight, fade)
	}
	return grove.weather.mistDensity(at, below, grove.seconds, weight)
}

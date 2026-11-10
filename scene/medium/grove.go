package medium

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/maths"
)

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
// See doc/redwood-atmosphere.md for the evidence and its limits.
type GroveWeather struct {
	Extinction, Depth, Coverage float64
	GroundFloor                 float64
	Seed                        uint64
	Wind                        maths.Vec
	Fade                        bool
}

// Grove holds one bounded field for a group. Its small horizontal index keeps
// density queries independent of the total number of trees in the scene.
type Grove struct {
	weather  GroveWeather
	ground   func(x, z float64) float64
	box      maths.Box
	canopies map[[2]int][]Canopy
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
	grove := &Grove{weather: weather, ground: ground, cell: weather.Depth,
		canopies: map[[2]int][]Canopy{}, box: maths.Box{
			Min: maths.V(math.Inf(1), math.Inf(1), math.Inf(1)),
			Max: maths.V(math.Inf(-1), math.Inf(-1), math.Inf(-1))}}
	for _, tree := range trees {
		if !weatherFinite(tree.At.Len()) || !weatherFinite(tree.Radius) ||
			tree.Radius <= weatherZero || !weatherFinite(tree.Height) ||
			tree.Height <= weatherZero || weather.GroundFloor > tree.At.Y {
			return nil, fmt.Errorf("medium: invalid grove canopy")
		}
		grove.cell = math.Max(grove.cell, tree.Radius)
	}
	for _, tree := range trees {
		lo := tree.At.Sub(maths.V(tree.Radius, 0, tree.Radius))
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
				key := [2]int{x, z}
				grove.canopies[key] = append(grove.canopies[key], tree)
			}
		}
	}
	return grove, nil
}

// indexable prevents float-to-int saturation and loop-counter overflow.
// float64(MaxInt) rounds upward on 64-bit hosts, so the upper bound is strict.
func (grove *Grove) indexable(v float64) bool {
	index := math.Floor(v / grove.cell)
	return index >= float64(math.MinInt) && index < float64(math.MaxInt)
}

// index assigns a horizontal coordinate to a canopy bucket.
func (grove *Grove) index(v float64) int { return int(math.Floor(v / grove.cell)) }

// Bounds encloses the prescribed ground floor and every canopy.
func (grove *Grove) Bounds() maths.Box { return grove.box }

// Finest requests enough grid cells to retain the prescribed layer. The GPU's
// existing memory cap still applies, so a large, thin grove is approximate.
func (grove *Grove) Finest() float64 {
	return grove.weather.Depth / mistTuning.samples
}

// Tick shares the immutable footprint and advances only the weather phase.
func (grove *Grove) Tick(at time.Duration) Medium {
	copy := *grove
	copy.seconds = at.Seconds()
	return &copy
}

// Density fades above the terrain, within the union of canopy footprints.
func (grove *Grove) Density(at maths.Vec) float64 {
	box := grove.box
	if at.X < box.Min.X || at.X > box.Max.X || at.Y < box.Min.Y ||
		at.Y > box.Max.Y || at.Z < box.Min.Z || at.Z > box.Max.Z {
		return 0
	}
	below := at.Y - grove.ground(at.X, at.Z)
	if below <= 0 || below >= grove.weather.Depth {
		return 0
	}
	weight := 0.0
	for _, tree := range grove.canopies[[2]int{grove.index(at.X), grove.index(at.Z)}] {
		distance := math.Hypot(at.X-tree.At.X, at.Z-tree.At.Z)
		if distance >= tree.Radius || at.Y > tree.At.Y+tree.Height {
			continue
		}
		fade := 1.0
		if grove.weather.Fade {
			fade = 1 - maths.Smooth(0.7, 1, distance/tree.Radius)
		}
		weight = math.Max(weight, fade)
	}
	return grove.weather.mistDensity(at, below, grove.seconds, weight)
}

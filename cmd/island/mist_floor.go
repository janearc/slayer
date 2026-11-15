package main

import (
	"fmt"
	"time"

	"github.com/janearc/slayer/maths"
)

// floorKey describes a source-grounded grid in the session's fixed world.
// Camera translation, weather phase and density parameters do not change it.
type floorKey struct {
	space      maths.Subspace
	bounds     localBounds
	side       int
	tolerance  float64
	iterations int
}

type floorEntry struct {
	grid floorGrid
	used uint64
}

// floorStore retains immutable height payloads under an explicit byte limit.
// Map metadata and grids still referenced by the current frame are separate.
type floorStore struct {
	values       map[floorKey]floorEntry
	limit, bytes int64
	clock        uint64
}

// mistFloor shares only the fixed source projection. Every weather layer still
// builds its own time-dependent density field. A zero allowance disables reuse.
func (site *siteBuilder) mistFloor(mist mistPlan) (floorGrid, error) {
	started := time.Now()
	defer func() { site.record.MistFloorTime += time.Since(started) }()
	if err := site.ctx.Err(); err != nil {
		return floorGrid{}, err
	}
	if mist.FloorSide < 2 || mist.FloorSide > 129 {
		return floorGrid{}, fmt.Errorf("island: invalid mist floor sampling side")
	}
	store := site.floors
	if store == nil {
		return floorGrid{}, fmt.Errorf("island: mist floor store is unavailable")
	}
	key := floorKey{site.space, mist.Bounds, mist.FloorSide, mist.FloorTolerance, mist.FloorIterations}
	// Each height is an eight-byte float64 payload; dimensions are checked above.
	bytes := int64(mist.FloorSide) * int64(mist.FloorSide) * 8
	if store.limit > 0 {
		if bytes > store.limit {
			return floorGrid{}, fmt.Errorf("island: one mist floor exceeds its cache payload budget")
		}
		if entry, present := store.values[key]; present {
			store.clock++
			entry.used = store.clock
			store.values[key] = entry
			site.record.MistFloorHits++
			return entry.grid, nil
		}
	}
	floor := floorGrid{bounds: mist.Bounds, side: mist.FloorSide, heights: make([]float64, mist.FloorSide*mist.FloorSide)}
	for row := 0; row < floor.side; row++ {
		for column := 0; column < floor.side; column++ {
			x := floor.bounds.MinEast + float64(column)/float64(floor.side-1)*(floor.bounds.MaxEast-floor.bounds.MinEast)
			z := floor.bounds.MinSouth + float64(row)/float64(floor.side-1)*(floor.bounds.MaxSouth-floor.bounds.MinSouth)
			height, err := site.floorHeight(x, z, mist.FloorTolerance, mist.FloorIterations)
			if err != nil {
				return floorGrid{}, err
			}
			floor.heights[row*floor.side+column] = height
		}
	}
	site.record.MistFloorBuilds++
	if store.limit > 0 {
		for bytes > store.limit-store.bytes {
			var oldest floorKey
			first := true
			for candidate, entry := range store.values {
				if first || entry.used < store.values[oldest].used {
					oldest, first = candidate, false
				}
			}
			if first {
				return floorGrid{}, fmt.Errorf("island: mist floor cache accounting is inconsistent")
			}
			store.bytes -= int64(len(store.values[oldest].grid.heights)) * 8
			delete(store.values, oldest)
			site.record.MistFloorEvictions++
		}
		if store.values == nil {
			store.values = make(map[floorKey]floorEntry)
		}
		store.clock++
		store.values[key] = floorEntry{grid: floor, used: store.clock}
		store.bytes += bytes
	}
	return floor, nil
}

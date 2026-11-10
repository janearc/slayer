package terrain

import (
	"container/list"
	"fmt"
	"strconv"
)

// PatchCacheBudget bounds retained sampled payloads and bookkeeping entries.
// It excludes the raster tile cache, meshes and copies retained by consumers.
type PatchCacheBudget struct {
	Bytes   int64
	Entries int
}

// PatchCache retains immutable native sample grids for a borrowed open raster.
// Access is serial, like Raster. Source bytes must stay immutable for its life.
// A new source/content version needs a new cache; camera position is not a key.
type PatchCache struct {
	raster       *Raster
	budget       PatchCacheBudget
	used         int64
	hits, misses uint64
	items        map[patchSampleKey]*list.Element
	recent       list.List
}

// patchSampleKey canonicalizes an oversized side request to its actual axes.
type patchSampleKey struct {
	patch        Patch
	across, down int
}

type cachedPatch struct {
	key   patchSampleKey
	grid  PatchGrid
	bytes int64
}

// NewPatchCache does not decode or reserve storage proportional to world area.
// Either zero budget disables retention but still permits bounded direct reads.
func NewPatchCache(raster *Raster, budget PatchCacheBudget) (*PatchCache, error) {
	if raster == nil || raster.closed || !raster.placed || budget.Bytes < 0 ||
		budget.Bytes > addressableBytes || budget.Entries < 0 {
		return nil, fmt.Errorf("terrain: invalid patch cache source or budget")
	}
	return &PatchCache{raster: raster, budget: budget, items: map[patchSampleKey]*list.Element{}}, nil
}

// Read returns a grid with read-only accessors. Cache hits obey the current
// readBytes allowance too, rather than borrowing an earlier larger request.
func (cache *PatchCache) Read(patch Patch, side int, readBytes int64) (PatchGrid, error) {
	if cache == nil || cache.items == nil {
		return PatchGrid{}, fmt.Errorf("terrain: unchecked patch cache")
	}
	across, down, count, err := cache.raster.patchShape(patch, side, readBytes)
	if err != nil {
		if cache.raster == nil || cache.raster.closed {
			cache.Clear()
		}
		return PatchGrid{}, err
	}
	key := patchSampleKey{patch: patch, across: across, down: down}
	if entry, found := cache.items[key]; found {
		cache.hits++
		cache.recent.MoveToFront(entry)
		return entry.Value.(cachedPatch).grid, nil
	}
	cache.misses++
	const indexBytes = strconv.IntSize / byteBits
	bytes := int64(across)*indexBytes + int64(down)*indexBytes +
		int64(count)*(gridHeightBytes+gridAvailabilityBytes)
	retain := cache.budget.Entries > 0 && bytes <= cache.budget.Bytes
	if retain {
		cache.makeRoom(bytes)
	}
	grid, err := cache.raster.ReadPatch(patch, side, readBytes)
	if err != nil {
		return PatchGrid{}, err
	}
	if retain {
		cache.items[key] = cache.recent.PushFront(cachedPatch{key: key, grid: grid, bytes: bytes})
		cache.used += bytes
	}
	return grid, nil
}

// makeRoom evicts before allocating a retained replacement, bounding overlap.
// Consumers may still hold evicted grids; this controls the cache's own roots.
func (cache *PatchCache) makeRoom(bytes int64) {
	for cache.used > cache.budget.Bytes-bytes || len(cache.items) >= cache.budget.Entries {
		entry := cache.recent.Back()
		if entry == nil {
			return
		}
		old := entry.Value.(cachedPatch)
		delete(cache.items, old.key)
		cache.used -= old.bytes
		cache.recent.Remove(entry)
	}
}

// Usage exposes payload retention and actual successful lookup hits/misses.
// A miss may subsequently fail to decode; validation refusals count neither.
func (cache *PatchCache) Usage() (bytes int64, entries int, hits, misses uint64) {
	return cache.used, len(cache.items), cache.hits, cache.misses
}

// Clear releases only cache-held grids. It neither closes the borrowed raster
// nor invalidates read-only grids still held by a caller. Counters persist.
func (cache *PatchCache) Clear() {
	cache.items = map[patchSampleKey]*list.Element{}
	cache.recent.Init()
	cache.used = 0
}

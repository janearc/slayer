package terrain

import "container/list"

// tileCache evicts the least recently used decoded payload by bytes.
type tileCache struct {
	limit, used int64
	entries     int
	items       map[int]*list.Element
	recent      list.List
}

type cachedTile struct {
	index   int
	samples []float32
	bytes   int64
}

// newTileCache needs no allocation proportional to the raster's world extent.
func newTileCache(limit int64, entries int) tileCache {
	return tileCache{limit: limit, entries: entries, items: map[int]*list.Element{}}
}

// get makes a reused tile the most recent entry.
func (cache *tileCache) get(index int) ([]float32, bool) {
	element, found := cache.items[index]
	if !found {
		return nil, false
	}
	cache.recent.MoveToFront(element)
	entry := element.Value.(cachedTile)
	return entry.samples, true
}

// put never retains a tile larger than the cache's entire allowance.
func (cache *tileCache) put(index int, samples []float32) {
	bytes := int64(len(samples)) * sampleBytes
	if bytes > cache.limit || cache.entries == 0 {
		return
	}
	if old, found := cache.items[index]; found {
		cache.remove(old)
	}
	for cache.used > cache.limit-bytes || len(cache.items) >= cache.entries {
		oldest := cache.recent.Back()
		if oldest == nil {
			break
		}
		cache.remove(oldest)
	}
	entry := cachedTile{index: index, samples: samples, bytes: bytes}
	cache.items[index] = cache.recent.PushFront(entry)
	cache.used += bytes
}

// remove releases the payload without clearing unrelated useful tiles.
func (cache *tileCache) remove(element *list.Element) {
	entry := element.Value.(cachedTile)
	delete(cache.items, entry.index)
	cache.used -= entry.bytes
	cache.recent.Remove(element)
}

// clear releases retained sample payloads while keeping the raster reusable.
func (cache *tileCache) clear() {
	cache.items = map[int]*list.Element{}
	cache.recent.Init()
	cache.used = 0
}

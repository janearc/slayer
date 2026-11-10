package terrain

import (
	_ "embed"
	"fmt"
	"math"
	"sync"

	"github.com/janearc/puzzlemouth/numbers"
)

// RasterBudget bounds payloads held by one raster. It is not an RSS budget.
// A world with several rasters must divide its aggregate budget between them.
type RasterBudget struct {
	CacheBytes    int64
	CacheTiles    int
	TileBytes     int64
	MetadataBytes int64
}

// addressableBytes is defined by the largest positive native int.
const addressableBytes = int64(^uint(0) >> 1)

// sampleBytes and byteBits are defined by TIFF's float32 encoding.
const byteBits = 8
const sampleBytes = floatBits / byteBits

// directoryCountBytes is defined by BigTIFF's uint64 directory-entry count.
const directoryCountBytes = 64 / byteBits

//go:embed memory-policy.yaml
var memoryPolicyRaw []byte

var memoryPolicy = sync.OnceValues(readMemoryPolicy)

// readMemoryPolicy keeps default resource choices in the Puzzlemouth book.
func readMemoryPolicy() (numbers.Numbers, error) {
	return numbers.Read("memory-policy.yaml", memoryPolicyRaw)
}

// DefaultRasterBudget returns the authored default limits, without opening
// data.
func DefaultRasterBudget() (RasterBudget, error) {
	book, err := memoryPolicy()
	if err != nil {
		return RasterBudget{}, err
	}
	var budget RasterBudget
	for name, target := range map[string]*int64{
		"retained-bytes": &budget.CacheBytes,
		"tile-bytes":     &budget.TileBytes,
		"metadata-bytes": &budget.MetadataBytes,
	} {
		value, err := policyCount(book, name)
		if err != nil {
			return RasterBudget{}, err
		}
		*target = value
	}
	tiles, err := policyCount(book, "retained-tiles")
	if err != nil {
		return RasterBudget{}, err
	}
	budget.CacheTiles = int(tiles)
	return budget, budget.valid()

}

// DefaultWindowBytes returns the picked strict-window payload limit.
func DefaultWindowBytes() (int64, error) {
	book, err := memoryPolicy()
	if err != nil {
		return 0, err
	}
	return policyCount(book, "window-bytes")
}

// policyCount requires an exact, addressable nonnegative resource count.
func policyCount(book numbers.Numbers, name string) (int64, error) {
	value, err := book.Get(name)
	if err != nil {
		return 0, err
	}
	if value.Value < 0 || value.Value >= float64(addressableBytes) ||
		math.Trunc(value.Value) != value.Value {
		return 0, fmt.Errorf("terrain: invalid resource policy %s", name)
	}
	return int64(value.Value), nil
}

// valid checks allocation bounds before a file or tile is read.
func (budget RasterBudget) valid() error {
	if budget.CacheBytes < 0 || budget.CacheTiles < 0 || budget.TileBytes < sampleBytes ||
		budget.MetadataBytes < directoryCountBytes || budget.TileBytes > addressableBytes ||
		budget.MetadataBytes > addressableBytes {
		return fmt.Errorf("terrain: invalid raster byte budget")
	}
	return nil
}

// allocation checks multiplication before converting a file count to an int.
func allocation(count, width uint64, limit int64) (int, error) {
	if width == 0 || limit < 0 || count > uint64(limit)/width {
		return 0, fmt.Errorf("terrain: payload exceeds its %d-unit capacity", limit)
	}
	return int(count * width), nil
}

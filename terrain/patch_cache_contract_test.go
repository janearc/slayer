package terrain

import (
	"strconv"
	"strings"
	"testing"
)

type patchRectangle struct {
	MinColumn int `yaml:"min-column"`
	MaxColumn int `yaml:"max-column"`
	MinRow    int `yaml:"min-row"`
	MaxRow    int `yaml:"max-row"`
	Level     int
}

func (rect patchRectangle) patch() Patch {
	return Patch{MinColumn: rect.MinColumn, MaxColumn: rect.MaxColumn,
		MinRow: rect.MinRow, MaxRow: rect.MaxRow, Level: rect.Level}
}

type patchBook struct {
	Source       string
	IndexBytes   int `yaml:"index-bytes"`
	HeightBytes  int `yaml:"height-bytes"`
	MaskBytes    int `yaml:"availability-bytes"`
	Patches      map[string]patchRectangle
	Side         int
	PerGridBytes int64 `yaml:"per-grid-bytes"`
	CacheBytes   int64 `yaml:"cache-bytes"`
	CacheEntries int   `yaml:"cache-entries"`
	Access       []string
	After        struct {
		Bytes        int64
		Entries      int
		Hits, Misses uint64
		Evicted      string
	} `yaml:"after-access"`
	Stricter struct {
		Patch    string
		Refused  int64 `yaml:"refused-read-bytes"`
		Accepted int64 `yaml:"accepted-read-bytes"`
	} `yaml:"stricter-hit"`
	Alias struct {
		Patch string
		Sides []int
	}
	AfterClose struct {
		Reads string
		Bytes int64 `yaml:"retained-payload-after-refused-read"`
	} `yaml:"after-close"`
}

func checkedPatchBook(t *testing.T) patchBook {
	t.Helper()
	book := terrainContract[patchBook](t, "patch-cache")
	if len(book.Patches) != 3 || len(book.Access) != 4 ||
		len(book.Alias.Sides) != 2 {
		t.Fatal("the three-patch reuse fixture is incomplete")
	}
	if strconv.IntSize != book.IndexBytes*8 {
		t.Fatal("the authored payload book requires its stated native index width")
	}
	// Two native axes hold four indices; the grid holds four samples.
	payload := int64(4*book.IndexBytes + 4*(book.HeightBytes+book.MaskBytes))
	if book.HeightBytes != 8 || book.MaskBytes != 1 ||
		book.PerGridBytes != payload {
		t.Fatal("the authored payload arithmetic is inconsistent", payload)
	}
	if book.Source != "fixture-plain.tif" || book.AfterClose.Reads != "refused" {
		t.Fatal("the authored source or close contract changed")
	}
	return book
}

func checkedPatchCache(t *testing.T,
	budget PatchCacheBudget,
) (*PatchCache, *Raster) {
	t.Helper()
	streaming := terrainContract[streamingBook](t, "streaming")
	raster, _ := contractRaster(t, streamingBudget(streaming))
	cache, err := NewPatchCache(raster, budget)
	if err != nil {
		t.Fatal(err)
	}
	return cache, raster
}

func expectPatchUsage(t *testing.T, cache *PatchCache,
	wantBytes int64, wantEntries int, wantHits, wantMisses uint64,
) {
	t.Helper()
	bytes, entries, hits, misses := cache.Usage()
	if bytes != wantBytes || entries != wantEntries ||
		hits != wantHits || misses != wantMisses {
		t.Fatalf("cache %d bytes/%d entries/%d hits/%d misses; want %d/%d/%d/%d",
			bytes, entries, hits, misses, wantBytes, wantEntries, wantHits, wantMisses)
	}
}

func readContractPatch(t *testing.T, cache *PatchCache,
	book patchBook, name string, side int,
) PatchGrid {
	t.Helper()
	rect, known := book.Patches[name]
	if !known {
		t.Fatal("unknown authored patch", name)
	}
	grid, err := cache.Read(rect.patch(), side, book.PerGridBytes)
	if err != nil {
		t.Fatal(name, err)
	}
	across, down := grid.Size()
	if across != 2 || down != 2 || grid.Patch() != rect.patch() {
		t.Fatal("native terminal rectangle changed", across, down, grid.Patch())
	}
	return grid
}

func TestPatchCacheLRUAndCanonicalSides(t *testing.T) {
	book := checkedPatchBook(t)
	// Each cap must work without the other forcing the same eviction.
	for name, budget := range map[string]PatchCacheBudget{
		"byte cap":  {book.CacheBytes, book.CacheEntries + 1},
		"entry cap": {book.CacheBytes + book.PerGridBytes, book.CacheEntries},
	} {
		t.Run(name, func(t *testing.T) { exercisePatchReuse(t, book, budget) })
	}
}

func exercisePatchReuse(t *testing.T, book patchBook, budget PatchCacheBudget) {
	t.Helper()
	cache, _ := checkedPatchCache(t, budget)
	expectPatchUsage(t, cache, 0, 0, 0, 0)
	for _, name := range book.Access {
		readContractPatch(t, cache, book, name, book.Side)
	}
	expectPatchUsage(t, cache, book.After.Bytes, book.After.Entries,
		book.After.Hits, book.After.Misses)
	// A cache hit must obey this caller's smaller output allowance;
	// refusal is not a lookup hit and must not disturb retained data.
	patch := book.Patches[book.Stricter.Patch].patch()
	_, err := cache.Read(patch, book.Side, book.Stricter.Refused)
	if err == nil || !strings.Contains(err.Error(), "byte budget") {
		t.Fatal("a cached grid bypassed the stricter read allowance", err)
	}
	expectPatchUsage(t, cache, book.After.Bytes, book.After.Entries, 1, 3)
	if _, err := cache.Read(patch, book.Side, book.Stricter.Accepted); err != nil {
		t.Fatal("the exact authored grid allowance was refused", err)
	}
	expectPatchUsage(t, cache, book.After.Bytes, book.After.Entries, 2, 3)
	for _, side := range book.Alias.Sides {
		readContractPatch(t, cache, book, book.Alias.Patch, side)
	}
	expectPatchUsage(t, cache, book.After.Bytes, book.After.Entries, 4, 3)
	// B was older than the reused A when C arrived. Asking B again
	// must decode a miss, even though the raster's tile remains cached.
	readContractPatch(t, cache, book, book.After.Evicted, book.Side)
	expectPatchUsage(t, cache, book.After.Bytes, book.After.Entries, 4, 4)
}

func expectHeldPatchA(t *testing.T, grid PatchGrid) {
	t.Helper()
	// The TIFF writer's two-by-two corner is independently known:
	// heights (column+1000*row)/8, cells two metres apart.
	want := [4][3]float64{{1001.5, 8999.25, 0},
		{1003.5, 8999.25, 0.125}, {1001.5, 8997.25, 125},
		{1003.5, 8997.25, 125.125}}
	for i, cell := range want {
		east, north, height, present := grid.At(i%2, i/2)
		if !present || [3]float64{east, north, height} != cell {
			t.Fatalf("held cell %d: %g/%g/%g/%v, want %v",
				i, east, north, height, present, cell)
		}
	}
}

func TestPatchCacheClearAndClosedSource(t *testing.T) {
	book := checkedPatchBook(t)
	budget := PatchCacheBudget{book.CacheBytes, book.CacheEntries}
	cache, raster := checkedPatchCache(t, budget)
	held := readContractPatch(t, cache, book, "a", book.Side)
	expectHeldPatchA(t, held)
	cache.Clear()
	expectPatchUsage(t, cache, 0, 0, 0, 1)
	expectHeldPatchA(t, held)
	readContractPatch(t, cache, book, "a", book.Side)
	expectPatchUsage(t, cache, book.PerGridBytes, 1, 0, 2)
	if err := raster.Close(); err != nil {
		t.Fatal(err)
	}
	_, err := cache.Read(book.Patches["a"].patch(), book.Side,
		book.PerGridBytes)
	if err == nil || !strings.Contains(err.Error(), "closed source") {
		t.Fatal("a hit escaped the closed-source refusal", err)
	}
	expectPatchUsage(t, cache, book.AfterClose.Bytes, 0, 0, 2)
	expectHeldPatchA(t, held)
}

func TestPatchCacheDisabledRetentionStillReads(t *testing.T) {
	book := checkedPatchBook(t)
	for name, budget := range map[string]PatchCacheBudget{
		"zero bytes":     {0, book.CacheEntries},
		"zero entries":   {book.CacheBytes, 0},
		"one byte short": {book.PerGridBytes - 1, book.CacheEntries},
	} {
		t.Run(name, func(t *testing.T) {
			cache, _ := checkedPatchCache(t, budget)
			for range 2 {
				grid := readContractPatch(t, cache, book, "a", book.Side)
				expectHeldPatchA(t, grid)
			}
			expectPatchUsage(t, cache, 0, 0, 0, 2)
		})
	}
}

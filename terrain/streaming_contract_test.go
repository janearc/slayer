package terrain

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
)

// The book contains independently authored resource counts and source cells.
// These witnesses use the small TIFF writer, never an external terrain file.
type streamingBook struct {
	Budget struct {
		CacheBytes    int64 `yaml:"cache-bytes"`
		CacheTiles    int   `yaml:"cache-tiles"`
		TileBytes     int64 `yaml:"tile-bytes"`
		MetadataBytes int64 `yaml:"metadata-bytes"`
	}
	Sequence       []sourceCell
	RetainedBytes  int64      `yaml:"retained-bytes"`
	RetainedTiles  int        `yaml:"retained-tiles"`
	Cached         sourceCell `yaml:"cached-after-truncate"`
	Evicted        sourceCell `yaml:"evicted-after-truncate"`
	Coverage       Coverage
	Outside        []sourcePlace
	OversizedLimit int64 `yaml:"oversize-tile-bytes"`
}

type sourceCell struct{ Column, Row int }
type sourcePlace struct{ East, North float64 }

func terrainContract[T any](t *testing.T, name string) T {
	t.Helper()
	path := "testdata/" + name + "-contract.yaml"
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var book T
	if err := yamlfile.Read(path, raw, &book); err != nil {
		t.Fatal(err)
	}
	return book
}

func streamingBudget(book streamingBook) RasterBudget {
	return RasterBudget{CacheBytes: book.Budget.CacheBytes,
		CacheTiles: book.Budget.CacheTiles, TileBytes: book.Budget.TileBytes,
		MetadataBytes: book.Budget.MetadataBytes}
}

func contractRaster(t *testing.T, budget RasterBudget) (*Raster, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "plain.tif")
	writeTiff(t, path)
	raster, err := OpenWithBudget(path, budget)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { raster.Close() })
	return raster, path
}

// The fixture places two-metre cells at (1000.5,9000.25). Their centres
// and heights come from that authored layout, not Raster's own answers.
func sourceSample(raster *Raster, cell sourceCell) (float64, bool, error) {
	return raster.Elevation(1001.5+2*float64(cell.Column),
		8999.25-2*float64(cell.Row))
}

func expectSourceCell(t *testing.T, raster *Raster, cell sourceCell) {
	t.Helper()
	got, present, err := sourceSample(raster, cell)
	want := float64(cell.Column+1000*cell.Row) / 8
	if err != nil || !present || got != want {
		t.Fatalf("cell %v: %g, present %v, %v; want %g",
			cell, got, present, err, want)
	}
}

func TestStreamingTileLRURetainsReusedSourceAfterBackingFileLoss(t *testing.T) {
	book := terrainContract[streamingBook](t, "streaming")
	if len(book.Sequence) != 4 {
		t.Fatal("the authored A B A C sequence is incomplete")
	}
	raster, path := contractRaster(t, streamingBudget(book))
	if bytes, tiles := raster.CacheUsage(); bytes != 0 || tiles != 0 {
		t.Fatal("opening a source decoded tiles", bytes, tiles)
	}
	wantTiles := []int{1, 2, 2, 2}
	for i, cell := range book.Sequence {
		expectSourceCell(t, raster, cell)
		bytes, tiles := raster.CacheUsage()
		if tiles != wantTiles[i] || bytes != int64(wantTiles[i])*4096 {
			t.Fatalf("access %d retained %d bytes/%d tiles", i, bytes, tiles)
		}
	}
	bytes, tiles := raster.CacheUsage()
	if bytes != book.RetainedBytes || tiles != book.RetainedTiles {
		t.Fatal("retention disagrees with the resource book", bytes, tiles)
	}
	// Destroy only this test's temporary source after metadata and tiles
	// were read. A cached value can succeed; an evicted value needs I/O.
	if err := os.Truncate(path, 0); err != nil {
		t.Fatal(err)
	}
	expectSourceCell(t, raster, book.Cached)
	_, present, err := sourceSample(raster, book.Evicted)
	if err == nil || present {
		t.Fatal("the evicted source cell survived backing-file loss", present, err)
	}
	expectSourceCell(t, raster, book.Cached)
	raster.DiscardCache()
	if bytes, tiles := raster.CacheUsage(); bytes != 0 || tiles != 0 {
		t.Fatal("discard retained decoded tiles", bytes, tiles)
	}
	_, present, err = sourceSample(raster, book.Cached)
	if err == nil || present {
		t.Fatal("discarded cell did not need its missing source", present, err)
	}
}

func TestStreamingCoverageAndMalformedCoordinates(t *testing.T) {
	book := terrainContract[streamingBook](t, "streaming")
	raster, _ := contractRaster(t, streamingBudget(book))
	if got, placed := raster.Coverage(); !placed || got != book.Coverage {
		t.Fatal("source cell extent", got, placed, "want", book.Coverage)
	}
	// The north-west corner is included. Checking only the excluded
	// east and south edges would miss an exclusive west/north change.
	corner, included, err := raster.Elevation(book.Coverage.West,
		book.Coverage.North)
	if err != nil || !included || corner != 0 {
		t.Fatal("the included source corner was lost", corner, included, err)
	}
	if len(book.Outside) != 4 {
		t.Fatal("all four authored source edges must be checked")
	}
	for _, place := range book.Outside {
		_, present, err := raster.Elevation(place.East, place.North)
		if err != nil || present {
			t.Errorf("outside %v: present %v, %v", place, present, err)
		}
	}
	for _, place := range []sourcePlace{{math.NaN(), 9000},
		{1001, math.Inf(1)}, {math.Inf(-1), 9000}} {
		_, present, err := raster.Elevation(place.East, place.North)
		if err == nil || present {
			t.Errorf("malformed %v: present %v, %v", place, present, err)
		}
	}
	// The written last cell of row zero is no-data; a zero height at
	// the first cell is a real sample and must not share its mask.
	height, present, err := sourceSample(raster, sourceCell{})
	if err != nil || !present || height != 0 {
		t.Fatal("zero elevation became absence", height, present, err)
	}
	_, present, err = sourceSample(raster, sourceCell{99, 0})
	if err != nil || present {
		t.Fatal("no-data became measured ground", present, err)
	}
}

func TestStreamingTileLimitRefusesBeforeRetention(t *testing.T) {
	book := terrainContract[streamingBook](t, "streaming")
	budget := streamingBudget(book)
	budget.TileBytes = book.OversizedLimit
	raster, _ := contractRaster(t, budget)
	_, present, err := sourceSample(raster, sourceCell{})
	if err == nil || present || !strings.Contains(err.Error(), "capacity") {
		t.Fatal("oversized tile was decoded", present, err)
	}
	if bytes, tiles := raster.CacheUsage(); bytes != 0 || tiles != 0 {
		t.Fatal("a refused tile retained payload", bytes, tiles)
	}
	// The exact 4096-byte counterpart is exercised by every successful
	// read in the LRU witness, rather than calling the capacity helper.
}

func TestStreamingZeroRetentionNeedsTheSource(t *testing.T) {
	book := terrainContract[streamingBook](t, "streaming")
	for _, name := range []string{"bytes", "entries"} {
		t.Run(name, func(t *testing.T) {
			budget := streamingBudget(book)
			if name == "bytes" {
				budget.CacheBytes = 0
			} else {
				budget.CacheTiles = 0
			}
			raster, path := contractRaster(t, budget)
			expectSourceCell(t, raster, book.Cached)
			if bytes, tiles := raster.CacheUsage(); bytes != 0 || tiles != 0 {
				t.Fatal("disabled retention held a tile", bytes, tiles)
			}
			if err := os.Truncate(path, 0); err != nil {
				t.Fatal(err)
			}
			_, present, err := sourceSample(raster, book.Cached)
			if err == nil || present {
				t.Fatal("uncached read did not need its source", present, err)
			}
		})
	}
}

// The byte book holds two tiles; an independent one-entry limit must
// evict the older tile even though another whole payload would fit.
func TestStreamingTileEntryLimitActsBeforeTheByteLimit(t *testing.T) {
	book := terrainContract[streamingBook](t, "streaming")
	budget := streamingBudget(book)
	budget.CacheTiles = 1
	raster, path := contractRaster(t, budget)
	for _, cell := range []sourceCell{book.Cached, book.Evicted} {
		expectSourceCell(t, raster, cell)
		bytes, entries := raster.CacheUsage()
		if bytes != 4096 || entries != 1 {
			t.Fatal("entry cap retained a second tile", bytes, entries)
		}
	}
	if err := os.Truncate(path, 0); err != nil {
		t.Fatal(err)
	}
	expectSourceCell(t, raster, book.Evicted)
	_, present, err := sourceSample(raster, book.Cached)
	if err == nil || present {
		t.Fatal("entry eviction did not require the source", present, err)
	}
}

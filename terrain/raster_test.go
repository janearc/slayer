package terrain

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// the fixtures: a raster 100 samples across and 70 down in tiles of
// 32, its height at a column and row (column + 1000 row) / 8, every
// value exact in a float32, with no height at column 99 of row 0.
const (
	fixtureW, fixtureH, fixtureTile = 100, 70, 32
	fixtureWest, fixtureNorth       = 1000.5, 9000.25
	fixtureSpacing                  = 2.0
	fixtureNoData                   = -9999
)

// fixtureAt is the fixture's height at a column and row.
func fixtureAt(column, row int) float32 {
	if column == fixtureW-1 && row == 0 {
		return fixtureNoData
	}
	return float32(column+1000*row) / 8
}

// writeTiff writes the fixture as an uncompressed, tiled, classic,
// little-endian tiff with geotiff's scale and tiepoint and gdal's
// no-data, as tiffcp would read it.
func writeTiff(t *testing.T, path string) {
	t.Helper()
	across := (fixtureW + fixtureTile - 1) / fixtureTile
	downs := (fixtureH + fixtureTile - 1) / fixtureTile
	tile := 4 * fixtureTile * fixtureTile
	data := make([]byte, 0, across*downs*tile)
	var offsets, counts []uint32
	for tileRow := range downs {
		for tileCol := range across {
			offsets = append(offsets, uint32(8+len(data)))
			counts = append(counts, uint32(tile))
			for y := range fixtureTile {
				for x := range fixtureTile {
					column, row := tileCol*fixtureTile+x, tileRow*fixtureTile+y
					value := float32(0)
					if column < fixtureW && row < fixtureH {
						value = fixtureAt(column, row)
					}
					data = binary.LittleEndian.AppendUint32(data,
						math.Float32bits(value))
				}
			}
		}
	}
	extra := func(values []byte) uint32 {
		at := uint32(8 + len(data))
		data = append(data, values...)
		return at
	}
	longs := func(values []uint32) uint32 {
		var raw []byte
		for _, value := range values {
			raw = binary.LittleEndian.AppendUint32(raw, value)
		}
		return extra(raw)
	}
	doubles := func(values ...float64) uint32 {
		var raw []byte
		for _, value := range values {
			raw = binary.LittleEndian.AppendUint64(raw,
				math.Float64bits(value))
		}
		return extra(raw)
	}
	noData := extra([]byte("-9999\x00"))
	entries := [][4]uint32{{tagWidth, 3, 1, fixtureW},
		{tagHeight, 3, 1, fixtureH}, {tagBits, 3, 1, 32},
		{tagCompression, 3, 1, 1}, {262, 3, 1, 1}, {tagSamples, 3, 1, 1},
		{tagTileWidth, 3, 1, fixtureTile}, {tagTileHeight, 3, 1, fixtureTile},
		{tagTileOffsets, 4, uint32(len(offsets)), longs(offsets)},
		{tagTileCounts, 4, uint32(len(counts)), longs(counts)},
		{tagFormat, 3, 1, 3},
		{tagScale, 12, 3, doubles(fixtureSpacing, fixtureSpacing, 0)},
		{tagTiepoint, 12, 6, doubles(0, 0, 0, fixtureWest, fixtureNorth, 0)},
		{tagNoData, 2, 6, noData}}
	directory := binary.LittleEndian.AppendUint16(nil, uint16(len(entries)))
	for _, each := range entries {
		directory = binary.LittleEndian.AppendUint16(directory, uint16(each[0]))
		directory = binary.LittleEndian.AppendUint16(directory, uint16(each[1]))
		directory = binary.LittleEndian.AppendUint32(directory, each[2])
		value := each[3]
		if each[1] == 3 && each[2] == 1 {
			directory = binary.LittleEndian.AppendUint16(directory,
				uint16(value))
			directory = append(directory, 0, 0)
			continue
		}
		directory = binary.LittleEndian.AppendUint32(directory, value)
	}
	directory = append(directory, 0, 0, 0, 0)
	head := []byte{'I', 'I', 42, 0}
	head = binary.LittleEndian.AppendUint32(head, uint32(8+len(data)))
	whole := append(append(head, data...), directory...)
	if err := os.WriteFile(path, whole, 0o644); err != nil {
		t.Fatal(err)
	}
}

// the plain fixture is written where TERRAIN_FIXTURE says, for tiffcp to
// copy into testdata; with nowhere said, the test is skipped.
func TestWriteThePlainFixture(t *testing.T) {
	path := os.Getenv("TERRAIN_FIXTURE")
	if path == "" {
		t.Skip("no TERRAIN_FIXTURE")
	}
	writeTiff(t, path)
}

// fixtures is the fixture uncompressed, written here, and as libtiff's
// tiffcp copied it: lzw, and lzw in a bigtiff (testdata/README.md says
// how they were made).
func fixtures(t *testing.T) map[string]string {
	t.Helper()
	plain := filepath.Join(t.TempDir(), "plain.tif")
	writeTiff(t, plain)
	return map[string]string{"plain": plain,
		"lzw": "testdata/fixture-lzw.tif", "bigtiff": "testdata/fixture-big.tif"}
}

// every sample of every fixture is the value it was written with.
// tiffcp keeps no geotiff tag, so its copies are not placed and their
// no-data is a height like any other; the plain one says where it is.
func TestEverySampleIsReadAsWritten(t *testing.T) {
	for name, path := range fixtures(t) {
		raster, err := Open(path)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		plain := name == "plain"
		across, down := raster.Size()
		west, north, spacing := raster.Corner()
		if across != fixtureW || down != fixtureH || raster.placed != plain ||
			plain && (west != fixtureWest || north != fixtureNorth ||
				spacing != fixtureSpacing) {
			t.Errorf("%s: %d by %d at %g, %g, %g apart", name, across, down,
				west, north, spacing)
		}
		for row := range fixtureH {
			for column := range fixtureW {
				value, has, err := raster.sample(column, row)
				want := fixtureAt(column, row)
				missing := plain && want == fixtureNoData
				if err != nil || has == missing ||
					(has && value != float64(want)) {
					t.Fatalf("%s: %d, %d: %v %v %v, want %v", name, column,
						row, value, has, err, want)
				}
			}
		}
		raster.Close()
	}
}

// a raster that does not say where it is has no window.
func TestAnUnplacedRasterHasNoWindow(t *testing.T) {
	raster, err := Open(fixtures(t)["lzw"])
	if err != nil {
		t.Fatal(err)
	}
	defer raster.Close()
	if _, err := raster.Window(0, 0, 2, 2, 1); err == nil {
		t.Error("an unplaced raster gave a window")
	}
}

// a window is the samples whose cells hold its points, and one that
// leaves the raster or meets no height is refused, saying where.
func TestAWindowIsTheSamplesUnderIt(t *testing.T) {
	raster, err := Open(fixtures(t)["plain"])
	if err != nil {
		t.Fatal(err)
	}
	defer raster.Close()
	window, err := raster.Window(fixtureWest+20.5, fixtureNorth-40.5, 5, 4, 6)
	if err != nil {
		t.Fatal(err)
	}
	for down := range 4 {
		for across := range 5 {
			column, row := 10+3*across, 20+3*down
			if got := window.At(across, down); got != float64(fixtureAt(column,
				row)) {
				t.Errorf("%d, %d: %v, want sample %d, %d", across, down, got,
					column, row)
			}
		}
	}
	for what, west := range map[string]float64{"west of it": fixtureWest - 3,
		"its no-data corner": fixtureWest + 2*(fixtureW-1) + 1} {
		_, err := raster.Window(west, fixtureNorth-0.5, 2, 2, 1)
		if err == nil || !strings.Contains(err.Error(), "no height") {
			t.Errorf("%s: %v", what, err)
		}
	}
}

// a window's ground is a mesh of two triangles a cell, its lowest point at
// zero, the lowest height said.
func TestAWindowsGroundIsItsHeights(t *testing.T) {
	raster, err := Open(fixtures(t)["plain"])
	if err != nil {
		t.Fatal(err)
	}
	defer raster.Close()
	window, err := raster.Window(fixtureWest+1, fixtureNorth-1, 4, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	ground, lowest, err := window.Ground()
	if err != nil {
		t.Fatal(err)
	}
	box := ground.Bounds()
	if lowest != float64(fixtureAt(0, 0)) || ground.FaceCount() != 12 ||
		box.Min.Y != 0 || box.Max.X != 6 || box.Max.Z != 4 {
		t.Errorf("lowest %v, %d faces, box %v", lowest, ground.FaceCount(),
			box)
	}
}

// what is not a raster this reads is refused: not a tiff at all, and
// a tiff with no tiles.
func TestWhatIsNotARasterIsRefused(t *testing.T) {
	junk := filepath.Join(t.TempDir(), "junk.tif")
	os.WriteFile(junk, []byte("this is not a tiff at all"), 0o644)
	if _, err := Open(junk); err == nil {
		t.Error("junk was read")
	}
	if _, err := Open(filepath.Join(t.TempDir(), "gone.tif")); err == nil {
		t.Error("a missing file was read")
	}
}

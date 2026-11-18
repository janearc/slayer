package terrain

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// lzwLiterals encodes data as TIFF LZW using only literal codes, which keeps
// the code width at nine bits for the short payloads these tests use. The end
// code is left out when withEnd is false.
func lzwLiterals(data []byte, withEnd bool) []byte {
	var out []byte
	var held uint32
	var width uint
	put := func(code uint32) {
		held = held<<9 | code
		width += 9
		for width >= 8 {
			out = append(out, byte(held>>(width-8)))
			width -= 8
		}
	}
	put(256)
	for _, b := range data {
		put(uint32(b))
	}
	if withEnd {
		put(257)
	}
	if width > 0 {
		out = append(out, byte(held<<(8-width)))
	}
	return out
}

// floatBytes is the little-endian float32 bytes of the heights.
func floatBytes(values ...float32) []byte {
	var raw []byte
	for _, value := range values {
		raw = binary.LittleEndian.AppendUint32(raw, math.Float32bits(value))
	}
	return raw
}

// TestMalformedRastersAreRefused builds files that break one rule each and
// checks that opening them says which rule.
func TestMalformedRastersAreRefused(t *testing.T) {
	base := func() *tiffSpec {
		return newTiffSpec(tiffLittle, false, 5, 4, 4, 4, rampAt).place(100, 50, 1, 1)
	}
	cases := []struct {
		name string
		edit func(*tiffSpec)
		want string
	}{
		{"fill order", func(s *tiffSpec) { s.shorts(tagFillOrder, 2) }, "fill order 2"},
		{"predictor", func(s *tiffSpec) { s.shorts(tagPredictor, 2) }, "predictor 2"},
		{"no width", func(s *tiffSpec) { s.shorts(tagWidth, 0) }, "no size"},
		{"empty width tag", func(s *tiffSpec) { s.tags[tagWidth] = tiffTag{id: tagWidth, kind: 3} }, "no size"},
		{"not tiled", func(s *tiffSpec) { delete(s.tags, tagTileWidth); delete(s.tags, tagTileHeight) }, "not tiled"},
		{"sixteen bits", func(s *tiffSpec) { s.shorts(tagBits, 16) }, "not float32"},
		{"integer format", func(s *tiffSpec) { s.shorts(tagFormat, 1) }, "not float32"},
		{"three bands", func(s *tiffSpec) { s.shorts(tagSamples, 3) }, "more than one band"},
		{"compression", func(s *tiffSpec) { s.shorts(tagCompression, 7) }, "compression 7"},
		{"too few offsets", func(s *tiffSpec) { s.offsets = []uint64{8} }, "1 tile offsets"},
		{"too few counts", func(s *tiffSpec) { s.counts = []uint64{8} }, "1 counts"},
		{"scale without tiepoint", func(s *tiffSpec) { delete(s.tags, tagTiepoint) }, "not placed"},
		{"tiepoint off the origin", func(s *tiffSpec) { s.doubles(tagTiepoint, 1, 0, 0, 100, 50, 0) }, "not placed"},
		{"unequal projected scales", func(s *tiffSpec) { s.place(100, 50, 1, 2) }, "not placed"},
		{"negative scale", func(s *tiffSpec) { s.place(100, 50, -1, -1) }, "not placed"},
		{"infinite tiepoint", func(s *tiffSpec) { s.place(math.Inf(1), 50, 1, 1) }, "not placed"},
		{"extent overflow", func(s *tiffSpec) { s.place(0, 0, 1e308, 1e308) }, "extent overflows"},
		{"bad no-data", func(s *tiffSpec) { s.ascii(tagNoData, "none") }, "no-data"},
		{"bad geokeys", func(s *tiffSpec) { s.shorts(34735, 1, 1) }, "GeoKey"},
	}
	for _, each := range cases {
		spec := base()
		each.edit(spec)
		_, err := Open(spec.write(t))
		if err == nil || !strings.Contains(err.Error(), each.want) {
			t.Errorf("%s: %v, want %q", each.name, err, each.want)
		}
	}
}

// TestTileDirectoryOverTheMetadataBudgetIsRefused checks that a directory
// that would need more offsets than the metadata budget holds is refused
// before any offsets are read.
func TestTileDirectoryOverTheMetadataBudgetIsRefused(t *testing.T) {
	spec := newTiffSpec(tiffLittle, false, 5, 4, 4, 4, rampAt)
	spec.shorts(tagWidth, 2000).shorts(tagHeight, 2000).shorts(tagTileWidth, 1).shorts(tagTileHeight, 1)
	_, err := Open(spec.write(t))
	if err == nil || !strings.Contains(err.Error(), "tile directory capacity") {
		t.Fatalf("got %v", err)
	}
}

// TestUnknownAndByteTagsAreRead checks that a tag of a type the reader does
// not know is skipped, and that a one-byte tag is read as a number.
func TestUnknownAndByteTagsAreRead(t *testing.T) {
	spec := newTiffSpec(tiffLittle, false, 5, 4, 4, 4, rampAt).place(100, 50, 1, 1)
	spec.tags[40000] = tiffTag{id: 40000, kind: 5, count: 1, raw: make([]byte, 8)}
	spec.tags[tagSamples] = tiffTag{id: tagSamples, kind: 1, count: 1, raw: []byte{1}}
	raster := openSpec(t, spec)
	if value, has, err := raster.sample(4, 3); err != nil || !has || value != 34 {
		t.Fatalf("sample %v %v %v", value, has, err)
	}
}

// TestBrokenHeadersAndDirectoriesAreRefused checks files that fail while the
// header or directory is read.
func TestBrokenHeadersAndDirectoriesAreRefused(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, raw []byte) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, raw, 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	good := newTiffSpec(tiffLittle, false, 5, 4, 4, 4, rampAt).bytes()
	longCount := bytes.Clone(good)
	binary.LittleEndian.PutUint16(longCount[binary.LittleEndian.Uint32(longCount[4:]):], 5000)
	farDirectory := bytes.Clone(good)
	binary.LittleEndian.PutUint32(farDirectory[4:], uint32(len(good)+100))
	bigShort := []byte{'I', 'I', 43, 0, 8, 0, 0, 0}
	external := newTiffSpec(tiffLittle, false, 5, 4, 4, 4, rampAt)
	external.tags[tagNoData] = tiffTag{id: tagNoData, kind: 1, count: 100000, raw: []byte{1, 2, 3, 4, 5}}
	huge := newTiffSpec(tiffLittle, false, 5, 4, 4, 4, rampAt)
	huge.tags[tagNoData] = tiffTag{id: tagNoData, kind: 1, count: 1 << 30, raw: []byte{1, 2, 3, 4, 5}}
	cases := map[string]struct {
		path string
		want string
	}{
		"empty":               {write("empty", nil), "EOF"},
		"unknown version":     {write("version", []byte{'I', 'I', 99, 0, 8, 0, 0, 0}), "not a tiff or a bigtiff"},
		"short bigtiff":       {write("bigshort", bigShort), "EOF"},
		"directory too far":   {write("far", farDirectory), "outside the source file"},
		"too many entries":    {write("long", longCount), "outside the source file"},
		"value too far":       {write("external", external.bytes()), "outside the source file"},
		"value over budget":   {write("huge", huge.bytes()), "byte budget"},
		"no byte order":       {write("order", []byte("XX*\x00\x08\x00\x00\x00")), "no byte order"},
		"directory not there": {filepath.Join(dir, "none"), "no such file"},
	}
	for name, each := range cases {
		_, err := Open(each.path)
		if err == nil || !strings.Contains(err.Error(), each.want) {
			t.Errorf("%s: %v, want %q", name, err, each.want)
		}
	}
}

// TestSmallMetadataBudgetRefusesTheDirectory checks that a directory larger
// than the metadata budget is refused, and so is one whose tag values are.
func TestSmallMetadataBudgetRefusesTheDirectory(t *testing.T) {
	spec := newTiffSpec(tiffLittle, false, 5, 4, 4, 4, rampAt).place(100, 50, 1, 1)
	path := spec.write(t)
	for _, limit := range []int64{64, 160} {
		budget := RasterBudget{CacheBytes: 1 << 20, CacheTiles: 4, TileBytes: 1 << 20, MetadataBytes: limit}
		if _, err := OpenWithBudget(path, budget); err == nil ||
			!strings.Contains(err.Error(), "byte budget") {
			t.Errorf("metadata limit %d: %v", limit, err)
		}
	}
}

// TestMetadataAndSectionReadsAreBounded reads the raw helpers directly: a
// section past the file, a read over the remaining budget, and a file that is
// shorter than its size was said to be.
func TestMetadataAndSectionReadsAreBounded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ten")
	if err := os.WriteFile(path, []byte("0123456789"), 0o644); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	raster := &Raster{file: file, fileBytes: 10, budget: RasterBudget{MetadataBytes: 6}}
	if _, err := raster.section(11, 0); err == nil {
		t.Error("a section starting past the file was made")
	}
	if _, err := raster.section(4, 7); err == nil {
		t.Error("a section running past the file was made")
	}
	raw, err := raster.readMetadata(2, 4)
	if err != nil || string(raw) != "2345" || raster.metadataBytes != 4 {
		t.Fatalf("read %q, %v, spent %d", raw, err, raster.metadataBytes)
	}
	if _, err := raster.readMetadata(0, 3); err == nil || !strings.Contains(err.Error(), "metadata exceeds") {
		t.Errorf("a read over the budget: %v", err)
	}
	if _, err := raster.readMetadata(9, 2); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Errorf("a read past the file: %v", err)
	}
	raster.fileBytes = 20
	raster.budget.MetadataBytes = 100
	if _, err := raster.readMetadata(5, 10); err == nil {
		t.Error("a file shorter than it said was read in full")
	}
	if raster.metadataBytes != 4 {
		t.Errorf("a failed read was charged: %d", raster.metadataBytes)
	}
}

// TestStrippedRasterReadsItsShortLastStrip builds a file in strips of three
// rows, the last strip holding one row, and checks every sample.
func TestStrippedRasterReadsItsShortLastStrip(t *testing.T) {
	spec := newTiffSpec(tiffLittle, false, 5, 4, 5, 3, rampAt).place(100, 50, 1, 1)
	delete(spec.tags, tagTileWidth)
	delete(spec.tags, tagTileHeight)
	spec.shorts(tagRowsPerStrip, 3)
	spec.tiles[1] = spec.tiles[1][:5*4]
	raster := openSpec(t, spec)
	for row := range 4 {
		for column := range 5 {
			got, has, err := raster.sample(column, row)
			if err != nil || !has || got != float64(rampAt(column, row)) {
				t.Fatalf("%d,%d: %v %v %v", column, row, got, has, err)
			}
		}
	}
}

// TestLzwTilesMustEndWhereTheGridEnds checks a well formed compressed tile,
// then one with a missing end code and one with output past the tile grid.
func TestLzwTilesMustEndWhereTheGridEnds(t *testing.T) {
	values := make([]float32, 16)
	for i := range values {
		values[i] = float32(i) + 0.5
	}
	cases := []struct {
		name    string
		payload []byte
		want    string
	}{
		{"whole", lzwLiterals(floatBytes(values...), true), ""},
		{"no end code", lzwLiterals(floatBytes(values...), false), "missing compressed tile end"},
		{"extra output", lzwLiterals(append(floatBytes(values...), 1), true), "exceeds the declared tile grid"},
		{"short output", lzwLiterals(floatBytes(values[:3]...), true), "EOF"},
	}
	for _, each := range cases {
		spec := newTiffSpec(tiffLittle, false, 4, 4, 4, 4, rampAt)
		spec.shorts(tagCompression, lzwed)
		spec.tiles = [][]byte{each.payload}
		raster := openSpec(t, spec)
		got, _, err := raster.sample(3, 3)
		if each.want == "" {
			if err != nil || got != 15.5 {
				t.Errorf("%s: %v %v", each.name, got, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), each.want) {
			t.Errorf("%s: %v, want %q", each.name, err, each.want)
		}
	}
}

// TestTileReadFailuresAreReported checks a tile whose bytes are outside the
// file, and a floating predictor tile that cannot hold its scanline buffer.
func TestTileReadFailuresAreReported(t *testing.T) {
	spec := newTiffSpec(tiffLittle, false, 4, 4, 4, 4, rampAt)
	spec.offsets = []uint64{1 << 20}
	raster := openSpec(t, spec)
	if _, _, err := raster.sample(0, 0); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Errorf("tile outside the file: %v", err)
	}
	predicted := newTiffSpec(tiffLittle, false, 4, 4, 4, 4, rampAt)
	predicted.shorts(tagPredictor, 3)
	path := predicted.write(t)
	budget := RasterBudget{CacheBytes: 0, CacheTiles: 0, TileBytes: 8, MetadataBytes: 1 << 20}
	small, err := OpenWithBudget(path, budget)
	if err != nil {
		t.Fatal(err)
	}
	defer small.Close()
	if _, _, err := small.sample(0, 0); err == nil || !strings.Contains(err.Error(), "capacity") {
		t.Errorf("scanline over the tile budget: %v", err)
	}
}

// TestFloatingPredictorRefusesARaggedTile checks that a sample count which is
// not a whole number of scanlines is refused, and so is a zero-width scanline.
func TestFloatingPredictorRefusesARaggedTile(t *testing.T) {
	raster := Raster{tileW: 4, order: binary.LittleEndian, predictor: 3}
	if _, err := raster.decode(bytes.NewReader(nil), 6); err == nil {
		t.Error("six samples in rows of four were accepted")
	}
	raster.tileW = 0
	if _, err := raster.decode(bytes.NewReader(nil), 0); err == nil {
		t.Error("a zero-width scanline was accepted")
	}
}

// room for: a payload that fits its limit while the cache's own books
// claim more is in use than the list holds.
func TestTileCacheEvictsOldestFirst(t *testing.T) {
	cache := newTileCache(16, 2)
	cache.put(1, make([]float32, 2))
	cache.put(2, make([]float32, 2))
	cache.put(3, make([]float32, 2))
	if _, found := cache.get(1); found {
		t.Error("the oldest tile outlived the entry limit")
	}
	if _, found := cache.get(3); !found || cache.used != 16 {
		t.Errorf("newest tile missing, %d bytes in use", cache.used)
	}
	cache.put(3, make([]float32, 4))
	if _, found := cache.get(2); found || cache.used != 16 {
		t.Errorf("replacing a tile kept the old: %d bytes in use", cache.used)
	}
	cache.put(4, make([]float32, 5))
	if len(cache.items) != 1 {
		t.Errorf("an oversize payload was kept: %d items", len(cache.items))
	}
}

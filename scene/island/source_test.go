package island

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/terrain"
)

// tiffEntry is one directory entry: its tag, type, count and value bytes.
type tiffEntry struct {
	id, kind uint16
	count    uint32
	raw      []byte
}

// shorts is a tiff entry of 16-bit numbers.
func shorts(id uint16, values ...uint16) tiffEntry {
	var raw []byte
	for _, value := range values {
		raw = binary.LittleEndian.AppendUint16(raw, value)
	}
	return tiffEntry{id, 3, uint32(len(values)), raw}
}

// doubles is a tiff entry of 64-bit floats.
func doubles(id uint16, values ...float64) tiffEntry {
	var raw []byte
	for _, value := range values {
		raw = binary.LittleEndian.AppendUint64(raw, math.Float64bits(value))
	}
	return tiffEntry{id, 12, uint32(len(values)), raw}
}

// geoKeys is a GeoKey directory entry of key, location, count, value rows.
func geoKeys(rows ...[4]uint16) tiffEntry {
	values := []uint16{1, 1, 0, uint16(len(rows))}
	for _, row := range rows {
		values = append(values, row[:]...)
	}
	return shorts(34735, values...)
}

// degreeKeys declares a degree grid with the given pixel registration and
// vertical units, where a zero means the key is left out.
func degreeKeys(registration, vertical uint16) tiffEntry {
	rows := [][4]uint16{{1024, 0, 1, 2}, {1025, 0, 1, registration},
		{2048, 0, 1, 4326}, {2054, 0, 1, 9102}}
	if vertical != 0 {
		rows = append(rows, [4]uint16{4099, 0, 1, vertical})
	}
	return geoKeys(rows...)
}

// writeRaster writes a four by four float32 raster, one tile, whose sample
// at column and row is column plus ten times row, first sample at longitude
// minus one and latitude one, a degree apart, with the extra entries, and
// opens it.
func writeRaster(t *testing.T, extra ...tiffEntry) *terrain.Raster {
	t.Helper()
	order := binary.LittleEndian
	body := make([]byte, 8)
	for row := range 4 {
		for column := range 4 {
			body = order.AppendUint32(body, math.Float32bits(float32(column+10*row)))
		}
	}
	entries := []tiffEntry{shorts(256, 4), shorts(257, 4), shorts(258, 32), shorts(259, 1),
		shorts(277, 1), shorts(322, 4), shorts(323, 4), shorts(339, 3),
		{324, 4, 1, order.AppendUint32(nil, 8)}, {325, 4, 1, order.AppendUint32(nil, 64)},
		doubles(33550, 1, 1, 0), doubles(33922, 0, 0, 0, -1, 1, 0)}
	entries = append(entries, extra...)
	slices.SortFunc(entries, func(one, other tiffEntry) int { return int(one.id) - int(other.id) })
	var table []byte
	for _, entry := range entries {
		table = order.AppendUint16(table, entry.id)
		table = order.AppendUint16(table, entry.kind)
		table = order.AppendUint32(table, entry.count)
		if len(entry.raw) > 4 {
			table = order.AppendUint32(table, uint32(len(body)))
			body = append(body, entry.raw...)
			continue
		}
		table = append(table, entry.raw...)
		table = append(table, make([]byte, 4-len(entry.raw))...)
	}
	directory := uint32(len(body))
	body = order.AppendUint16(body, uint16(len(entries)))
	body = append(body, table...)
	body = append(body, 0, 0, 0, 0)
	copy(body, "II")
	order.PutUint16(body[2:], 42)
	order.PutUint32(body[4:], directory)
	path := filepath.Join(t.TempDir(), "made.tif")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	raster, err := terrain.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { raster.Close() })
	return raster
}

// TestNewRasterHeightsChecksItsReference checks a good point-registered
// source and then each reference mismatch.
func TestNewRasterHeightsChecksItsReference(t *testing.T) {
	good := writeRaster(t, degreeKeys(2, 9001))
	source, err := NewRasterHeights(good, 4326, "test datum", measure.Metres(25))
	if err != nil {
		t.Fatal(err)
	}
	crs, datum, offset := source.Reference()
	if crs != 4326 || datum != "test datum" || offset.InMetres() != 25 {
		t.Fatalf("reference %d, %q, %v", crs, datum, offset)
	}
	if _, err := NewRasterHeights(writeRaster(t, degreeKeys(1, 0)), 4326, "area", measure.Metres(0)); err != nil {
		t.Fatalf("an area-registered source with no vertical units: %v", err)
	}
	closed := writeRaster(t, degreeKeys(2, 0))
	if err := closed.Close(); err != nil {
		t.Fatal(err)
	}
	projected := writeRaster(t, geoKeys([4]uint16{1024, 0, 1, 1}, [4]uint16{1025, 0, 1, 2}))
	feet := writeRaster(t, degreeKeys(2, 9002))
	unspecified := writeRaster(t, degreeKeys(0, 0))
	cases := []struct {
		name   string
		raster *terrain.Raster
		crs    int
		datum  string
		offset measure.Length
	}{
		{"no raster", nil, 4326, "datum", measure.Metres(0)},
		{"no crs", good, 0, "datum", measure.Metres(0)},
		{"blank datum", good, 4326, "  ", measure.Metres(0)},
		{"infinite offset", good, 4326, "datum", measure.Metres(math.Inf(1))},
		{"other crs", good, 4269, "datum", measure.Metres(0)},
		{"projected model", projected, 4326, "datum", measure.Metres(0)},
		{"non-metre heights", feet, 4326, "datum", measure.Metres(0)},
		{"unspecified registration", unspecified, 4326, "datum", measure.Metres(0)},
		{"closed raster", closed, 4326, "datum", measure.Metres(0)},
	}
	for _, each := range cases {
		t.Run(each.name, func(t *testing.T) {
			if source, err := NewRasterHeights(each.raster, each.crs, each.datum, each.offset); err == nil || source != nil {
				t.Fatalf("got %v, %v", source, err)
			}
		})
	}
}

// TestRasterHeightsInterpolatesAndOffsets checks worked heights: the ramp
// at a sample, between two samples and between four, each raised by the
// declared offset, and an absent answer outside the raster.
func TestRasterHeightsInterpolatesAndOffsets(t *testing.T) {
	source, err := NewRasterHeights(writeRaster(t, degreeKeys(2, 9001)), 4326, "datum", measure.Metres(100))
	if err != nil {
		t.Fatal(err)
	}
	for _, each := range []struct{ longitude, latitude, want float64 }{
		{-1, 1, 100}, {0, 0, 111}, {-0.5, 1, 100.5}, {0.5, 0.5, 106.5}, {2, -2, 100 + 33},
	} {
		got, has, err := source.Height(each.longitude, each.latitude)
		if err != nil || !has || math.Abs(got-each.want) > 1e-9 {
			t.Errorf("Height(%g, %g) = %g, %v, %v; wanted %g", each.longitude, each.latitude, got, has, err, each.want)
		}
	}
	if got, has, err := source.Height(5, 0); err != nil || has || got != 0 {
		t.Errorf("outside the raster: %g, %v, %v", got, has, err)
	}
	if _, _, err := source.Height(200, 0); err == nil {
		t.Error("a longitude past the antimeridian was accepted")
	}
	var nobody *RasterHeights
	if _, _, err := nobody.Height(0, 0); err == nil {
		t.Error("a nil source answered")
	}
	if _, _, err := (&RasterHeights{}).Height(0, 0); err == nil {
		t.Error("an empty source answered")
	}
}

// TestRasterHeightsRangeNeedsAnIndex checks that an unindexed source claims
// no bound, an indexed one reports its extremes plus the offset, and a
// region off the raster has none.
func TestRasterHeightsRangeNeedsAnIndex(t *testing.T) {
	source, err := NewRasterHeights(writeRaster(t, degreeKeys(2, 9001)), 4326, "datum", measure.Metres(100))
	if err != nil {
		t.Fatal(err)
	}
	if low, high, has, err := source.HeightRange(-1, -2, 2, 1); err != nil || has || low != 0 || high != 0 {
		t.Fatalf("unindexed: %g, %g, %v, %v", low, high, has, err)
	}
	indexed, err := source.Indexed(context.Background(), 2, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if source.index != nil {
		t.Fatal("indexing changed the source it was made from")
	}
	low, high, has, err := indexed.HeightRange(-1, -2, 2, 1)
	if err != nil || !has || low != 100 || high != 133 {
		t.Fatalf("indexed: %g, %g, %v, %v", low, high, has, err)
	}
	if _, _, has, err := indexed.HeightRange(50, 50, 51, 51); err != nil || has {
		t.Fatalf("off the raster: %v, %v", has, err)
	}
	if _, _, _, err := indexed.HeightRange(math.NaN(), 0, 1, 1); err == nil {
		t.Fatal("a region that is not a number was accepted")
	}
	var nobody *RasterHeights
	if _, _, has, err := nobody.HeightRange(0, 0, 1, 1); has || err != nil {
		t.Fatalf("nil source: %v, %v", has, err)
	}
}

// TestIndexedRefusals checks the refusals of Indexed.
func TestIndexedRefusals(t *testing.T) {
	source, err := NewRasterHeights(writeRaster(t, degreeKeys(2, 9001)), 4326, "datum", measure.Metres(0))
	if err != nil {
		t.Fatal(err)
	}
	var nobody *RasterHeights
	if _, err := nobody.Indexed(context.Background(), 2, 1<<20); err == nil {
		t.Error("a nil source was indexed")
	}
	if _, err := (&RasterHeights{}).Indexed(context.Background(), 2, 1<<20); err == nil {
		t.Error("an empty source was indexed")
	}
	if _, err := source.Indexed(context.Background(), 0, 1<<20); err == nil {
		t.Error("a block of zero was accepted")
	}
}

// layerStub is a source with fixed answers that counts its calls.
type layerStub struct {
	height float64
	has    bool
	err    error
	calls  int
}

// Height answers as configured.
func (stub *layerStub) Height(float64, float64) (float64, bool, error) {
	stub.calls++
	return stub.height, stub.has, stub.err
}

// boundedStub is a layerStub that also reports a range.
type boundedStub struct {
	layerStub
	low, high float64
	has       bool
	err       error
}

// HeightRange answers as configured.
func (stub *boundedStub) HeightRange(float64, float64, float64, float64) (float64, float64, bool, error) {
	return stub.low, stub.high, stub.has, stub.err
}

// TestNewLayersRefusals checks the empty list and an absent layer, and that
// the layers keep their own copy of the list.
func TestNewLayersRefusals(t *testing.T) {
	if layers, err := NewLayers(); err == nil || layers != nil {
		t.Errorf("no sources: %v, %v", layers, err)
	}
	if layers, err := NewLayers(&layerStub{}, nil); err == nil || layers != nil {
		t.Errorf("an absent source: %v, %v", layers, err)
	}
	list := []Heights{&layerStub{height: 1, has: true}}
	layers, err := NewLayers(list...)
	if err != nil {
		t.Fatal(err)
	}
	list[0] = &layerStub{height: 2, has: true}
	if got, _, _ := layers.Height(0, 0); got != 1 {
		t.Errorf("a change to the caller's list changed the layers: %g", got)
	}
}

// TestLayersFallBackOnlyOnAbsence checks that the first layer with an answer
// wins, that an absent answer falls through, and that an error stops the
// search without asking later layers.
func TestLayersFallBackOnlyOnAbsence(t *testing.T) {
	sentinel := errors.New("layer failed")
	missing, second, third := &layerStub{}, &layerStub{height: 7, has: true}, &layerStub{height: 9, has: true}
	layers, err := NewLayers(missing, second, third)
	if err != nil {
		t.Fatal(err)
	}
	if got, has, err := layers.Height(0, 0); err != nil || !has || got != 7 || third.calls != 0 || missing.calls != 1 {
		t.Fatalf("got %g, %v, %v; calls %d %d", got, has, err, missing.calls, third.calls)
	}
	broken := &layerStub{err: sentinel}
	later := &layerStub{height: 1, has: true}
	layers, err = NewLayers(broken, later)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := layers.Height(0, 0); !errors.Is(err, sentinel) || later.calls != 0 {
		t.Fatalf("error %v, later layer asked %d times", err, later.calls)
	}
	none, err := NewLayers(&layerStub{}, &layerStub{})
	if err != nil {
		t.Fatal(err)
	}
	if got, has, err := none.Height(0, 0); err != nil || has || got != 0 {
		t.Fatalf("all absent: %g, %v, %v", got, has, err)
	}
	var nobody *Layers
	if _, _, err := nobody.Height(0, 0); err == nil {
		t.Error("a nil set of layers answered")
	}
	if _, _, err := (&Layers{}).Height(0, 0); err == nil || !strings.Contains(err.Error(), "no height layers") {
		t.Errorf("an empty set of layers answered: %v", err)
	}
}

// TestLayersRangeCoversEveryLayer checks that the range is the union of the
// layers' ranges, and that any layer without a bound, with an error or with
// no data removes the bound.
func TestLayersRangeCoversEveryLayer(t *testing.T) {
	sentinel := errors.New("range failed")
	bounded := func(low, high float64) *boundedStub { return &boundedStub{low: low, high: high, has: true} }
	build := func(sources ...Heights) *Layers {
		layers, err := NewLayers(sources...)
		if err != nil {
			t.Fatal(err)
		}
		return layers
	}
	low, high, has, err := build(bounded(2, 5), bounded(-3, 4), bounded(0, 9)).HeightRange(0, 0, 1, 1)
	if err != nil || !has || low != -3 || high != 9 {
		t.Fatalf("union: %g, %g, %v, %v", low, high, has, err)
	}
	if _, _, has, err := build(bounded(2, 5), &layerStub{}).HeightRange(0, 0, 1, 1); has || err != nil {
		t.Errorf("a layer with no range method: %v, %v", has, err)
	}
	if _, _, has, err := build(bounded(2, 5), &boundedStub{}).HeightRange(0, 0, 1, 1); has || err != nil {
		t.Errorf("a layer with no data: %v, %v", has, err)
	}
	if _, _, has, err := build(&boundedStub{err: sentinel}, bounded(2, 5)).HeightRange(0, 0, 1, 1); has || !errors.Is(err, sentinel) {
		t.Errorf("a layer that failed: %v, %v", has, err)
	}
	var nobody *Layers
	if _, _, has, err := nobody.HeightRange(0, 0, 1, 1); has || err != nil {
		t.Errorf("nil layers: %v, %v", has, err)
	}
	if _, _, has, err := (&Layers{}).HeightRange(0, 0, 1, 1); has || err != nil {
		t.Errorf("empty layers: %v, %v", has, err)
	}
}

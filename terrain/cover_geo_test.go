package terrain

import (
	"context"
	"math"
	"os"
	"strings"
	"testing"
)

// geoSpec is a five by four degree grid, tiles of four, first sample at 10
// east and 20 north, one degree apart in both axes, its height column plus
// ten times row, with the given pixel registration.
func geoSpec(registration uint16) *tiffSpec {
	return newTiffSpec(tiffLittle, false, 5, 4, 4, 4, rampAt).degrees(registration).place(10, 20, 1, 1)
}

// TestGeographicElevationInterpolatesBilinearly checks hand worked values on
// a ramp, in both pixel registrations and with unequal axis scales.
func TestGeographicElevationInterpolatesBilinearly(t *testing.T) {
	half := geoSpec(2).place(10, 20, 1, 0.5)
	cases := []struct {
		name         string
		spec         *tiffSpec
		lon, lat     float64
		want         float64
		wantPresent  bool
		wantNoHeight bool
	}{
		{"point, middle of a cell", geoSpec(2), 10.5, 19.5, 5.5, true, false},
		{"point, on a sample", geoSpec(2), 12, 18, 22, true, false},
		{"point, last column", geoSpec(2), 14, 20, 4, true, false},
		{"point, last row", geoSpec(2), 10, 17, 30, true, false},
		{"area, centre of first cell", geoSpec(1), 10.5, 19.5, 0, true, false},
		{"area, corner between four cells", geoSpec(1), 11, 19, 5.5, true, false},
		{"point, half-degree rows", half, 10.5, 19.75, 5.5, true, false},
		{"point, west of the lattice", geoSpec(2), 9.5, 19.5, 0, false, false},
		{"point, south of the lattice", geoSpec(2), 10.5, 16.5, 0, false, false},
		{"area, half a cell short of the first centre", geoSpec(1), 10.25, 19.5, 0, false, false},
	}
	for _, each := range cases {
		raster := openSpec(t, each.spec)
		got, present, err := raster.GeographicElevation(each.lon, each.lat)
		if err != nil || present != each.wantPresent || got != each.want {
			t.Errorf("%s: %v %v %v, want %v %v", each.name, got, present, err, each.want, each.wantPresent)
		}
	}
}

// TestGeographicElevationKeepsNoDataAbsent checks that a no-data corner makes
// a query absent only when it carries weight.
func TestGeographicElevationKeepsNoDataAbsent(t *testing.T) {
	spec := newTiffSpec(tiffLittle, false, 5, 4, 4, 4, func(column, row int) float32 {
		if column == 1 && row == 0 {
			return -9999
		}
		return rampAt(column, row)
	}).degrees(2).place(10, 20, 1, 1).ascii(tagNoData, "-9999")
	raster := openSpec(t, spec)
	if got, present, err := raster.GeographicElevation(10, 20); err != nil || !present || got != 0 {
		t.Errorf("on the sample beside the hole: %v %v %v", got, present, err)
	}
	if got, present, err := raster.GeographicElevation(10.5, 20); err != nil || present || got != 0 {
		t.Errorf("between the hole and a sample: %v %v %v", got, present, err)
	}
	if got, present, err := raster.GeographicElevation(12.5, 19.5); err != nil || !present || got != 7.5 {
		t.Errorf("away from the hole: %v %v %v", got, present, err)
	}
}

// TestGeographicElevationRefusesWhatItCannotAnswer checks each refusal.
func TestGeographicElevationRefusesWhatItCannotAnswer(t *testing.T) {
	projected := openSpec(t, newTiffSpec(tiffLittle, false, 5, 4, 4, 4, rampAt).place(10, 20, 1, 1))
	noCRS := openSpec(t, newTiffSpec(tiffLittle, false, 5, 4, 4, 4, rampAt).place(10, 20, 1, 1).
		geo([4]uint16{1024, 0, 1, 2}, [4]uint16{1025, 0, 1, 2}, [4]uint16{2054, 0, 1, 9102}))
	unregistered := openSpec(t, newTiffSpec(tiffLittle, false, 5, 4, 4, 4, rampAt).place(10, 20, 1, 1).
		geo([4]uint16{1024, 0, 1, 2}, [4]uint16{2048, 0, 1, 4326}, [4]uint16{2054, 0, 1, 9102}))
	unplaced := openSpec(t, newTiffSpec(tiffLittle, false, 5, 4, 4, 4, rampAt).degrees(2))
	narrow := openSpec(t, newTiffSpec(tiffLittle, false, 1, 4, 4, 4, rampAt).degrees(2).place(10, 20, 1, 1))
	flat := openSpec(t, newTiffSpec(tiffLittle, false, 4, 1, 4, 4, rampAt).degrees(2).place(10, 20, 1, 1))
	closed := openSpec(t, geoSpec(2))
	closed.Close()
	good := openSpec(t, geoSpec(2))
	var missing *Raster
	cases := []struct {
		name     string
		raster   *Raster
		lon, lat float64
		want     string
	}{
		{"nil raster", missing, 11, 19, "unavailable"},
		{"closed raster", closed, 11, 19, "unavailable"},
		{"unplaced raster", unplaced, 11, 19, "unavailable"},
		{"projected raster", projected, 11, 19, "declared degree grid"},
		{"no geographic CRS", noCRS, 11, 19, "declared degree grid"},
		{"no pixel registration", unregistered, 11, 19, "declared degree grid"},
		{"NaN longitude", good, math.NaN(), 19, "invalid geographic query"},
		{"infinite latitude", good, 11, math.Inf(1), "invalid geographic query"},
		{"longitude past the antimeridian", good, 181, 19, "invalid geographic query"},
		{"longitude west of it", good, -181, 19, "invalid geographic query"},
		{"latitude past the pole", good, 11, 91, "invalid geographic query"},
		{"latitude south of it", good, 11, -91, "invalid geographic query"},
		{"one column", narrow, 10, 19, "two samples per axis"},
		{"one row", flat, 10, 20, "two samples per axis"},
	}
	for _, each := range cases {
		_, present, err := each.raster.GeographicElevation(each.lon, each.lat)
		if err == nil || present || !strings.Contains(err.Error(), each.want) {
			t.Errorf("%s: %v %v, want %q", each.name, present, err, each.want)
		}
	}
}

// TestGeographicElevationReportsAnUnreadableTile checks that a read failure
// is an error, not an absent height.
func TestGeographicElevationReportsAnUnreadableTile(t *testing.T) {
	spec := geoSpec(2)
	path := spec.write(t)
	raster, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer raster.Close()
	if err := os.Truncate(path, 8); err != nil {
		t.Fatal(err)
	}
	if _, present, err := raster.GeographicElevation(10.5, 19.5); err == nil || present {
		t.Errorf("an unreadable tile gave %v, %v", present, err)
	}
}

// gridSpec is a five by four projected raster with its origin at 100 east and
// 50 north, one metre cells, height column plus ten row, and no height at
// column 4 of row 0.
func gridSpec() *tiffSpec {
	return newTiffSpec(tiffLittle, false, 5, 4, 4, 4, func(column, row int) float32 {
		if column == 4 && row == 0 {
			return -9999
		}
		return rampAt(column, row)
	}).place(100, 50, 1, 1).ascii(tagNoData, "-9999")
}

// TestReadGridSamplesTheContainingCells checks heights and availability of a
// grid on hand worked cells, including cells outside the source and a
// no-data cell.
func TestReadGridSamplesTheContainingCells(t *testing.T) {
	raster := openSpec(t, gridSpec())
	request := gridRequest{West: 102.5, North: 49.5, Step: 1, Across: 4, Down: 3}
	grid, err := raster.ReadGrid(request, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if grid.Layout() != request {
		t.Errorf("layout %+v", grid.Layout())
	}
	want := [3][4]struct {
		height  float64
		present bool
	}{
		{{2, true}, {3, true}, {0, false}, {0, false}},
		{{12, true}, {13, true}, {14, true}, {0, false}},
		{{22, true}, {23, true}, {24, true}, {0, false}},
	}
	for row := range 3 {
		for column := range 4 {
			height, present := grid.At(column, row)
			if height != want[row][column].height || present != want[row][column].present {
				t.Errorf("%d,%d: %v %v", column, row, height, present)
			}
		}
	}
	for _, at := range [][2]int{{-1, 0}, {0, -1}, {4, 0}, {0, 3}} {
		if height, present := grid.At(at[0], at[1]); height != 0 || present {
			t.Errorf("outside the grid at %v: %v %v", at, height, present)
		}
	}
}

// TestReadGridRefusesBadRequests checks each refusal of a grid request.
func TestReadGridRefusesBadRequests(t *testing.T) {
	raster := openSpec(t, gridSpec())
	good := gridRequest{West: 100.5, North: 49.5, Step: 1, Across: 2, Down: 2}
	with := func(edit func(*gridRequest)) gridRequest {
		request := good
		edit(&request)
		return request
	}
	cases := []struct {
		name    string
		request gridRequest
		limit   int64
		want    string
	}{
		{"no columns", with(func(r *gridRequest) { r.Across = 0 }), 1 << 20, "invalid source grid layout"},
		{"no rows", with(func(r *gridRequest) { r.Down = 0 }), 1 << 20, "invalid source grid layout"},
		{"NaN west", with(func(r *gridRequest) { r.West = math.NaN() }), 1 << 20, "invalid source grid layout"},
		{"infinite north", with(func(r *gridRequest) { r.North = math.Inf(-1) }), 1 << 20, "invalid source grid layout"},
		{"zero step", with(func(r *gridRequest) { r.Step = 0 }), 1 << 20, "invalid source grid layout"},
		{"negative step", with(func(r *gridRequest) { r.Step = -1 }), 1 << 20, "invalid source grid layout"},
		{"negative limit", good, -1, "invalid source grid byte budget"},
		{"one byte short", good, 4*9 - 1, "payload budget"},
		{"coordinates overflow", with(func(r *gridRequest) { r.West, r.Step, r.Across = 1.7e308, 1.7e308, 3 }), 1 << 20, "coordinates overflow"},
	}
	for _, each := range cases {
		if _, err := raster.ReadGrid(each.request, each.limit); err == nil || !strings.Contains(err.Error(), each.want) {
			t.Errorf("%s: %v, want %q", each.name, err, each.want)
		}
	}
	if _, err := raster.ReadGrid(good, 4*9); err != nil {
		t.Errorf("a grid that exactly fits was refused: %v", err)
	}
	if _, err := good.capacity(100, 0); err == nil {
		t.Error("a zero-byte sample was accepted")
	}
	geographic := openSpec(t, geoSpec(2))
	if _, err := geographic.ReadGrid(good, 1<<20); err == nil || !strings.Contains(err.Error(), "geographic world adapter") {
		t.Errorf("geographic source: %v", err)
	}
	unplaced := openSpec(t, newTiffSpec(tiffLittle, false, 5, 4, 4, 4, rampAt))
	if _, err := unplaced.ReadGrid(good, 1<<20); err == nil || !strings.Contains(err.Error(), "no projected placement") {
		t.Errorf("unplaced source: %v", err)
	}
}

// TestReadGridReportsAnUnreadableTile checks a corrupt source is an error.
func TestReadGridReportsAnUnreadableTile(t *testing.T) {
	path := gridSpec().write(t)
	raster, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer raster.Close()
	if err := os.Truncate(path, 8); err != nil {
		t.Fatal(err)
	}
	request := gridRequest{West: 100.5, North: 49.5, Step: 1, Across: 2, Down: 2}
	if _, err := raster.ReadGrid(request, 1<<20); err == nil {
		t.Error("a grid was read from a cut file")
	}
}

// TestWindowRefusesBadRequestsAndClosedSources checks the window guards, the
// size a window reports and a read from a closed or cut source.
func TestWindowRefusesBadRequestsAndClosedSources(t *testing.T) {
	path := gridSpec().write(t)
	raster, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	for name, call := range map[string]func() error{
		"one column": func() error { _, err := raster.Window(100, 50, 1, 2, 1); return err },
		"one row":    func() error { _, err := raster.Window(100, 50, 2, 1, 1); return err },
		"zero step":  func() error { _, err := raster.Window(100, 50, 2, 2, 0); return err },
		"NaN step":   func() error { _, err := raster.Window(100, 50, 2, 2, math.NaN()); return err },
		"budget":     func() error { _, err := raster.windowWithBudget(100, 50, 2, 2, 1, 31); return err },
	} {
		if err := call(); err == nil {
			t.Errorf("%s: no refusal", name)
		}
	}
	window, err := raster.windowWithBudget(100.5, 49.5, 3, 2, 1, 6*8)
	if err != nil {
		t.Fatal(err)
	}
	across, down, step := window.Size()
	if across != 3 || down != 2 || step != 1 || window.At(2, 1) != 12 {
		t.Errorf("window %d by %d, step %v, corner %v", across, down, step, window.At(2, 1))
	}
	geographic := openSpec(t, geoSpec(2))
	if _, err := geographic.Window(10, 20, 2, 2, 1); err == nil {
		t.Error("a geographic source gave a metre window")
	}
	if err := os.Truncate(path, 8); err != nil {
		t.Fatal(err)
	}
	raster.DiscardCache()
	if _, err := raster.Window(100.5, 49.5, 2, 2, 1); err == nil {
		t.Error("a window was cut from a cut file")
	}
	raster.Close()
	if _, err := raster.Window(100.5, 49.5, 2, 2, 1); err == nil || !strings.Contains(err.Error(), "closed") {
		t.Errorf("a window from a closed source: %v", err)
	}
}

// TestCoverageOfAnUnplacedSourceIsAbsent checks Coverage and Elevation on
// sources that cannot answer.
func TestCoverageOfAnUnplacedSourceIsAbsent(t *testing.T) {
	unplaced := openSpec(t, newTiffSpec(tiffLittle, false, 5, 4, 4, 4, rampAt))
	if _, placed := unplaced.Coverage(); placed {
		t.Error("an unplaced source reported coverage")
	}
	if _, _, err := unplaced.Elevation(1, 1); err == nil || !strings.Contains(err.Error(), "no projected placement") {
		t.Errorf("unplaced elevation: %v", err)
	}
	geographic := openSpec(t, geoSpec(2))
	if _, _, err := geographic.Elevation(11, 19); err == nil {
		t.Error("a geographic source answered in metres")
	}
	placed := openSpec(t, gridSpec())
	if cover, ok := placed.Coverage(); !ok || cover != (Coverage{West: 100, North: 50, East: 105, South: 46}) {
		t.Errorf("coverage %+v %v", cover, ok)
	}
	if _, _, err := placed.Elevation(math.NaN(), 49); err == nil {
		t.Error("a NaN location was accepted")
	}
}

// heightSpec is a five by four degree grid for the height index, 100 east and
// 50 north, point registration, heights column plus ten row.
func heightSpec() *tiffSpec {
	return newTiffSpec(tiffLittle, false, 5, 4, 4, 4, rampAt).degrees(2).place(100, 50, 1, 1)
}

// TestHeightIndexHoldsBlockExtrema checks the extrema of each two by two
// block against hand worked values, and the ranges returned over them.
func TestHeightIndexHoldsBlockExtrema(t *testing.T) {
	raster := openSpec(t, heightSpec())
	index, err := raster.BuildHeightIndex(context.Background(), 2, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	wantLow := []float32{0, 2, 4, 20, 22, 24}
	wantHigh := []float32{11, 13, 14, 31, 33, 34}
	for cell := range wantLow {
		if index.minimum[cell] != wantLow[cell] || index.maximum[cell] != wantHigh[cell] || !index.present[cell] {
			t.Errorf("cell %d: %v..%v present %v", cell, index.minimum[cell], index.maximum[cell], index.present[cell])
		}
	}
	cases := []struct {
		name                     string
		west, south, east, north float64
		low, high                float64
		present                  bool
	}{
		{"whole raster", 100, 47, 104, 50, 0, 34, true},
		{"top left with halo", 100, 49, 101, 50, 0, 33, true},
		{"far east sample", 104, 50, 104, 50, 2, 14, true},
		{"east of it", 200, 47, 201, 50, 0, 0, false},
		{"west of it", 10, 47, 20, 50, 0, 0, false},
		{"south of it", 100, 10, 104, 20, 0, 0, false},
		{"north of it", 100, 90, 104, 99, 0, 0, false},
	}
	for _, each := range cases {
		low, high, present, err := index.Range(each.west, each.south, each.east, each.north)
		if err != nil || present != each.present || low != each.low || high != each.high {
			t.Errorf("%s: %v..%v %v %v; want %v..%v %v", each.name, low, high, present, err, each.low, each.high, each.present)
		}
	}
	for name, args := range map[string][4]float64{
		"west after east":   {5, 0, 4, 1},
		"south after north": {0, 5, 1, 4},
		"NaN":               {math.NaN(), 0, 1, 1},
		"infinite":          {0, 0, math.Inf(1), 1},
	} {
		if _, _, _, err := index.Range(args[0], args[1], args[2], args[3]); err == nil {
			t.Errorf("%s: no refusal", name)
		}
	}
	var none *HeightIndex
	if _, _, _, err := none.Range(0, 0, 1, 1); err == nil {
		t.Error("a nil index answered")
	}
}

// TestHeightIndexSkipsMissingSamples checks that no-data and NaN samples add
// nothing to a block and that a block with none stays absent.
func TestHeightIndexSkipsMissingSamples(t *testing.T) {
	spec := newTiffSpec(tiffLittle, false, 5, 4, 4, 4, func(column, row int) float32 {
		switch {
		case column == 0 && row == 0:
			return -9999
		case column == 2 && row == 0:
			return float32(math.NaN())
		case column == 4 && row >= 2:
			return float32(math.NaN())
		}
		return rampAt(column, row)
	}).degrees(2).place(100, 50, 1, 1).ascii(tagNoData, "-9999")
	raster := openSpec(t, spec)
	index, err := raster.BuildHeightIndex(context.Background(), 2, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if index.minimum[0] != 1 || index.maximum[0] != 11 || index.minimum[1] != 3 || index.maximum[1] != 13 {
		t.Errorf("blocks 0 and 1: %v..%v, %v..%v", index.minimum[0], index.maximum[0], index.minimum[1], index.maximum[1])
	}
	if index.present[5] {
		t.Error("a block with no samples was present")
	}
	low, high, present, err := index.Range(100, 47, 104, 50)
	if err != nil || !present || low != 1 || high != 33 {
		t.Errorf("range %v..%v %v %v", low, high, present, err)
	}
}

// TestHeightIndexRefusesBadRequests checks each guard of BuildHeightIndex.
func TestHeightIndexRefusesBadRequests(t *testing.T) {
	raster := openSpec(t, heightSpec())
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	var noContext context.Context
	projected := openSpec(t, gridSpec())
	unplaced := openSpec(t, newTiffSpec(tiffLittle, false, 5, 4, 4, 4, rampAt).degrees(2))
	unregistered := openSpec(t, newTiffSpec(tiffLittle, false, 5, 4, 4, 4, rampAt).place(100, 50, 1, 1).
		geo([4]uint16{1024, 0, 1, 2}, [4]uint16{2048, 0, 1, 4326}, [4]uint16{2054, 0, 1, 9102}))
	closed := openSpec(t, heightSpec())
	closed.Close()
	var nilRaster *Raster
	cases := []struct {
		name   string
		raster *Raster
		ctx    context.Context
		block  int
		limit  int64
	}{
		{"no context", raster, noContext, 2, 1 << 20},
		{"nil raster", nilRaster, context.Background(), 2, 1 << 20},
		{"closed raster", closed, context.Background(), 2, 1 << 20},
		{"unplaced raster", unplaced, context.Background(), 2, 1 << 20},
		{"projected raster", projected, context.Background(), 2, 1 << 20},
		{"zero block", raster, context.Background(), 0, 1 << 20},
		{"negative limit", raster, context.Background(), 2, -1},
		{"unspecified registration", unregistered, context.Background(), 2, 1 << 20},
		{"too small a budget", raster, context.Background(), 2, 6*9 - 1},
	}
	for _, each := range cases {
		if _, err := each.raster.BuildHeightIndex(each.ctx, each.block, each.limit); err == nil {
			t.Errorf("%s: no refusal", each.name)
		}
	}
	if _, err := raster.BuildHeightIndex(context.Background(), 2, 6*9); err != nil {
		t.Errorf("a budget that exactly fits was refused: %v", err)
	}
	if _, err := raster.BuildHeightIndex(cancelled, 2, 1<<20); err != context.Canceled {
		t.Errorf("a cancelled context: %v", err)
	}
}

// TestHeightIndexReportsBadSamples checks a non-finite native sample and an
// unreadable tile are errors.
func TestHeightIndexReportsBadSamples(t *testing.T) {
	infinite := openSpec(t, newTiffSpec(tiffLittle, false, 5, 4, 4, 4, func(column, row int) float32 {
		if column == 3 && row == 2 {
			return float32(math.Inf(1))
		}
		return rampAt(column, row)
	}).degrees(2).place(100, 50, 1, 1))
	if _, err := infinite.BuildHeightIndex(context.Background(), 2, 1<<20); err == nil ||
		!strings.Contains(err.Error(), "non-finite") {
		t.Errorf("an infinite sample: %v", err)
	}
	path := heightSpec().write(t)
	cut, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer cut.Close()
	if err := os.Truncate(path, 8); err != nil {
		t.Fatal(err)
	}
	if _, err := cut.BuildHeightIndex(context.Background(), 2, 1<<20); err == nil {
		t.Error("an index was built from a cut file")
	}
}

// TestSampleIsAbsentOutsideTheRaster checks the raw sample accessor outside
// the lattice on each side.
func TestSampleIsAbsentOutsideTheRaster(t *testing.T) {
	raster := openSpec(t, gridSpec())
	for _, at := range [][2]int{{-1, 0}, {0, -1}, {5, 0}, {0, 4}} {
		if value, present, err := raster.sample(at[0], at[1]); value != 0 || present || err != nil {
			t.Errorf("sample %v: %v %v %v", at, value, present, err)
		}
	}
}

// TestSampleLocationRefusesWhatItCannotPlace checks the refusals of
// SampleLocation and the registration offsets it applies.
func TestSampleLocationRefusesWhatItCannotPlace(t *testing.T) {
	var none *Raster
	closed := openSpec(t, geoSpec(2))
	closed.Close()
	unplaced := openSpec(t, newTiffSpec(tiffLittle, false, 5, 4, 4, 4, rampAt).degrees(2))
	good := openSpec(t, geoSpec(2))
	for name, call := range map[string]func() error{
		"nil":          func() error { _, _, err := none.SampleLocation(0, 0); return err },
		"closed":       func() error { _, _, err := closed.SampleLocation(0, 0); return err },
		"unplaced":     func() error { _, _, err := unplaced.SampleLocation(0, 0); return err },
		"negative":     func() error { _, _, err := good.SampleLocation(-1, 0); return err },
		"past columns": func() error { _, _, err := good.SampleLocation(5, 0); return err },
		"past rows":    func() error { _, _, err := good.SampleLocation(0, 4); return err },
	} {
		if err := call(); err == nil {
			t.Errorf("%s: no refusal", name)
		}
	}
	if east, north, err := good.SampleLocation(1, 2); err != nil || east != 11 || north != 18 {
		t.Errorf("point registration: %v %v %v", east, north, err)
	}
	area := openSpec(t, geoSpec(1))
	if east, north, err := area.SampleLocation(1, 2); err != nil || east != 11.5 || north != 17.5 {
		t.Errorf("area registration: %v %v %v", east, north, err)
	}
}

// TestProjectedWindowsRefuseNonMetreUnits checks that a source declared in
// feet cannot be read through the metre-based calls.
func TestProjectedWindowsRefuseNonMetreUnits(t *testing.T) {
	feet := openSpec(t, rampSpec().geo([4]uint16{1024, 0, 1, 1}, [4]uint16{3076, 0, 1, 9002}))
	if _, _, err := feet.Elevation(100.5, 49.5); err == nil || !strings.Contains(err.Error(), "not declared metres") {
		t.Errorf("a source in feet: %v", err)
	}
	metres := openSpec(t, rampSpec().geo([4]uint16{1024, 0, 1, 1}, [4]uint16{3076, 0, 1, 9001}))
	if got, present, err := metres.Elevation(100.5, 49.5); err != nil || !present || got != 0 {
		t.Errorf("a source in metres: %v %v %v", got, present, err)
	}
}

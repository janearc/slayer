package terrain

import (
	"encoding/binary"
	"strings"
	"testing"
)

func referenceDirectory(values []uint16) map[int]entry {
	raw := make([]byte, 2*len(values))
	for i, v := range values {
		binary.LittleEndian.PutUint16(raw[2*i:], v)
	}
	return map[int]entry{34735: {kind: 3, count: len(values), raw: raw}}
}

func TestTerrainReferenceReadsDeclaredCodesAndRefusesMalformedKeys(t *testing.T) {
	keys := []uint16{1, 1, 0, 5, 1024, 0, 1, 2, 1025, 0, 1, 2,
		2048, 0, 1, 4269, 2054, 0, 1, 9102, 4099, 0, 1, 9001}
	r := Raster{order: binary.LittleEndian}
	if err := r.readReference(referenceDirectory(keys)); err != nil {
		t.Fatal(err)
	}
	if r.Reference() != (SourceReference{Model: 2, RasterType: 2,
		GeographicCRS: 4269, AngularUnits: 9102, VerticalUnits: 9001}) {
		t.Fatalf("wrong reference %+v", r.Reference())
	}
	for _, bad := range [][]uint16{
		{1, 1, 0, 2, 1024, 0, 1, 2},
		{1, 1, 0, 2, 1024, 0, 1, 2, 1024, 0, 1, 1},
		{1, 1, 0, 1, 1024, 0, 2, 2},
		{1, 2, 0, 0},
	} {
		r := Raster{order: binary.LittleEndian}
		if err := r.readReference(referenceDirectory(bad)); err == nil {
			t.Fatal("malformed GeoKey directory accepted", bad)
		}
	}
}

func TestTerrainSampleRegistrationHasNoHalfCellShift(t *testing.T) {
	r := Raster{placed: true, width: 3, height: 3, west: 100, north: 200, spacing: 10}
	for _, row := range []struct {
		kind int
		x, y float64
	}{
		{1, 105, 195}, {2, 100, 200},
	} {
		r.reference.RasterType = row.kind
		x, y, err := r.SampleLocation(0, 0)
		if err != nil || x != row.x || y != row.y {
			t.Fatalf("registration %d: %g,%g,%v", row.kind, x, y, err)
		}
		g := PatchGrid{reference: r.reference, west: 100, north: 200, spacing: 10,
			columns: []int{0, 2}, rows: []int{0, 2},
			heights: []float64{1, 1, 1, 1}, present: []bool{true, true, true, true}}
		x, y, _, ok := g.At(0, 0)
		if !ok || x != row.x || y != row.y {
			t.Fatal("grid sample registration differs")
		}
	}
}

func TestTerrainGeographicSourceCannotEnterLegacyMetreWindow(t *testing.T) {
	r := Raster{placed: true, reference: SourceReference{Model: 2}}
	_, err := r.WindowWithBudget(0, 0, 2, 2, 1, 32)
	if err == nil || !strings.Contains(err.Error(), "geographic") {
		t.Fatal(err)
	}
	_, err = r.ReadGrid(GridRequest{Across: 2, Down: 2, Step: 1}, 64)
	if err == nil || !strings.Contains(err.Error(), "geographic") {
		t.Fatal(err)
	}
	_, _, err = r.Elevation(0, 0)
	if err == nil || !strings.Contains(err.Error(), "geographic") {
		t.Fatal(err)
	}
}

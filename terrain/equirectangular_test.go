package terrain

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testRadius is picked: a sphere of a thousand metres keeps the numbers a
// hand can check.
const testRadius = 1000.0

// testTolerance is picked: far above float rounding in these sums and far
// below the smallest difference any test looks for.
const testTolerance = 1e-9

// the real-file test reads the map from the variable that names it, or,
// without it, from the shelf under the shelf's own name for it
const (
	magellanVariable = "SLAYER_MAGELLAN_TOPOGRAPHY"
	shelfVariable    = "SLAYER_SHELF"
	magellanOnShelf  = "usgs-magellan-venus-topography.tif"
)

// the two byte orders a tiff may be written in
var tiffBig tiffOrder = binary.BigEndian

// signedSampleBytes are whole numbers as a tiff writes 16-bit samples.
func signedSampleBytes(order tiffOrder, values ...int16) []byte {
	var raw []byte
	for _, value := range values {
		raw = order.AppendUint16(raw, uint16(value))
	}
	return raw
}

// signedSpec starts a spec of one band of signed 16-bit samples, whole,
// uncompressed. A tileW of zero makes strips of tileH rows, the last strip
// as short as the raster leaves it; anything else makes padded tiles.
func signedSpec(order tiffOrder, width, height, tileW, tileH int,
	at func(column, row int) int16,
) *tiffSpec {
	spec := &tiffSpec{order: order, tags: map[uint16]tiffTag{},
		width: width, height: height, tileW: tileW, tileH: tileH}
	striped := tileW == 0
	if striped {
		tileW = width
	}
	across, down := (width+tileW-1)/tileW, (height+tileH-1)/tileH
	for tileRow := range down {
		for tileColumn := range across {
			var values []int16
			for y := range tileH {
				row := tileRow*tileH + y
				if striped && row >= height {
					break
				}
				for x := range tileW {
					column := tileColumn*tileW + x
					if column < width && row < height {
						values = append(values, at(column, row))
					} else {
						values = append(values, 0)
					}
				}
			}
			spec.tiles = append(spec.tiles, signedSampleBytes(order, values...))
		}
	}
	spec.shorts(tagWidth, uint16(width)).shorts(tagHeight, uint16(height))
	spec.shorts(tagBits, signedBits).shorts(tagCompression, uncompressed)
	spec.shorts(tagSamples, 1).shorts(tagFormat, signedFormat)
	if striped {
		return spec.shorts(tagRowsPerStrip, uint16(tileH))
	}
	return spec.shorts(tagTileWidth, uint16(tileW)).shorts(tagTileHeight, uint16(tileH))
}

// signedRamp is the height column plus ten times row, as a signed whole.
func signedRamp(column, row int) int16 { return int16(column + 10*row) }

// sphereRows are the GeoKeys of an equirectangular file on a sphere, in the
// order the Magellan file lists them, for a transform and linear unit code.
// The doubles they point at are set by sphereSpec.
func sphereRows(transform, units uint16) [][4]uint16 {
	return [][4]uint16{{1024, 0, 1, 1}, {1025, 0, 1, 1}, {2048, 0, 1, 32767},
		{2054, 0, 1, 9102}, {2057, geoKeyDoubles, 1, 0}, {2058, geoKeyDoubles, 1, 1},
		{3072, 0, 1, 32767}, {3075, 0, 1, transform}, {3076, 0, 1, units},
		{3078, geoKeyDoubles, 1, 2}, {3082, geoKeyDoubles, 1, 3},
		{3083, geoKeyDoubles, 1, 4}, {3088, geoKeyDoubles, 1, 5},
		{3089, geoKeyDoubles, 1, 6}}
}

// sphereSpec is a whole-circle equirectangular map of a sphere of testRadius,
// signed 16-bit, in strips of one row, centred on longitude zero and the
// equator, as the Magellan file is. At width 8 and height 4 the sample
// centres are at longitudes 45 apart from -157.5 and latitudes 45 apart
// from 67.5.
func sphereSpec(order tiffOrder, width, height int, parallel float64,
	at func(column, row int) int16,
) *tiffSpec {
	circle := fullTurn * testRadius * math.Cos(radians(parallel))
	scale := circle / float64(width)
	spec := signedSpec(order, width, height, 0, 1, at)
	spec.place(-circle/2, scale*float64(height)/2, scale, scale)
	spec.doubles(geoKeyDoubles, testRadius, testRadius, parallel, 0, 0, 0, 0)
	return spec.geo(sphereRows(transformEquirectangular, metreUnits)...)
}

// centreOf is the longitude and latitude of the centre of a sample of the
// eight by four sphere.
func centreOf(column, row int) (longitude, latitude float64) {
	const step = 45.0 // picked: 360 degrees over eight columns, 180 over four rows
	return -180 + step*(float64(column)+0.5), 90 - step*(float64(row)+0.5)
}

// closeTo says two heights agree to the test tolerance.
func closeTo(got, want float64) bool { return math.Abs(got-want) < testTolerance }

// TestSignedSamplesAreReadAsWritten reads striped and tiled files in both
// byte orders, with the extremes of the 16-bit range among the samples.
func TestSignedSamplesAreReadAsWritten(t *testing.T) {
	at := func(column, row int) int16 {
		switch {
		case column == 0 && row == 0:
			return math.MaxInt16
		case column == 1 && row == 0:
			return math.MinInt16 + 1
		}
		return int16(100*column - 700*row)
	}
	for _, order := range []tiffOrder{tiffLittle, tiffBig} {
		for name, spec := range map[string]*tiffSpec{
			"one row strips":                   signedSpec(order, 5, 6, 0, 1, at),
			"three row strips, the last short": signedSpec(order, 5, 7, 0, 3, at),
			"padded tiles":                     signedSpec(order, 5, 6, 4, 4, at),
		} {
			raster := openSpec(t, spec)
			for row := range spec.height {
				for column := range 5 {
					got, has, err := raster.sample(column, row)
					if err != nil || !has || got != float64(at(column, row)) {
						t.Fatalf("%s %v %d,%d: %v %v %v", name, order, column, row, got, has, err)
					}
				}
			}
		}
	}
}

// TestOtherSampleKindsAreRefusedByName is the anti-test: each width and
// format that is not float32 or signed 16-bit is refused in words that say
// which it was.
func TestOtherSampleKindsAreRefusedByName(t *testing.T) {
	cases := []struct {
		name string
		edit func(*tiffSpec)
		want string
	}{
		{"unsigned 16-bit", func(s *tiffSpec) { s.shorts(tagFormat, unsignedFormat) },
			"not float32 or signed 16-bit samples: 16-bit unsigned integer"},
		{"signed 8-bit", func(s *tiffSpec) { s.shorts(tagBits, 8) },
			"8-bit signed integer"},
		{"signed 32-bit", func(s *tiffSpec) { s.shorts(tagBits, 32) },
			"32-bit signed integer"},
		{"no format", func(s *tiffSpec) { delete(s.tags, tagFormat) },
			"unstated format"},
		{"predictor on whole numbers", func(s *tiffSpec) { s.shorts(tagPredictor, 3) },
			"predictor 3 on signed 16-bit samples"},
	}
	for _, each := range cases {
		spec := signedSpec(tiffLittle, 4, 4, 0, 2, signedRamp)
		each.edit(spec)
		_, err := Open(spec.write(t))
		if err == nil || !strings.Contains(err.Error(), each.want) {
			t.Errorf("%s: %v, want %q", each.name, err, each.want)
		}
	}
}

// TestSignedNoDataIsAbsent checks that the no-data value, and only it, is
// absent, as it is for floats.
func TestSignedNoDataIsAbsent(t *testing.T) {
	const none = -32768 // defined: the GDAL no-data value the Magellan file declares
	at := func(column, row int) int16 {
		if column == 2 && row == 1 {
			return none
		}
		return signedRamp(column, row) - 100
	}
	raster := openSpec(t, signedSpec(tiffLittle, 5, 4, 0, 1, at).ascii(tagNoData, "-32768"))
	if got, has, err := raster.sample(2, 1); err != nil || has || got != 0 {
		t.Errorf("the no-data sample: %v %v %v", got, has, err)
	}
	if got, has, err := raster.sample(3, 1); err != nil || !has || got != -87 {
		t.Errorf("its neighbour, an ordinary negative: %v %v %v", got, has, err)
	}
}

// TestSignedSamplesReadThroughLzw checks a compressed tile of whole numbers,
// and, as its anti-test, one that lacks its end code.
func TestSignedSamplesReadThroughLzw(t *testing.T) {
	values := make([]int16, 16)
	for index := range values {
		values[index] = int16(index - 8)
	}
	for _, withEnd := range []bool{true, false} {
		spec := signedSpec(tiffLittle, 4, 4, 4, 4, signedRamp).shorts(tagCompression, lzwed)
		spec.tiles = [][]byte{lzwLiterals(signedSampleBytes(tiffLittle, values...), withEnd)}
		raster := openSpec(t, spec)
		got, _, err := raster.sample(3, 3)
		if withEnd && (err != nil || got != 7) {
			t.Errorf("whole: %v %v", got, err)
		}
		if !withEnd && (err == nil || !strings.Contains(err.Error(), "missing compressed tile end")) {
			t.Errorf("no end code: %v", err)
		}
	}
}

// TestANegativeTiepointPlacesTheWindow puts a raster west and south of the
// origin and reads it back; the anti-test is the places just outside it.
func TestANegativeTiepointPlacesTheWindow(t *testing.T) {
	raster := openSpec(t, signedSpec(tiffLittle, 5, 4, 4, 4, signedRamp).place(-100, -50, 1, 1))
	coverage, placed := raster.Coverage()
	if !placed || coverage != (Coverage{West: -100, North: -50, East: -95, South: -54}) {
		t.Fatalf("coverage %+v %v", coverage, placed)
	}
	for _, each := range []struct{ east, north, want float64 }{
		{-99.5, -50.5, 0}, {-95.5, -53.5, 34}, {-97.5, -51.5, 12},
	} {
		got, has, err := raster.Elevation(each.east, each.north)
		if err != nil || !has || got != each.want {
			t.Errorf("%v,%v: %v %v %v", each.east, each.north, got, has, err)
		}
	}
	for _, each := range [][2]float64{{-100.5, -50.5}, {-99.5, -49.5}, {-94.5, -50.5}, {-99.5, -54.5}} {
		if got, has, err := raster.Elevation(each[0], each[1]); err != nil || has || got != 0 {
			t.Errorf("outside %v: %v %v %v", each, got, has, err)
		}
	}
}

// TestEquirectangularSphereIsRecognised reads the radius of a sphere file;
// the anti-test is each way a file stops being one.
func TestEquirectangularSphereIsRecognised(t *testing.T) {
	for _, order := range []tiffOrder{tiffLittle, tiffBig} {
		raster := openSpec(t, sphereSpec(order, 8, 4, 0, signedRamp))
		if radius, ok := raster.EquirectangularSphere(); !ok || radius != testRadius {
			t.Errorf("%v: %v %v", order, radius, ok)
		}
	}
	cases := []struct {
		name string
		edit func(*tiffSpec)
	}{
		{"another transform", func(s *tiffSpec) { s.geo(sphereRows(11, metreUnits)...) }},
		{"feet", func(s *tiffSpec) { s.geo(sphereRows(transformEquirectangular, 9002)...) }},
		{"unequal axes", func(s *tiffSpec) {
			s.doubles(geoKeyDoubles, testRadius, testRadius+1, 0, 0, 0, 0, 0)
		}},
		{"no axes", func(s *tiffSpec) { s.doubles(geoKeyDoubles, 0, 0, 0, 0, 0, 0, 0) }},
		{"parallel at a pole", func(s *tiffSpec) {
			s.doubles(geoKeyDoubles, testRadius, testRadius, 90, 0, 0, 0, 0)
		}},
		{"geographic", func(s *tiffSpec) { s.degrees(1) }},
		{"no geokeys", func(s *tiffSpec) { delete(s.tags, geoKeyDirectory) }},
		{"not placed", func(s *tiffSpec) { delete(s.tags, tagScale); delete(s.tags, tagTiepoint) }},
	}
	for _, each := range cases {
		spec := sphereSpec(tiffLittle, 8, 4, 0, signedRamp)
		each.edit(spec)
		raster := openSpec(t, spec)
		if radius, ok := raster.EquirectangularSphere(); ok || radius != 0 {
			t.Errorf("%s: said %v %v", each.name, radius, ok)
		}
		if _, _, err := raster.EquirectangularElevation(0, 0); err == nil {
			t.Errorf("%s: answered a height", each.name)
		}
		if _, _, err := raster.GroundSpacing(0); err == nil {
			t.Errorf("%s: answered a spacing", each.name)
		}
	}
}

// TestSphereKeysThatPointNowhereAreRefused checks the GeoKeys held as
// doubles: one not in the doubles tag, and one set as a short.
func TestSphereKeysThatPointNowhereAreRefused(t *testing.T) {
	for name, row := range map[string][4]uint16{
		"past the doubles": {2057, geoKeyDoubles, 1, 9},
		"as a short":       {2057, 0, 1, 6},
		"two values":       {2057, geoKeyDoubles, 2, 0},
	} {
		spec := sphereSpec(tiffLittle, 8, 4, 0, signedRamp)
		var rows [][4]uint16
		for _, each := range sphereRows(transformEquirectangular, metreUnits) {
			if each[0] != keySemiMajor {
				rows = append(rows, each)
			}
		}
		spec.geo(append(rows, row)...)
		if _, err := Open(spec.write(t)); err == nil || !strings.Contains(err.Error(), "GeoKey 2057") {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// TestEquirectangularHeightAtACentreIsThatSample reads every sample back by
// the latitude and longitude of its centre.
func TestEquirectangularHeightAtACentreIsThatSample(t *testing.T) {
	for _, order := range []tiffOrder{tiffLittle, tiffBig} {
		raster := openSpec(t, sphereSpec(order, 8, 4, 0, signedRamp))
		for row := range 4 {
			for column := range 8 {
				longitude, latitude := centreOf(column, row)
				got, has, err := raster.EquirectangularElevation(longitude, latitude)
				expected := float64(signedRamp(column, row))
				if err != nil || !has || !closeTo(got, expected) {
					t.Fatalf("%d,%d: %v %v %v", column, row, got, has, err)
				}
			}
		}
	}
}

// TestEquirectangularHeightBetweenCentresIsBlended checks hand worked
// weights on the ramp; the anti-test is that none of them is a centre's own.
func TestEquirectangularHeightBetweenCentresIsBlended(t *testing.T) {
	raster := openSpec(t, sphereSpec(tiffLittle, 8, 4, 0, signedRamp))
	longitude, latitude := centreOf(2, 1)
	const step = 45.0 // picked: the spacing of centres in degrees
	for name, each := range map[string]struct{ east, south, want float64 }{
		"midway east":       {step / 2, 0, 12.5},
		"a quarter east":    {step / 4, 0, 12.25},
		"midway south":      {0, step / 2, 17},
		"midway both ways":  {step / 2, step / 2, 17.5},
		"a quarter of each": {step / 4, step / 4, 14.75},
	} {
		got, has, err := raster.EquirectangularElevation(longitude+each.east, latitude-each.south)
		if err != nil || !has || !closeTo(got, each.want) || closeTo(got, 12) {
			t.Errorf("%s: %v %v %v, want %v", name, got, has, err, each.want)
		}
	}
}

// TestEquirectangularLongitudeWraps checks that places a turn apart are one
// place; the anti-test is a degree away, which is not.
func TestEquirectangularLongitudeWraps(t *testing.T) {
	raster := openSpec(t, sphereSpec(tiffLittle, 8, 4, 0, signedRamp))
	_, latitude := centreOf(0, 1)
	base, has, err := raster.EquirectangularElevation(-57, latitude)
	if err != nil || !has {
		t.Fatal(base, has, err)
	}
	for _, longitude := range []float64{303, -417, 663, -57 + 3600} {
		got, has, err := raster.EquirectangularElevation(longitude, latitude)
		if err != nil || !has || !closeTo(got, base) {
			t.Errorf("%v: %v %v %v, want %v", longitude, got, has, err, base)
		}
	}
	other, _, _ := raster.EquirectangularElevation(-58, latitude)
	if closeTo(other, base) {
		t.Errorf("a degree away is the same height, %v", other)
	}
}

// TestEquirectangularSeamWrapsAcrossTheEdge checks neighbours across the
// east and west edges of a whole-circle file; the anti-test is a file that
// is half a circle, which has no neighbour past its edge.
func TestEquirectangularSeamWrapsAcrossTheEdge(t *testing.T) {
	raster := openSpec(t, sphereSpec(tiffLittle, 8, 4, 0, signedRamp))
	_, latitude := centreOf(0, 1)
	const step = 45.0 // picked: the spacing of centres in degrees
	for _, each := range []struct{ longitude, want float64 }{
		{180, 13.5}, {-180, 13.5}, {157.5 + step/4, 10 + 7*0.75}, {-157.5 - step/4, 10 + 7*0.25},
	} {
		got, has, err := raster.EquirectangularElevation(each.longitude, latitude)
		if err != nil || !has || !closeTo(got, each.want) {
			t.Errorf("%v: %v %v %v, want %v", each.longitude, got, has, err, each.want)
		}
	}
	circle := fullTurn * testRadius
	half := sphereSpec(tiffLittle, 8, 4, 0, signedRamp).
		place(-circle/4, circle/8, circle/16, circle/16)
	partial := openSpec(t, half)
	if got, has, err := partial.EquirectangularElevation(78.75, 11.25); err != nil || !has || !closeTo(got, 17) {
		t.Errorf("the last centre: %v %v %v", got, has, err)
	}
	for _, longitude := range []float64{88, 100, 180, -88} {
		if got, has, err := partial.EquirectangularElevation(longitude, 11.25); err != nil || has || got != 0 {
			t.Errorf("beyond a half circle at %v: %v %v %v", longitude, got, has, err)
		}
	}
}

// TestEquirectangularHoldsToTheEdgeRowTowardAPole checks the pole rule: the
// nearest row's height, not a row beyond it. The anti-tests are a latitude
// that is not on the sphere, and a point past the edge of a file that stops
// short of the pole.
func TestEquirectangularHoldsToTheEdgeRowTowardAPole(t *testing.T) {
	raster := openSpec(t, sphereSpec(tiffLittle, 8, 4, 0, signedRamp))
	longitude, _ := centreOf(3, 0)
	for _, each := range []struct{ latitude, want float64 }{
		{90, 3}, {80, 3}, {-90, 33}, {-80, 33}, {45, 8},
	} {
		got, has, err := raster.EquirectangularElevation(longitude, each.latitude)
		if err != nil || !has || !closeTo(got, each.want) {
			t.Errorf("%v: %v %v %v, want %v", each.latitude, got, has, err, each.want)
		}
	}
	for _, latitude := range []float64{90.5, -91, math.NaN(), math.Inf(1)} {
		if _, _, err := raster.EquirectangularElevation(longitude, latitude); err == nil {
			t.Errorf("latitude %v answered", latitude)
		}
	}
	circle := fullTurn * testRadius
	band := openSpec(t, sphereSpec(tiffLittle, 8, 4, 0, signedRamp).
		place(-circle/4, circle/8, circle/16, circle/16))
	if got, has, err := band.EquirectangularElevation(-70, 40); err != nil || !has ||
		!closeTo(got, 20.0/22.5-cellCentre) {
		t.Errorf("held within the band's edge: %v %v %v", got, has, err)
	}
	if got, has, err := band.EquirectangularElevation(-70, 60); err != nil || has || got != 0 {
		t.Errorf("north of the band: %v %v %v", got, has, err)
	}
}

// TestEquirectangularNoDataMakesTheHeightAbsent checks that a no-data sample
// among the four makes the height absent, in the file and across its seam;
// the anti-test is places whose four samples are all heights.
func TestEquirectangularNoDataMakesTheHeightAbsent(t *testing.T) {
	const none = -32768 // defined: the GDAL no-data value the Magellan file declares
	at := func(column, row int) int16 {
		if (column == 2 && row == 1) || (column == 0 && row == 2) {
			return none
		}
		return signedRamp(column, row)
	}
	raster := openSpec(t, sphereSpec(tiffLittle, 8, 4, 0, at).ascii(tagNoData, "-32768"))
	const step = 45.0 // picked: the spacing of centres in degrees
	for name, each := range map[string]struct {
		column, row int
		east, south float64
		present     bool
	}{
		"the hole itself":           {2, 1, 0, 0, false},
		"between the hole and next": {2, 1, step / 2, 0, false},
		"a corner of the hole":      {1, 0, step / 2, step / 2, false},
		"beside it, by the rule":    {1, 1, 0, 0, false},
		"its neighbour clear of it": {3, 1, 0, 0, true},
		"far from it":               {5, 3, step / 4, step / 4, true},
		"across the seam from one":  {7, 2, step / 2, 0, false},
		"the seam, clear of it":     {7, 0, step / 2, 0, true},
	} {
		longitude, latitude := centreOf(each.column, each.row)
		got, has, err := raster.EquirectangularElevation(longitude+each.east, latitude-each.south)
		if err != nil || has != each.present || (!has && got != 0) {
			t.Errorf("%s: %v %v %v", name, got, has, err)
		}
	}
}

// TestEquirectangularRefusesWhatItCannotAnswer checks a raster of one row,
// and a longitude that is not a number.
func TestEquirectangularRefusesWhatItCannotAnswer(t *testing.T) {
	flat := openSpec(t, sphereSpec(tiffLittle, 8, 1, 0, signedRamp))
	if _, _, err := flat.EquirectangularElevation(0, 0); err == nil ||
		!strings.Contains(err.Error(), "two samples per axis") {
		t.Errorf("one row: %v", err)
	}
	raster := openSpec(t, sphereSpec(tiffLittle, 8, 4, 0, signedRamp))
	if _, _, err := raster.EquirectangularElevation(math.NaN(), 0); err == nil {
		t.Error("an unknown longitude answered")
	}
	if _, has, err := raster.EquirectangularElevation(0, 0); err != nil || !has {
		t.Errorf("a plain place: %v %v", has, err)
	}
}

// TestEquirectangularReportsAnUnreadableTile checks a read failure is an
// error, not an absent height.
func TestEquirectangularReportsAnUnreadableTile(t *testing.T) {
	spec := sphereSpec(tiffLittle, 8, 4, 0, signedRamp)
	spec.offsets = []uint64{1 << 20, 1 << 20, 1 << 20, 1 << 20}
	raster := openSpec(t, spec)
	if _, has, err := raster.EquirectangularElevation(0, 0); err == nil || has {
		t.Errorf("an unreadable tile: %v %v", has, err)
	}
}

// TestGroundSpacingShrinksEastwardOnly checks both spacings at the equator,
// at 60 degrees and at a pole, on a map with its parallel at the equator and
// on one with it at 60; the anti-test is a latitude beyond a pole.
func TestGroundSpacingShrinksEastwardOnly(t *testing.T) {
	for _, parallel := range []float64{0, 60} {
		raster := openSpec(t, sphereSpec(tiffLittle, 8, 4, parallel, signedRamp))
		_, _, scale := raster.Corner()
		ratio := func(latitude float64) float64 {
			return math.Cos(radians(latitude)) / math.Cos(radians(parallel))
		}
		for _, latitude := range []float64{0, 60, 90, -30} {
			east, south, err := raster.GroundSpacing(latitude)
			if err != nil || !closeTo(east, scale*ratio(latitude)) || south != scale {
				t.Errorf("parallel %v latitude %v: %v %v %v", parallel, latitude, east, south, err)
			}
		}
		if _, _, err := raster.GroundSpacing(90.5); err == nil {
			t.Error("beyond a pole answered")
		}
	}
}

// TestAGeographicFileKeepsItsOwnPath checks a signed geographic file is
// answered by GeographicElevation as a float one is, and is not a sphere.
func TestAGeographicFileKeepsItsOwnPath(t *testing.T) {
	signed := openSpec(t, signedSpec(tiffLittle, 5, 4, 4, 4, signedRamp).degrees(2).place(10, 20, 1, 1))
	floats := openSpec(t, geoSpec(2))
	for _, each := range [][2]float64{{10.5, 19.5}, {12, 18}, {14, 20}} {
		a, hasA, errA := signed.GeographicElevation(each[0], each[1])
		b, hasB, errB := floats.GeographicElevation(each[0], each[1])
		if errA != nil || errB != nil || !hasA || !hasB || a != b {
			t.Errorf("%v: %v %v %v against %v %v %v", each, a, hasA, errA, b, hasB, errB)
		}
	}
	if _, ok := signed.EquirectangularSphere(); ok {
		t.Error("a geographic file said it was a sphere map")
	}
	sphere := openSpec(t, sphereSpec(tiffLittle, 8, 4, 0, signedRamp))
	if _, _, err := sphere.GeographicElevation(0, 0); err == nil {
		t.Error("a sphere map took the geographic path")
	}
}

// TestMagellanVenusTopographyTheFileSays reads the real file, when it is
// there: its size, its sphere, and a height at Venera 13's landing place by
// both longitudes that name it. The height is logged, not expected.
func TestMagellanVenusTopographyTheFileSays(t *testing.T) {
	path := os.Getenv(magellanVariable)
	if path == "" && os.Getenv(shelfVariable) != "" {
		path = filepath.Join(os.Getenv(shelfVariable), magellanOnShelf)
	}
	if path == "" {
		t.Skipf("set %s, or %s to a shelf holding %s, to read the Magellan topography.", magellanVariable, shelfVariable, magellanOnShelf)
	}
	if _, err := os.Stat(path); err != nil {
		t.Skipf("the Magellan topography file is not at %s; set %s to read it.", path, magellanVariable)
	}
	raster, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer raster.Close()
	if width, height := raster.Size(); width != 8192 || height != 4096 {
		t.Errorf("size %d by %d", width, height)
	}
	if radius, ok := raster.EquirectangularSphere(); !ok || radius != 6051000 {
		t.Errorf("sphere %v %v", radius, ok)
	}
	const venera = -7.5 // picked: Venera 13 landed at about 7.5 degrees south
	var heights []float64
	for _, longitude := range []float64{303, -57} {
		height, has, err := raster.EquirectangularElevation(longitude, venera)
		if err != nil || !has || math.IsNaN(height) || height < -3000 || height > 12000 {
			t.Fatalf("longitude %v: %v %v %v", longitude, height, has, err)
		}
		t.Logf("height at latitude %v, longitude %v: %v", venera, longitude, height)
		heights = append(heights, height)
	}
	if !closeTo(heights[0], heights[1]) {
		t.Errorf("303 and -57 differ: %v", heights)
	}
}

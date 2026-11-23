package terrain

import (
	"fmt"
	"math"
)

// the numbers an equirectangular file is read with
const (
	transformEquirectangular = 17          // defined: CT_Equirectangular in GeoTIFF1.1
	metreUnits               = 9001        // defined: EPSG linear unit code for the metre
	halfTurn                 = 180         // defined: degrees in half a turn
	quarterTurn              = 90          // defined: degrees from the equator to a pole
	fullTurn                 = 2 * math.Pi // defined: radians in a whole turn
	areaCells                = 1           // defined: GeoTIFF RasterPixelIsArea
	pointCells               = 2           // defined: GeoTIFF RasterPixelIsPoint
	cellCentre               = 0.5         // defined: an area sample's centre is half a cell in
	neighbourCount           = 2           // defined: bilinear filling reads two samples per axis
	edgeSlack                = 1e-6        // picked: a millionth of a sample, above float rounding at a pole's edge
)

// sphereFrame is an equirectangular file's projection resolved on a sphere:
// the sphere's radius in metres, the cosine of the standard parallel, the
// false easting and northing in metres, the centre longitude and latitude in
// degrees, and the offset of a sample's centre from its corner in cells.
type sphereFrame struct {
	radius, cosParallel   float64
	falseEast, falseNorth float64
	centreLongitude       float64
	centreLatitude        float64
	centreOffset          float64
}

// lattice is a place among the sample centres: the column to its left, how
// far it is along from that column to the next, and its row as a fraction.
// The left column is -1 for a place before the first centre of a file that
// wraps round; filled brings it onto the file.
type lattice struct {
	left        int
	across, row float64
}

// equirectangular resolves the one projection this package resolves:
// equirectangular, in metres, on a sphere, from the file's own keys, in a
// placed raster whose pixels are areas or points. A file that is anything
// else is not resolved and reports false.
func (raster *Raster) equirectangular() (sphereFrame, bool) {
	if raster == nil || raster.closed || !raster.placed {
		return sphereFrame{}, false
	}
	ref, keys := raster.reference, raster.sphere
	if ref.Model != 1 || ref.Transform != transformEquirectangular ||
		ref.LinearUnits != metreUnits ||
		(ref.RasterType != areaCells && ref.RasterType != pointCells) ||
		!(keys.semiMajor > 0) || keys.semiMajor != keys.semiMinor ||
		!finiteCoordinate(keys.semiMajor) || math.Abs(keys.parallel) >= quarterTurn {
		return sphereFrame{}, false
	}
	frame := sphereFrame{radius: keys.semiMajor,
		cosParallel: math.Cos(radians(keys.parallel)),
		falseEast:   keys.falseEast, falseNorth: keys.falseNorth,
		centreLongitude: keys.centreLongitude,
		centreLatitude:  keys.centreLatitude}
	if ref.RasterType == areaCells {
		frame.centreOffset = cellCentre
	}
	return frame, true
}

// radians is an angle in degrees as radians.
func radians(degrees float64) float64 { return degrees * math.Pi / halfTurn }

// EquirectangularSphere says whether the file is an equirectangular map of a
// sphere, in metres, by its own GeoKeys, and if so the sphere's radius in
// metres. It names no body and no datum: the radius is the file's own
// semi-major axis, which it declares equal to its semi-minor axis. A file
// that is any other projection, or has unequal axes, reports false.
func (raster *Raster) EquirectangularSphere() (radius float64, ok bool) {
	frame, ok := raster.equirectangular()
	return frame.radius, ok
}

// EquirectangularElevation interpolates an equirectangular file by
// planetocentric latitude and longitude, in degrees, bilinearly between the
// centres of the four nearest samples. The longitude comes first, as in
// GeographicElevation, so a caller treats the two alike.
//
// Longitude is east and may be in any range; it is wrapped onto the circle,
// so 303 and -57 are one place. Where the file spans the whole circle, a
// point within half a sample of its east or west edge takes its neighbours
// across the seam. Where it does not, a point beyond the first or last
// column centre is absent.
//
// Poleward of the first or last row's centre there is no neighbour. The
// point is held to that row and filled along it only, so the height at a
// pole is the nearest row's. A point beyond the file's north or south edge
// is absent. A latitude outside minus 90 to 90 is an error.
//
// If any of the four samples is no-data the height is absent, never zero,
// even where that sample's weight is nil. The height is the file's own
// samples, taken as metres above the sphere the file names; USGS's label
// says so and was not read here.
func (raster *Raster) EquirectangularElevation(longitude, latitude float64) (float64, bool, error) {
	frame, ok := raster.equirectangular()
	if !ok {
		return 0, false, fmt.Errorf("terrain: not an equirectangular source on a sphere")
	}
	if !finiteCoordinate(longitude) || !finiteCoordinate(latitude) ||
		math.Abs(latitude) > quarterTurn {
		return 0, false, fmt.Errorf("terrain: invalid latitude or longitude")
	}
	if raster.width < neighbourCount || raster.height < neighbourCount {
		return 0, false, fmt.Errorf("terrain: interpolation needs two samples per axis")
	}
	place, inside := frame.locate(raster, longitude, latitude)
	if !inside {
		return 0, false, nil
	}
	return raster.filled(place)
}

// circle is the length in metres of the circle of the standard parallel.
func (frame sphereFrame) circle() float64 {
	return fullTurn * frame.radius * frame.cosParallel
}

// wholeCircle says whether a file's columns cover the circle of its standard
// parallel, to within half a sample, so that its east edge meets its west.
func (frame sphereFrame) wholeCircle(raster *Raster) bool {
	return math.Abs(float64(raster.width)*raster.spacing-frame.circle()) <=
		raster.spacing*cellCentre
}

// locate gives a point's place among the sample centres, with inside false
// where the point is beyond the file. The column is wrapped onto the circle
// first; the row is held between the first and last row.
func (frame sphereFrame) locate(raster *Raster, longitude, latitude float64) (lattice, bool) {
	east := frame.falseEast + frame.radius*frame.cosParallel*
		radians(longitude-frame.centreLongitude)
	north := frame.falseNorth + frame.radius*radians(latitude-frame.centreLatitude)
	rows := (raster.north - north) / raster.rowSpacing()
	if rows < -edgeSlack || rows > float64(raster.height)+edgeSlack {
		return lattice{}, false
	}
	along := math.Mod(east-raster.west, frame.circle())
	if along < 0 {
		along += frame.circle()
	}
	column := along/raster.spacing - frame.centreOffset
	last := float64(raster.width - 1)
	whole := frame.wholeCircle(raster)
	if !whole && (column < 0 || column > last) {
		return lattice{}, false
	}
	left := math.Floor(column)
	if !whole {
		left = math.Min(left, last-1)
	}
	row := rows - frame.centreOffset
	return lattice{left: int(left), across: column - left,
		row: min(max(row, 0), float64(raster.height-1))}, true
}

// filled blends the four samples about a place among the sample centres.
// A column before the first or after the last wraps round to the other end;
// locate leaves such a column only where the file is a whole circle.
func (raster *Raster) filled(place lattice) (float64, bool, error) {
	rowFloor := int(math.Floor(place.row))
	top := min(rowFloor, raster.height-neighbourCount)
	down := place.row - float64(top)
	columnWeights := [neighbourCount]float64{1 - place.across, place.across}
	rowWeights := [neighbourCount]float64{1 - down, down}
	var height float64
	for step := range neighbourCount {
		column := wrap(place.left+step, raster.width)
		for lower := range neighbourCount {
			value, has, err := raster.sample(column, top+lower)
			if err != nil || !has {
				return 0, false, err
			}
			height += columnWeights[step] * rowWeights[lower] * value
		}
	}
	return height, true, nil
}

// wrap is an index brought into the range 0 up to size, going round the ends.
func wrap(index, size int) int {
	return ((index % size) + size) % size
}

// GroundSpacing gives the distance in metres between neighbouring samples on
// the ground at a planetocentric latitude, in degrees, as the eastward and
// the southward spacing. Both are returned because they differ, and a caller
// choosing a detail size needs the one it is refining along.
//
// Southward is the file's row scale at every latitude. Eastward is the column
// scale times the cosine of the latitude over the cosine of the standard
// parallel, and shrinks to nothing at a pole. A latitude outside minus 90 to
// 90 is an error, as is a file that is not an equirectangular sphere map.
func (raster *Raster) GroundSpacing(latitude float64) (eastward, southward float64, err error) {
	frame, ok := raster.equirectangular()
	if !ok {
		return 0, 0, fmt.Errorf("terrain: not an equirectangular source on a sphere")
	}
	if !finiteCoordinate(latitude) || math.Abs(latitude) > quarterTurn {
		return 0, 0, fmt.Errorf("terrain: invalid latitude")
	}
	return raster.spacing * math.Cos(radians(latitude)) / frame.cosParallel,
		raster.rowSpacing(), nil
}

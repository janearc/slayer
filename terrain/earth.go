package terrain

import (
	"fmt"
	"math"
)

// earthRadius is the mean radius of the earth, in metres, which a flat
// patch of ground is laid on: IUGG's mean radius, 6371.0088 km. A degree
// grid of the earth's surface says no radius of its own, as a map of
// another world does, so a ground laid on it takes this one.
const earthRadius = 6371008.8

// EarthRadius is earthRadius, for a caller that places one window in
// another on the same sphere.
const EarthRadius = earthRadius

// earth is a raster of the earth read as a map laid on a sphere: a grid of
// samples in degrees of longitude and latitude, as NOAA's and the USGS's
// elevation models are. It answers as a map of a sphere does, so a window of
// local metres can be cut from it about a latitude and longitude, as one is
// from a map of Venus. Heights keep the model's own datum.
type earth struct{ raster *Raster }

// Earth is the raster as a map of the earth, and whether it is one: a
// placed grid declared in degrees with a geographic CRS, as
// GeographicElevation reads.
func (raster *Raster) Earth() (earth, bool) {
	if raster == nil || raster.closed || !raster.placed {
		return earth{}, false
	}
	return earth{raster}, raster.reference.degreeGrid()
}

// angularDegree is the GeoTIFF code for degrees as an angular unit.
const angularDegree = 9102

// degreeCRS are the EPSG geographic systems, NAD27, NAD83, ETRS89 and
// WGS84, whose units are degrees by their registry entry, so a file naming
// one of them need not name the unit again.
var degreeCRS = map[int]bool{4267: true, 4269: true, 4258: true, 4326: true}

// degreeGrid is whether the reference is a geographic grid in degrees: the
// unit named, or left to the registered system the file names. NOAA's
// CUDEM tiles say the second.
func (ref SourceReference) degreeGrid() bool {
	// ModelTypeGeographic=2 is a GeoTIFF code.
	if ref.Model != 2 || ref.GeographicCRS == 0 {
		return false
	}
	return ref.AngularUnits == angularDegree ||
		(ref.AngularUnits == 0 && degreeCRS[ref.GeographicCRS])
}

// EquirectangularSphere is the radius of the earth, which the map is laid
// on. It is always so for an Earth.
func (earth earth) EquirectangularSphere() (radius float64, ok bool) {
	return earthRadius, true
}

// EquirectangularElevation is the height at a longitude and latitude in
// degrees, longitude first, and whether the map has one there.
func (earth earth) EquirectangularElevation(longitude,
	latitude float64) (float64, bool, error) {
	return earth.raster.GeographicElevation(longitude, latitude)
}

// GroundSpacing is how far apart the samples are on the ground at a
// latitude, eastward and southward, in metres.
func (earth earth) GroundSpacing(latitude float64) (eastward,
	southward float64, err error) {
	if !finiteCoordinate(latitude) || math.Abs(latitude) > quarterTurn {
		return 0, 0, fmt.Errorf("terrain: invalid latitude")
	}
	across := earth.raster.spacing * earthRadius * math.Pi / halfTurn
	down := earth.raster.rowSpacing() * earthRadius * math.Pi / halfTurn
	return across * math.Cos(radians(latitude)), down, nil
}

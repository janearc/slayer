package titan

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/celestial/grissom/mesh"
)

// equator is the latitude, in degrees, at which a map's row spacing is read
// for the sentence that says what it is. defined: the equator is nought.
const equator = 0

// Sphere is a raster of a whole sphere's heights, laid out by longitude and
// latitude: exactly the three things titan asks of one. package terrain's
// Raster has them. They are named here so titan does not import terrain and
// so a test can stand a mock in the raster's place.
type Sphere interface {
	// EquirectangularSphere says whether the raster is an equirectangular
	// map of a sphere, and if so the sphere's radius in metres.
	EquirectangularSphere() (radius float64, ok bool)
	// EquirectangularElevation is the height at a longitude and latitude in
	// degrees, longitude first, and whether the raster has one there.
	EquirectangularElevation(longitude, latitude float64) (float64, bool, error)
	// GroundSpacing is how far apart the raster's samples are on the ground
	// at a latitude, eastward and southward, in metres.
	GroundSpacing(latitude float64) (eastward, southward float64, err error)
}

// SphereMap is a Sphere as a Provider: the map of a world, read by the
// longitude and latitude of the places titan asks about.
type SphereMap struct {
	raster Sphere
	key    string
	source string
}

// NewSphereMap is a raster as a Provider. key is the name the raster's
// source has in sources.yaml, which Source repeats to whoever asks. It
// refuses a raster that is not an equirectangular map of a sphere, and
// one that cannot say how far apart its samples are.
func NewSphereMap(raster Sphere, key string) (*SphereMap, error) {
	if raster == nil {
		return nil, errors.New("titan: a sphere map needs a raster")
	}
	if _, ok := raster.EquirectangularSphere(); !ok {
		return nil, fmt.Errorf("titan: %s is not an equirectangular map of "+
			"a sphere", key)
	}
	_, southward, err := raster.GroundSpacing(equator)
	if err != nil {
		return nil, fmt.Errorf("titan: %s: %w", key, err)
	}
	return &SphereMap{raster: raster, key: key,
		source: fmt.Sprintf("%s, an equirectangular map whose rows are "+
			"%.4g m apart", key, southward)}, nil
}

// Height is the raster's height at the place, bilinear between its samples.
// A place the raster has no height for is not known, and its height is
// nought only because a number must be returned: nobody should read it.
func (from *SphereMap) Height(at mesh.Point) (float64, bool, error) {
	return from.raster.EquirectangularElevation(at.Lon(), at.Lat())
}

// Finest is the larger of the raster's two spacings on the ground at the
// place's latitude, in metres. Closer than the larger, the raster has
// nothing to say in that direction, so titan must not call it measured.
// Where the raster cannot give a spacing the answer is infinite: it has
// nothing to say at any distance.
func (from *SphereMap) Finest(at mesh.Point) float64 {
	eastward, southward, err := from.raster.GroundSpacing(at.Lat())
	if err != nil {
		return math.Inf(1)
	}
	return math.Max(eastward, southward)
}

// Source is a sentence naming the source and how far apart its rows are.
func (from *SphereMap) Source() string { return from.source }

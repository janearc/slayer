package frame

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// Position maps spherical geography into the fixed anchor's float64 frame.
// Near the anchor, half-angle differences avoid subtracting two planet radii.
func (sphere Sphere) Position(location Location) (maths.Vec, error) {
	if err := sphere.valid(); err != nil {
		return maths.Vec{}, err
	}
	latitude, longitude := location.Latitude.InRadians(), location.Longitude.InRadians()
	height := location.Height.InMetres()
	radius := sphere.radius + height
	if !validAngles(latitude, longitude) || !finite(height) || !finite(radius) || radius <= 0 {
		return maths.Vec{}, fmt.Errorf("frame: invalid spherical location")
	}
	deltaLatitude := latitude - sphere.latitude
	deltaLongitude := math.Remainder(longitude-sphere.longitude, 2*math.Pi)
	latHalf, lonHalf := math.Sin(deltaLatitude/2), math.Sin(deltaLongitude/2)
	sinAnchor, cosAnchor := math.Sincos(sphere.latitude)
	cosLatitude := math.Cos(latitude)
	haversine := latHalf*latHalf + cosAnchor*cosLatitude*lonHalf*lonHalf
	point := maths.V(radius*cosLatitude*math.Sin(deltaLongitude),
		(height-sphere.height)-2*radius*haversine,
		radius*(-math.Sin(deltaLatitude)-2*sinAnchor*cosLatitude*lonHalf*lonHalf))
	if !finitePoint(point) {
		return maths.Vec{}, fmt.Errorf("frame: spherical position overflow")
	}
	return point, nil
}

// radial returns a finite direction and radius relative to the sphere's centre.
func (sphere Sphere) radial(point maths.Vec) (maths.Vec, float64, error) {
	if err := sphere.valid(); err != nil {
		return maths.Vec{}, 0, err
	}
	if !finitePoint(point) {
		return maths.Vec{}, 0, fmt.Errorf("frame: non-finite world position")
	}
	radial := point.Sub(sphere.Centre())
	radius := math.Hypot(math.Hypot(radial.X, radial.Z), radial.Y)
	if !finite(radius) || radius <= 0 {
		return maths.Vec{}, 0, fmt.Errorf("frame: position has no finite radial direction")
	}
	return radial, radius, nil
}

// Location maps a world point back to spherical geography and radial height.
// Zero horizontal components use the anchor's meridian by convention.
func (sphere Sphere) Location(point maths.Vec) (Location, error) {
	radial, radius, err := sphere.radial(point)
	if err != nil {
		return Location{}, err
	}
	up := radialUnit(radial, radius)
	latitude, longitude := sphere.angular(up)
	height := sphere.radialHeight(point, radial, radius)
	if !finite(height) {
		return Location{}, fmt.Errorf("frame: radial height overflow")
	}
	return Location{Latitude: measure.Radians(latitude), Longitude: measure.Radians(longitude),
		Height: measure.Metres(height)}, nil
}

// angular rotates a radial unit vector into the anchor's geographic basis.
func (sphere Sphere) angular(up maths.Vec) (latitude, longitude float64) {
	sine, cosine := math.Sincos(sphere.latitude)
	meridian := up.Y*cosine + up.Z*sine
	pole := up.Y*sine - up.Z*cosine
	latitude = math.Atan2(pole, math.Hypot(meridian, up.X))
	delta := math.Atan2(up.X, meridian)
	if meridian == 0 && up.X == 0 {
		delta = 0
	}
	longitude = math.Remainder(sphere.longitude+delta, 2*math.Pi)
	return latitude, longitude
}

// radialUnit divides components directly so a tiny radius needs no reciprocal.
func radialUnit(radial maths.Vec, radius float64) maths.Vec {
	return maths.V(radial.X/radius, radial.Y/radius, radial.Z/radius)
}

// radialHeight rationalizes the small near-anchor radius difference.
// The opposite hemisphere uses ordinary float64 radial subtraction.
func (sphere Sphere) radialHeight(point, radial maths.Vec, radius float64) float64 {
	if radial.Y < 0 {
		return radius - sphere.radius
	}
	denominator := radius + radial.Y
	if !finite(denominator) {
		return radius - sphere.radius
	}
	return sphere.height + point.Y +
		radial.X*(radial.X/denominator) + radial.Z*(radial.Z/denominator)
}

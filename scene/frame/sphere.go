// Package frame defines an anchored float64 world and render-relative
// positions. It supplies geometry, not a projection, ephemeris or GPU transport
// adapter.
package frame

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// Location uses spherical latitude, longitude and height above a stated sphere.
// A DEM's orthometric height needs an explicit datum adapter before use here.
type Location struct {
	Latitude, Longitude measure.Angle
	Height              measure.Length
}

// Sphere retains one fixed anchor for a whole scene or sequence.
// Its coordinates are metres east, up and south in the anchor's tangent basis.
type Sphere struct {
	radius, latitude, longitude, height float64
}

// NewSphere fixes a reference radius and anchor without consulting a DEM crop.
// The radius is supplied by the caller; no planetary size is inferred.
func NewSphere(radius measure.Length, anchor Location) (Sphere, error) {
	sphere := Sphere{radius: radius.InMetres(), height: anchor.Height.InMetres(),
		latitude: anchor.Latitude.InRadians(), longitude: anchor.Longitude.InRadians()}
	if err := sphere.valid(); err != nil {
		return Sphere{}, err
	}
	return sphere, nil
}

// Radius is the caller's reference-sphere radius.
func (sphere Sphere) Radius() measure.Length { return measure.Metres(sphere.radius) }

// Anchor is the fixed geographic position represented by the world origin.
func (sphere Sphere) Anchor() Location {
	return Location{Latitude: measure.Radians(sphere.latitude),
		Longitude: measure.Radians(sphere.longitude), Height: measure.Metres(sphere.height)}
}

// Centre is the reference sphere's centre in the fixed east/up/south frame.
func (sphere Sphere) Centre() maths.Vec {
	return maths.V(0, -(sphere.radius + sphere.height), 0)
}

// valid also refuses the zero Sphere before coordinate operations.
func (sphere Sphere) valid() error {
	if !finite(sphere.radius) || sphere.radius <= 0 ||
		!finite(sphere.height) || !finite(sphere.radius+sphere.height) ||
		sphere.radius+sphere.height <= 0 ||
		!validAngles(sphere.latitude, sphere.longitude) {
		return fmt.Errorf("frame: invalid spherical reference or anchor")
	}
	return nil
}

// validAngles defines geographic ranges in radians, including both poles.
func validAngles(latitude, longitude float64) bool {
	return finite(latitude) && finite(longitude) &&
		latitude >= -math.Pi/2 && latitude <= math.Pi/2 &&
		longitude >= -math.Pi && longitude <= math.Pi
}

// finite separates invalid geometry from a legitimate large displacement.
func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

// finitePoint checks every position before it reaches a subtraction or
// encoding.
func finitePoint(point maths.Vec) bool { return finite(point.X) && finite(point.Y) && finite(point.Z) }

package frame

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// Nadir points from a world position to the sphere's centre, not along local
// -y.
func (sphere Sphere) Nadir(point maths.Vec) (maths.Vec, error) {
	radial, radius, err := sphere.radial(point)
	if err != nil {
		return maths.Vec{}, err
	}
	return radialUnit(radial, radius).Neg(), nil
}

// Tangent gives local east/up/south axes at a world position's radial subpoint.
// The pole convention follows Location; no time-evolving orbit is implied.
func (sphere Sphere) Tangent(point maths.Vec) (maths.Orientation, error) {
	radial, radius, err := sphere.radial(point)
	if err != nil {
		return maths.Orientation{}, err
	}
	up := radialUnit(radial, radius)
	_, longitude := sphere.angular(up)
	delta := longitude - sphere.longitude
	sinDelta, cosDelta := math.Sincos(delta)
	sinAnchor, cosAnchor := math.Sincos(sphere.latitude)
	east := maths.V(cosDelta, -cosAnchor*sinDelta, -sinAnchor*sinDelta)
	return maths.Orientation{X: east, Y: up, Z: east.Cross(up).Unit()}, nil
}

// Compass resolves azimuth clockwise from north, with elevation above the
// position's own tangent horizon. Roll belongs to the instrument/camera pose.
func (sphere Sphere) Compass(point maths.Vec,
	azimuth, elevation measure.Angle,
) (maths.Vec, error) {
	az, el := azimuth.InRadians(), elevation.InRadians()
	if !finite(az) || !finite(el) || el < -math.Pi/2 || el > math.Pi/2 {
		return maths.Vec{}, fmt.Errorf("frame: invalid local compass direction")
	}
	basis, err := sphere.Tangent(point)
	if err != nil {
		return maths.Vec{}, err
	}
	sinAz, cosAz := math.Sincos(az)
	sinEl, cosEl := math.Sincos(el)
	local := maths.V(sinAz*cosEl, sinEl, -cosAz*cosEl)
	return basis.Out(local).Unit(), nil
}

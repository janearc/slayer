package set

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	yaml "gopkg.in/yaml.v3"
)

// laserDirection is one shared boresight, in degrees east of north and up.
type laserDirection struct{ azimuth, altitude float64 }

// arcsecondsPerDegree is defined by sixty minutes of sixty seconds.
const arcsecondsPerDegree = 3600

// laserPointings reads named directions before any references are resolved.
func (reading reader) laserPointings(_, block *yaml.Node) error {
	if len(block.Content) == laserZero {
		return reading.refuse(block,
			"laser-pointings needs a direction")
	}
	reading.set.laserPointings = map[string]laserDirection{}
	return reading.each(block, reading.namedLaserDirection)
}

// namedLaserDirection reads one entry without exposing mutable scene state.
func (reading reader) namedLaserDirection(name, value *yaml.Node) error {
	direction := laserDirection{}
	err := reading.laserDirection(value,
		&direction.azimuth, &direction.altitude)
	if err != nil {
		return err
	}
	reading.set.laserPointings[name.Value] = direction
	return nil
}

// laserDirection requires both angles even when either is explicitly zero.
func (reading reader) laserDirection(block *yaml.Node,
	azimuth, altitude *float64,
) error {
	err := reading.keys(block, map[string]step{
		"azimuth": reading.number(azimuth, laserZero, laserTurn),
		"altitude": reading.number(altitude,
			-laserQuarter, laserQuarter),
	}, nil)
	if err != nil {
		return err
	}
	return reading.all(block, "laser pointing", "azimuth", "altitude")
}

// offset uses sky-tangent angles, not independent azimuth differences.
// Positive horizontal follows increasing azimuth at the common direction.
func (input laserReader) offset(_, block *yaml.Node) error {
	limit := laserLimit("maximum-offset-arcsec")
	laser, reading := input.laser, input.reading
	err := reading.keys(block, map[string]step{
		"horizontal": reading.quantity(&laser.horizontal, "arcsec",
			-limit, limit),
		"vertical": reading.quantity(&laser.vertical, "arcsec",
			-limit, limit),
	}, nil)
	if err != nil {
		return err
	}
	return reading.all(block, "guide-star offset", "horizontal", "vertical")
}

// resolveLaserPointing runs after all scene keys, allowing forward references.
func (reading reader) resolveLaserPointing(laser *Laser) error {
	if laser.shared != "" {
		direction, found := reading.set.laserPointings[laser.shared]
		if !found {
			return fmt.Errorf("%s: laser %s "+
				"names unknown pointing %q",
				reading.file, laser.name, laser.shared)
		}
		laser.azimuth = direction.azimuth
		laser.altitude = direction.altitude
	}
	if laser.horizontal == laserZero && laser.vertical == laserZero {
		return nil
	}
	laser.azimuth, laser.altitude = laser.offsetDirection()
	return nil
}

// offsetDirection projects a tangent-plane location onto the viewing sphere.
// The boresight azimuth also fixes the tangent-frame roll at the zenith.
func (laser Laser) offsetDirection() (float64, float64) {
	azimuth, altitude := laser.azimuth*maths.Degree,
		laser.altitude*maths.Degree
	sinAz, cosAz := math.Sincos(azimuth)
	sinAlt, cosAlt := math.Sincos(altitude)
	axis := maths.V(sinAz*cosAlt, sinAlt, -cosAz*cosAlt)
	horizontal := maths.V(cosAz, laserZero, sinAz)
	vertical := maths.V(-sinAz*sinAlt, cosAlt, cosAz*sinAlt)
	angle := maths.Degree / arcsecondsPerDegree
	across := math.Tan(laser.horizontal * angle)
	up := math.Tan(laser.vertical * angle)
	direction := axis.Add(horizontal.Scale(across)).Add(vertical.Scale(up))
	bearing := math.Atan2(direction.X, -direction.Z) / maths.Degree
	bearing = math.Mod(bearing+laserTurn, laserTurn)
	run := math.Hypot(direction.X, direction.Z)
	rise := math.Atan2(direction.Y, run) / maths.Degree
	return bearing, rise
}

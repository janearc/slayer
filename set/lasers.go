package set

import (
	_ "embed"
	"fmt"
	"slices"
	"sync"

	"github.com/janearc/puzzlemouth/numbers"

	yaml "gopkg.in/yaml.v3"
)

//go:embed laser-limits.yaml
var laserLimitRaw []byte

var laserLimits = sync.OnceValues(readLaserLimits)

// readLaserLimits checks the authored input budgets through Puzzlemouth.
func readLaserLimits() (numbers.Numbers, error) {
	return numbers.Read("laser-limits.yaml", laserLimitRaw)
}

// laserLimit names a checked limit; a missing embedded name is a code fault.
func laserLimit(name string) float64 {
	limits, err := laserLimits()
	if err != nil {
		panic(err)
	}
	value, err := limits.Get(name)
	if err != nil {
		panic(err)
	}
	return value.Value
}

// laserZero is defined as no angle or no emitted light.
const laserZero = 0

// laserTurn and laserQuarter are defined in degrees per full/quarter turn.
const laserTurn, laserQuarter = 360, 90

// foundLasers checks references after all scene blocks have been read.
func (reading reader) foundLasers() error {
	for index := range reading.set.lasers {
		laser := &reading.set.lasers[index]
		if err := reading.resolveLaserPointing(laser); err != nil {
			return err
		}
		if laser.object != "" && !reading.set.named(laser.object) {
			return fmt.Errorf("%s: laser %s "+
				"names unknown object %q",
				reading.file, laser.name, laser.object)
		}
	}
	return nil
}

// Laser is a checked scene beam. Offsets are world east, up and south in
// metres, added to the named object's origin when one is supplied.
type Laser struct {
	name, object              string
	shared                    string
	launcher                  bool
	horizontal, vertical      float64
	x, y, z                   float64
	azimuth, altitude         float64
	radius, reach, divergence float64
	wavelength, strength      float64
}

// Lasers returns a caller-owned list of checked launch specifications.
func (made Set) Lasers() []Laser { return slices.Clone(made.lasers) }

// Name identifies the launch in diagnostics and scene catalogues.
func (laser Laser) Name() string { return laser.name }

// Origin is an optional object and an offset in the scene's world axes.
func (laser Laser) Origin() (string, float64, float64, float64) {
	return laser.object, laser.x, laser.y, laser.z
}

// Pointing is azimuth east of north and altitude above the horizon, degrees.
func (laser Laser) Pointing() (float64, float64) {
	return laser.azimuth, laser.altitude
}

// Envelope is launch radius and axial reach in metres, and half-divergence
// in degrees. The radius follows the 1/e^2 intensity convention.
func (laser Laser) Envelope() (float64, float64, float64) {
	return laser.radius, laser.reach, laser.divergence
}

// Spectrum is wavelength in nanometres and authored peak display strength.
func (laser Laser) Spectrum() (float64, float64) {
	return laser.wavelength, laser.strength
}

// lasers reads finite outgoing beams. Display strength deliberately does not
// claim watts or a measured sodium return. The beam model is in guide-stars.md.
func (reading reader) lasers(_, node *yaml.Node) error {
	return reading.each(node, reading.laser)
}

// laser reads one launch, keeping nested coordinates independently checked.
func (reading reader) laser(name, block *yaml.Node) error {
	laser := Laser{name: name.Value}
	input := laserReader{reading: reading, laser: &laser}
	steps := map[string]step{
		"origin":            input.origin,
		"launcher":          reading.yes(&laser.launcher),
		"pointing":          input.pointing,
		"guide-star-offset": input.offset,
		"radius": reading.quantity(&laser.radius, "m",
			laserLimit("minimum-radius"),
			laserLimit("maximum-radius")),
		"reach": reading.quantity(&laser.reach, "m",
			laserLimit("minimum-reach"), farthest),
		"divergence": reading.quantity(&laser.divergence, "deg",
			laserZero, laserLimit("maximum-divergence")),
		"wavelength": reading.quantity(&laser.wavelength, "nm",
			laserLimit("minimum-wavelength"),
			laserLimit("maximum-wavelength")),
		"display-strength": reading.number(&laser.strength, laserZero,
			laserLimit("maximum-strength")),
	}
	err := reading.keys(block, steps, nil)
	if err != nil {
		return err
	}
	if err := reading.all(block, "laser", "origin", "pointing", "radius",
		"reach", "divergence", "wavelength",
		"display-strength"); err != nil {
		return err
	}
	reading.set.lasers = append(reading.set.lasers, laser)
	return nil
}

// laserReader owns the target while reading its nested blocks.
type laserReader struct {
	reading reader
	laser   *Laser
}

// origin accepts an optional object, requiring each world-axis offset.
func (input laserReader) origin(_, block *yaml.Node) error {
	reading, laser := input.reading, input.laser
	err := reading.keys(block, map[string]step{
		"object": reading.text(&laser.object),
		"east":   reading.quantity(&laser.x, "m", -farthest, farthest),
		"up":     reading.quantity(&laser.y, "m", -farthest, farthest),
		"south":  reading.quantity(&laser.z, "m", -farthest, farthest),
	}, nil)
	if err != nil {
		return err
	}
	return reading.all(block, "laser origin", "east", "up", "south")
}

// pointing accepts explicit angles or a named common boresight.
func (input laserReader) pointing(name, block *yaml.Node) error {
	if block.Kind == yaml.ScalarNode {
		return input.reading.text(&input.laser.shared)(name, block)
	}
	return input.reading.laserDirection(block,
		&input.laser.azimuth, &input.laser.altitude)
}

// Launcher requests a visible tube whose exit and axis match this launch.
// Its dimensions are authored scenery; the beam envelope remains independent.
func (laser Laser) Launcher() bool { return laser.launcher }

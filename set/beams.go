package set

import (
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
	yaml "gopkg.in/yaml.v3"
)

// Beam is a finite Gaussian light column: where it begins, which way it
// travels, its 1/e² intensity radius, its reach, its half-divergence in
// degrees, and its peak black-body light.
type Beam struct {
	name                      string
	at, aim                   [3]float64
	radius, reach, divergence float64
	temperature, strength     float64
}

// Name identifies the authored beam.
func (made Beam) Name() string { return made.name }

// At is the beam's launch point in the world's coordinates.
func (made Beam) At() maths.Vec { return fromMap(made.at) }

// Aim is its direction in the world's coordinates, as long as written.
func (made Beam) Aim() maths.Vec { return fromMap(made.aim) }

// Envelope is the source-plane 1/e² intensity radius and finite reach in
// world units, followed by the half-divergence in degrees.
func (made Beam) Envelope() (radius, reach, divergence float64) {
	return made.radius, made.reach, made.divergence
}

// Shine is the black-body temperature in kelvin and peak source-plane
// luminance.
func (made Beam) Shine() (temperature, strength float64) {
	return made.temperature, made.strength
}

// Beams returns every authored beam in the order written.
func (made World) Beams() []Beam { return slices.Clone(made.beams) }

// beams reads finite Gaussian beams by name.
func (reading reader) beams(_, value *yaml.Node) error {
	return reading.each(value, func(name, block *yaml.Node) error {
		made := Beam{name: name.Value}
		var source string
		err := reading.keys(block, map[string]step{
			"at":  reading.three(&made.at, -farthest, farthest),
			"aim": reading.three(&made.aim, -farthest, farthest),
			"radius": reading.number(&made.radius,
				math.SmallestNonzeroFloat64, farthest),
			"reach": reading.number(&made.reach,
				math.SmallestNonzeroFloat64, farthest),
			"divergence": reading.number(&made.divergence, 0,
				math.Nextafter(90, 0)),
			"temperature": reading.number(&made.temperature, 1, hottest),
			"strength":    reading.number(&made.strength, 0, brightest),
			"source":      reading.source(&source),
		}, nil)
		if err == nil {
			err = reading.all(block, "a beam", "at", "aim", "radius",
				"reach", "divergence", "temperature", "strength", "source")
		}
		if err == nil && made.aim == [3]float64{} {
			err = reading.refuse(block, "beam %s aims at nought, nought and nought, which is no way", made.name)
		}
		if err == nil && !finiteBeamEnvelope(made.radius, made.reach, made.divergence) {
			err = reading.refuse(block, "beam %s has a non-finite envelope", made.name)
		}
		reading.set.world.beams = append(reading.set.world.beams, made)
		return err
	})
}

// finiteBeamEnvelope is whether the widest source radius plus its divergent
// growth remains finite in the world's coordinates.
func finiteBeamEnvelope(radius, reach, divergence float64) bool {
	return !math.IsInf(radius+reach*math.Tan(divergence*maths.Degree), 0) &&
		!math.IsNaN(radius+reach*math.Tan(divergence*maths.Degree))
}

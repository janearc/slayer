package thing

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/materials"
	"github.com/janearc/slayer/maths"
)

// sameBand is how near a band must be to a coefficient's photon to be
// that photon, as a share of the wavelength: the rounding of an energy
// turned into a wavelength and back, and no nearer than that.
const sameBand = 1e-9

// passing is one band a stopping material answers: the photon's
// wavelength in nanometres and the share of it one unit of the thing
// passes
type passing struct {
	nanometres float64
	share      float64
}

// stopping is the Radiant Stopping makes: the bands a material has
// coefficients at, each with the share one unit of the thing passes
type stopping struct {
	bands []passing
}

// Stopping is what a material does to a beam of x-rays or gamma rays,
// from its mass attenuation coefficients and its density: a Radiant
// that passes, of a band whose photon the material has a coefficient
// at, the share that gets through one unit of the thing.
//
// Every other
// band it stops, for nothing is interpolated between a table's rows:
// a picture in a band no row was read at shows the thing opaque, and
// not a share nobody measured. It gives off nothing and sends nothing
// back.
//
// The unit is the length of one of the thing's own units, in metres.
// The tracer measures the way through a thing in the thing's units and
// raises what one unit passes to the power of the way, which is Beer
// and Lambert's law when one unit's share is e to the minus mu over
// rho, times the density, times the unit.
//
// It refuses a material with no coefficients, by name, one with no
// density cited, and a unit that is not a positive number.
func Stopping(kind materials.Material, unit float64) (Radiant, error) {
	if !(unit > 0) || math.IsInf(unit, 0) {
		return nil, fmt.Errorf("thing: %s stopping a beam by a unit of %v m",
			kind.Name(), unit)
	}
	rows := kind.Attenuations().All()
	if len(rows) == 0 {
		return nil, fmt.Errorf("thing: %s has no attenuation coefficients",
			kind.Name())
	}
	made := stopping{bands: make([]passing, 0, len(rows))}
	for _, row := range rows {
		share, err := kind.Passes(row.Energy(), unit)
		if err != nil {
			return nil, fmt.Errorf("thing: %w", err)
		}
		made.bands = append(made.bands,
			passing{nanometres: row.Wavelength(), share: share})
	}
	return made, nil
}

// Through is how much of the band passes one unit of the thing: the
// share at the coefficient read at the band's photon, and nothing of a
// band no coefficient was read at
func (made stopping) Through(band maths.Band) float64 {
	for _, each := range made.bands {
		if math.Abs(band.Nanometres-each.nanometres) <=
			sameBand*each.nanometres {
			return each.share
		}
	}
	return 0
}

// Emits is nothing: a material that only stops a beam gives none off
func (made stopping) Emits(maths.Band) float64 { return 0 }

// Scatters is nothing: what the material takes from the beam is taken,
// and none is sent back toward a lamp or a radar
func (made stopping) Scatters(maths.Band) float64 { return 0 }

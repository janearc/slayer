package emit

import (
	"errors"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// Beam is a finite Gaussian light column. Radius is the source-plane 1/e^2
// intensity radius; divergence is its geometric half-angle. This ray-optical
// approximation omits diffraction, interference and atmospheric refraction.
// A beam has no visible skin: surfaces and participating media scatter it.
//
// See doc/guide-stars.md for the model, source and display conventions.
type Beam struct {
	point                Lit
	radius, reach, slope float64
}

// beamZero is the defined additive identity.
const beamZero = 0

// beamRightAngle is defined as one quarter turn, in radians.
const beamRightAngle = math.Pi / 2

// BeamExponent is defined by the 1/e^2 intensity-radius convention.
const BeamExponent = -2

// NewBeam uses point.Strength and Spectrum as peak source-plane irradiance,
// rather than point-source strength at unit distance. Toward must be nonzero.
// Point exclusions and cone controls are refused: radius, reach and divergence
// describe this source. A zero divergence keeps the cross section constant.
func NewBeam(point Lit, radius, reach measure.Length,
	divergence measure.Angle,
) (*Beam, error) {
	if err := checkBeamPoint(point); err != nil {
		return nil, err
	}
	width, length := radius.InMetres(), reach.InMetres()
	angle := divergence.InRadians()
	if width <= beamZero || length <= beamZero || angle < beamZero ||
		angle >= beamRightAngle {
		return nil, errors.New("emit: invalid beam radius, " +
			"reach or divergence")
	}
	slope := math.Tan(angle)
	// Positive lengths and a nonnegative slope cannot cancel an infinity.
	// This also rejects any NaN in the dimensions or divergence.
	if !beamFinite(width + length*slope) {
		return nil, errors.New("emit: non-finite beam envelope")
	}
	point.Toward = point.Toward.Unit()
	return &Beam{point: point, radius: width, reach: length,
		slope: slope}, nil
}

// checkBeamPoint checks launch metadata without imposing an RGB gamut.
func checkBeamPoint(point Lit) error {
	if point.Beyond != beamZero || point.Width != beamZero ||
		point.Soft != beamZero {
		return errors.New("emit: a beam has no point " +
			"exclusion or cone")
	}
	for _, vector := range []maths.Vec{point.At, point.Toward} {
		if !beamFinite(vector.Len()) {
			return errors.New("emit: non-finite beam " +
				"position or direction")
		}
	}
	if point.Toward.Len() <= beamZero {
		return errors.New("emit: a beam needs a direction")
	}
	// Monochromatic spectra can have signed linear RGB coordinates outside
	// the display gamut. Luminance stays nonnegative; clipping coordinates
	// here would change the spectrum's display conversion.
	// The positive luminance weights also propagate any non-finite channel.
	luminance := point.Strength.Luminance()
	if !beamFinite(luminance) || luminance < beamZero {
		return errors.New("emit: invalid beam strength")
	}
	return nil
}

// beamFinite refuses NaN and either infinity.
func beamFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, beamZero)
}

// Point returns the immutable launch description as a caller-owned copy.
func (beam *Beam) Point() Lit { return beam.point }

// Radius is the launch-plane 1/e^2 intensity radius.
func (beam *Beam) Radius() measure.Length { return measure.Metres(beam.radius) }

// Reach is the finite axial extent, measured from the launch plane.
func (beam *Beam) Reach() measure.Length { return measure.Metres(beam.reach) }

// Slope is the growth of the radius per unit of axial distance.
func (beam *Beam) Slope() float64 { return beam.slope }

// In is the source-plane peak spectrum in the requested band.
func (beam *Beam) In(band maths.Band) float64 { return beam.point.In(band) }

// Light returns the parallel-ray approximation toward the launch plane.
// Its distance is axial, so an obstruction casts a shadow along the beam.
func (beam *Beam) Light(at, _ maths.Vec) (maths.Vec, float64,
	maths.Light) {
	along, weight := beam.falling(at)
	if weight == beamZero {
		toward := maths.V(beamZero, beamZero, beamZero)
		return toward, beamZero, maths.Grey(beamZero)
	}
	return beam.point.Toward.Neg(), along, beam.point.Strength.Scale(weight)
}

// Lands uses the same beam envelope for spectral pictures and RGB lighting.
func (beam *Beam) Lands(at maths.Vec, band maths.Band) float64 {
	_, weight := beam.falling(at)
	return beam.In(band) * weight
}

// falling widens a Gaussian while preserving its transverse integral.
// Both ends are open: the launch plane and the far plane receive no light.
func (beam *Beam) falling(at maths.Vec) (float64, float64) {
	delta := at.Sub(beam.point.At)
	along := delta.Dot(beam.point.Toward)
	if !beamFinite(along) || along <= beamZero || along >= beam.reach {
		return beamZero, beamZero
	}
	width := beam.radius + along*beam.slope
	// Subtraction of axial vectors avoids cancellation of squared lengths.
	off := delta.Sub(beam.point.Toward.Scale(along)).Len() / width
	// If the transverse norm overflows, exp(-Inf) gives no arriving light.
	ratio := beam.radius / width
	return along, ratio * ratio * math.Exp(BeamExponent*off*off)
}

var _ Banded = (*Beam)(nil)

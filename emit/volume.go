package emit

import (
	"errors"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// Volume is a uniformly emitting sphere used as a finite source-cell proxy.
// Its point description holds the far-field strength. Direction and
// visibility use
// the centre ray; this is not a full angular integration of occlusion.
type Volume struct {
	point  Lit
	radius float64
}

// NewVolume spreads an isotropic source over a sphere without changing its
// far-field strength. Surface exclusions and beams are not volume sources.
func NewVolume(point Lit, radius measure.Length) (Volume, error) {
	at, strength := point.At, point.Strength
	if point.Beyond != 0 || point.Width != 0 || point.Soft != 0 {
		return Volume{}, errors.New("emit: a volume source is isotropic " +
			"and has no excluded interior")
	}
	size := radius.InMetres()
	if !(size > 0) || math.IsInf(size, 1) {
		return Volume{}, errors.New(
			"emit: radius must be finite and positive")
	}
	numbers := []float64{at.X, at.Y, at.Z,
		strength.R, strength.G, strength.B}
	for _, value := range numbers {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return Volume{},
				errors.New("emit: nonfinite volume source")
		}
	}
	if strength.R < 0 || strength.G < 0 || strength.B < 0 {
		return Volume{}, errors.New("emit: negative source strength")
	}
	return Volume{point: point, radius: size}, nil
}

// Point returns a copy of the far-field source description.
func (source Volume) Point() Lit { return source.point }

// In is the original source power in the requested band.
func (source Volume) In(band maths.Band) float64 { return source.point.In(band) }

// Radius is the sphere's physical radius.
func (source Volume) Radius() measure.Length {
	return measure.Metres(source.radius)
}

// Light uses the integrated scalar falloff of a uniform emitting ball.
// The centre has no unique direction; Y is a finite convention there.
func (source Volume) Light(at, _ maths.Vec) (maths.Vec, float64,
	maths.Light) {
	toward := source.point.At.Sub(at)
	distance := toward.Len()
	in := maths.YAxis
	if distance > 0 {
		in = toward.Scale(1 / distance)
	}
	falling := volumeFalloff(distance, source.radius)
	return in, distance, source.point.Strength.Scale(falling)
}

// Lands applies the same finite-source approximation to the band spectrum.
func (source Volume) Lands(at maths.Vec, band maths.Band) float64 {
	distance := source.point.At.Sub(at).Len()
	return source.In(band) * volumeFalloff(distance, source.radius)
}

// volumeFalloff is the volume average of inverse squared distance over a
// ball. Its centre and surface limits are 3/R^2 and 3/(2R^2). Near/far series
// avoid cancellation in the closed form; see doc/fire-recipes.md.
func volumeFalloff(distance, radius float64) float64 {
	q := distance / radius
	if q <= 0.25 {
		u := q * q
		return 3 / (radius * radius) * (1 - u*(1.0/3+u*(1.0/15+u*(1.0/35+u/63))))
	}
	if q >= 4 {
		u := (radius / distance) * (radius / distance)
		return (1 + u*(1.0/5+u*(3.0/35+u/21))) / (distance * distance)
	}
	if q == 1 {
		return 1.5 / (radius * radius)
	}
	return 1.5 / (radius * radius) * (1 + (1-q*q)/(2*q)*math.Log((1+q)/math.Abs(1-q)))
}

var _ Banded = Volume{}

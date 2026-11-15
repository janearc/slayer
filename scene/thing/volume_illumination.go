package thing

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// VolumeIllumination opts painted media into normalized isotropic scattering
// from lamps and sky. Zero keeps the legacy lamp-only response. This is single
// scattering; light scattered elsewhere in the medium is not sampled again.
type VolumeIllumination struct {
	samples int
	grid    *IncidentGrid
}

// NewVolumeIllumination bounds the angular work at each scattering location.
// Antipodal pairs keep a constant sky's two hemispheres equally represented.
// Sample count controls quadrature cost, not a physical property of the medium.
func NewVolumeIllumination(samples int) (VolumeIllumination, error) {
	if samples < 2 || samples > 64 || samples%2 != 0 {
		return VolumeIllumination{}, fmt.Errorf("thing: volume sky samples must be even and between 2 and 64")
	}
	return VolumeIllumination{samples: samples}, nil
}

// Samples returns zero for legacy lighting, otherwise the checked sphere count.
func (light VolumeIllumination) Samples() int { return light.samples }

// Directions returns equal-solid-angle Fibonacci samples in fixed world axes.
// There is no camera-dependent rotation or random seed to shimmer in a movie.
func (light VolumeIllumination) Directions() []maths.Vec {
	ways := make([]maths.Vec, light.samples)
	half := light.samples / 2
	for index := range half {
		y := (float64(index) + .5) / float64(half)
		radius := math.Sqrt(1 - y*y)
		angle := float64(index) * math.Pi * (3 - math.Sqrt(5))
		way := maths.V(radius*math.Cos(angle), y, radius*math.Sin(angle))
		ways[2*index], ways[2*index+1] = way, way.Neg()
	}
	return ways
}

// IsotropicPhase is the normalized phase density per steradian. Integrating
// it over the full sphere gives one: PBRT 4e, section 11.3, Phase Functions.
// https://www.pbr-book.org/4ed/Volume_Scattering/Phase_Functions
func IsotropicPhase() float64 { return 1 / (4 * math.Pi) }

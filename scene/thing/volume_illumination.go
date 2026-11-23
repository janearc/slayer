package thing

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// The counts a volume's sky samples are drawn with.
const (
	// volumeNone is defined: no amount.
	volumeNone = 0
	// volumeWhole is defined: one amount, the whole of a share.
	volumeWhole = 1
	// volumePair is defined: two, the parts of an antipodal pair, and what
	// halves a sample count.
	volumePair = 2
	// volumeFewest is defined: two samples, one antipodal pair.
	volumeFewest = 2
	// volumeMiddle is defined: half a step, the middle of a band of the
	// sphere.
	volumeMiddle = 0.5
	// volumeGoldenThree is defined: the three in the golden angle, which
	// is pi times three less the square root of five.
	volumeGoldenThree = 3
	// volumeGoldenFive is defined: the five in the golden angle.
	volumeGoldenFive = 5
	// volumeSphere is defined: four, the factor of pi in the solid angle
	// of a sphere, 4 pi steradians.
	volumeSphere = 4
)

// volumeMost is the most sphere samples at a scattering location. It is
// read from picks.yaml.
var volumeMost = int(pick("volume-samples-most"))

// VolumeIllumination opts painted media into normalized isotropic scattering
// from lamps and sky. Zero lights them by lamps alone. This is single
// scattering; light scattered elsewhere in the medium is not sampled again.
type VolumeIllumination struct {
	samples int
	grid    *IncidentGrid
}

// NewVolumeIllumination bounds the angular work at each scattering location.
// Antipodal pairs keep a constant sky's two hemispheres equally represented.
// Sample count controls quadrature cost, not a physical property of the medium.
func NewVolumeIllumination(samples int) (VolumeIllumination, error) {
	if samples < volumeFewest || samples > volumeMost ||
		samples%volumePair != volumeNone {
		return VolumeIllumination{}, fmt.Errorf("thing: volume sky samples must be even and between 2 and 64")
	}
	return VolumeIllumination{samples: samples}, nil
}

// Samples returns zero for lamp-only lighting, otherwise the checked sphere
// count.
func (light VolumeIllumination) Samples() int { return light.samples }

// Directions returns equal-solid-angle Fibonacci samples in fixed world axes.
// There is no camera-dependent rotation or random seed to shimmer in a movie.
func (light VolumeIllumination) Directions() []maths.Vec {
	ways := make([]maths.Vec, light.samples)
	half := light.samples / volumePair
	for index := range half {
		y := (float64(index) + volumeMiddle) / float64(half)
		radius := math.Sqrt(volumeWhole - y*y)
		angle := float64(index) * math.Pi *
			(volumeGoldenThree - math.Sqrt(volumeGoldenFive))
		way := maths.V(radius*math.Cos(angle), y, radius*math.Sin(angle))
		ways[volumePair*index] = way
		ways[volumePair*index+volumeWhole] = way.Neg()
	}
	return ways
}

// IsotropicPhase is the normalized phase density per steradian. Integrating
// it over the full sphere gives one: PBRT 4e, section 11.3, Phase Functions.
// https://www.pbr-book.org/4ed/Volume_Scattering/Phase_Functions
func IsotropicPhase() float64 { return volumeWhole / (volumeSphere * math.Pi) }

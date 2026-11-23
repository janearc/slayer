package over

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// How the made roughness is built. Each is picked.
const (
	// wavesPerOctave is how many plane waves are laid in each octave, along
	// directions all round, so that no way is the way the rock runs.
	wavesPerOctave = 8
	// wavelengthLeast and wavelengthSpan: a wave's wavelength is its
	// octave's times a share between these, so that no two are the same
	// size. The longest it can be is one and a quarter of the octave's.
	wavelengthLeast = 0.75
	wavelengthSpan  = 0.5
	// peakShare is how many times the root mean square a slope of made
	// relief may reach. The slope that is kept under the rock's is the
	// root mean square, and this is how much more the steepest place is.
	peakShare = 3.0
	// mostOctaves is the most octaves roughness is made in: far more than
	// halving from a survey's spacing to a centimetre takes.
	mostOctaves = 64
	// alongFrom is the share of the survey's wavelength, along the ground,
	// at which a wave begins to fade; it is gone at the survey's own.
	alongFrom = 0.75
)

// wave is one plane wave: its wave number, in radians a metre along each
// way, and where in its turn it starts.
type wave struct {
	number maths.Vec
	phase  float64
}

// octave is the plane waves of one size, each of a wavelength near it.
type octave struct {
	wavelength float64
	waves      [wavesPerOctave]wave
}

// octaves are the sizes made roughness comes in: the longest wavelength,
// then half of it, and so on while a size is no shorter than the shortest.
// Each wave is drawn from the seed and its place in its octave and nothing
// else. Every wave is shorter than the longest wavelength and a quarter
// over, in space. Along the ground a wave can be longer than in space:
// along is what keeps it under the survey's wavelength there.
func octaves(seed uint64, longest, shortest float64) []octave {
	var sizes []octave
	for index := range mostOctaves {
		wavelength := longest / math.Pow(2, float64(index))
		if wavelength < shortest {
			break
		}
		sizes = append(sizes, newOctave(seed+uint64(index), wavelength))
	}
	return sizes
}

// newOctave is one octave's waves, drawn from the seed.
func newOctave(seed uint64, wavelength float64) octave {
	made := octave{wavelength: wavelength}
	for i := range made.waves {
		draw := func(k int64) float64 {
			return noise.Hash(seed, int64(i)*4+k)
		}
		up := 2*draw(0) - 1
		round := math.Sqrt(1 - up*up)
		turn := 2 * math.Pi * draw(1)
		direction := maths.V(round*math.Cos(turn), up, round*math.Sin(turn))
		length := wavelength * (wavelengthLeast + wavelengthSpan*draw(2))
		made.waves[i] = wave{number: direction.Scale(2 * math.Pi / length),
			phase: 2 * math.Pi * draw(3)}
	}
	return made
}

// along is the octave's relief at a point on ground of a normal, as at
// is, but with each wave faded by its wavelength along that ground: a wave
// whose crests lie nearly in the ground's plane is long across it.
//
// None is kept whose wavelength along the ground is the survey's or
// longer, so made roughness has no energy there, in the plane of the
// survey's smoothed normal. Where that normal turns, within a survey cell,
// a wave is read against the plane at each place, and nothing more is
// claimed.
func (size octave) along(point, normal maths.Vec, survey float64) float64 {
	var sum float64
	for _, each := range size.waves {
		sum += math.Cos(each.number.Dot(point)+each.phase) *
			(1 - maths.Smooth(alongFrom*survey, survey,
				alongGround(each.number, normal)))
	}
	return sum * math.Sqrt(2.0/wavesPerOctave)
}

// alongGround is a wave's wavelength along ground of a unit normal,
// metres: its wave number less what of it is along the normal. A wave
// that runs straight along the normal has none, and is infinitely long.
func alongGround(number, normal maths.Vec) float64 {
	across := number.Sub(normal.Scale(number.Dot(normal))).Len()
	if across == 0 {
		return math.Inf(1)
	}
	return 2 * math.Pi / across
}

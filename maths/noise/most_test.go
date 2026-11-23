package noise_test

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/janearc/slayer/maths/noise"
)

// The search for how far past one Gradient goes. The doc says 1.033 was
// found at one point and a million random points never passed 0.98; the
// arithmetic allows two. This climbs from the found point and from
// random starts, and reports the most it finds. It is a measurement, and
// the bound it holds is the one the other tests use.

// climb walks uphill in the size of the noise from a start, stepping
// along each axis either way and taking the first step that rises, and
// shrinking the step when none does. It is the most it reached.
func climb(test reporter, seed uint64, at place) float64 {
	test.Helper()
	step, shrink := picked(test, "climb-step"), picked(test, "climb-shrink")
	best := math.Abs(noise.Gradient(seed, at.x, at.y, at.z))
	for range count(test, "climb-steps") {
		risen := false
		for _, next := range []place{{at.x + step, at.y, at.z},
			{at.x - step, at.y, at.z}, {at.x, at.y + step, at.z},
			{at.x, at.y - step, at.z}, {at.x, at.y, at.z + step},
			{at.x, at.y, at.z - step}} {
			if size := math.Abs(noise.Gradient(seed, next.x, next.y,
				next.z)); size > best {
				at, best, risen = next, size, true
				break
			}
		}
		if !risen {
			step *= shrink
		}
	}
	return best
}

// TestGradientMostFound climbs from the found point and from random
// starts in the box, logs the most it reached, and holds it under the
// bound the other tests use.
func TestGradientMostFound(test *testing.T) {
	most := climb(test, pickedSeed(test, "past-one-seed"), place{
		picked(test, "past-one-x"), picked(test, "past-one-y"),
		picked(test, "past-one-z")})
	row := read(test, sweepsFile).by(test, sweepsFile, "gradient")
	first, second := seed(test, row, "first"), seed(test, row, "second")
	random := rand.New(rand.NewPCG(first, second))
	span, reach := picked(test, "span"), picked(test, "reach")
	for range count(test, "climb-starts") {
		start := place{random.Float64()*span - reach,
			random.Float64()*span - reach, random.Float64()*span - reach}
		reached := climb(test, random.Uint64(), start)
		most = math.Max(most, reached)
	}
	test.Logf("the most Gradient reached was %v", most)
	if !(most <= picked(test, "noise-bound")) {
		test.Errorf("Gradient reached %v, past the bound", most)
	}
}

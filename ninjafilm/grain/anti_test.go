package grain_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/grain"
	"pgregory.net/rapid"
)

// shift is how far a point is moved to see that the noise is another.
var shift = data.Picked("grain-shift")

// pair is a count of two, defined as two.
const pair = 2

// anti-tests: a pinned point moved by a quarter of a cell is not the
// noise pinned for it. The table is of points, not of numbers any point
// may have.
func TestNoPointNearAPinnedPointHasItsNoise(t *testing.T) {
	for _, row := range rows(t, pinnedFile) {
		seed := seedOf(t, row, "seed")
		x, y, z := amount(t, row, "x"), amount(t, row, "y"),
			amount(t, row, "z")
		want := amount(t, row, "want")
		if got := grain.Gradient(seed, x+shift, y+shift, z+shift); got ==
			want {
			t.Errorf("seed %d at %v, %v, %v and a quarter on gives %v",
				seed, x, y, z, got)
		}
	}
}

// anti-tests: the field is not flat. Over a sample of points it takes
// many values, and not one.
func TestTheFieldIsNotOneValue(t *testing.T) {
	rapid.Check(t, fieldTakesManyValues)
}

func fieldTakesManyValues(t *rapid.T) {
	seed := rapid.Uint64().Draw(t, "seed")
	seen := map[float64]bool{}
	for range sample {
		x, y, z := point(t)
		seen[grain.Gradient(seed, x, y, z)] = true
	}
	if len(seen) < sample/pair {
		t.Fatalf("%d points took only %d values", sample, len(seen))
	}
}

// anti-tests: off the lattice the noise is not zero. Across a sample of points
// a quarter of a cell from it, some are not.
func TestOffTheLatticeTheNoiseIsNotAlwaysNought(t *testing.T) {
	rapid.Check(t, offLatticeIsNotNought)
}

func offLatticeIsNotNought(t *rapid.T) {
	seed := rapid.Uint64().Draw(t, "seed")
	lattice := rapid.Int64Range(-int64(farthest), int64(farthest))
	moved := false
	for range sample {
		x, y, z := lattice.Draw(t, "x"), lattice.Draw(t, "y"),
			lattice.Draw(t, "z")
		got := grain.Gradient(seed, float64(x)+shift, float64(y)+shift,
			float64(z)+shift)
		moved = moved || got != none
	}
	if !moved {
		t.Fatalf("the noise was nought at every point a quarter off")
	}
}

// anti-tests: the noise has both signs. Over a sample some points are greater
// than zero and some below, as hills and valleys are.
func TestTheNoiseHasHillsAndValleys(t *testing.T) {
	rapid.Check(t, noiseHasBothSigns)
}

func noiseHasBothSigns(t *rapid.T) {
	seed := rapid.Uint64().Draw(t, "seed")
	var above, below bool
	for range sample * sample {
		x, y, z := point(t)
		got := grain.Gradient(seed, x, y, z)
		above, below = above || got > none, below || got < none
	}
	if !above || !below {
		t.Fatalf("seed %d had hills %v and valleys %v", seed, above, below)
	}
}

// anti-tests: big steps do move the noise. Steps of a quarter of a cell
// move it by more than a step of the small size could.
func TestABigStepMovesTheNoiseFurtherThanASmallOne(t *testing.T) {
	rapid.Check(t, bigStepMovesFurther)
}

func bigStepMovesFurther(t *rapid.T) {
	seed := rapid.Uint64().Draw(t, "seed")
	moved := false
	for range sample {
		x, y, z := point(t)
		here := grain.Gradient(seed, x, y, z)
		over := grain.Gradient(seed, x+shift, y+shift, z+shift)
		moved = moved || math.Abs(over-here) > slopeMost*step
	}
	if !moved {
		t.Fatalf("a step of %v never moved the noise past %v", shift,
			slopeMost*step)
	}
}

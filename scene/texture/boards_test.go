package texture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// boardedTolerance is how close two shares of a colour are to be the same.
const boardedTolerance = 1e-12

// boardedFar is a pixel footprint, in metres, wider than any seam.
const boardedFar = 100.0

// the concrete and stain the board tests are drawn with
var (
	boardConcrete = maths.RGB(0.5, 0.45, 0.4)  // picked: a pale concrete
	boardStain    = maths.RGB(0.1, 0.09, 0.08) // picked: a dark stain
)

// boardedWith is a texture with boards .2 m tall and a seam of the width.
func boardedWith(joint float64) Boarded {
	return Boarded{Concrete: boardConcrete, Stain: boardStain,
		Board: 0.2, Joint: joint, Seed: 1}
}

// darkened is the red of the concrete darkened toward the stain by a share.
func darkened(share float64) float64 {
	return boardConcrete.R*(1-share) + boardStain.R*share
}

// TestBoardedAverageIsWhatTheSeamsAndGrainTake checks the average by hand.
// A seam .012 m wide on a board .2 m tall is a share 2 * .012 / .2 = .12 of
// the surface, taken twice as it has two edges. The seams darken that share
// by a joint's share, and the grain darkens all of it by half its own, so
// the share toward the stain is .12 jointShare + grainShare / 2.
func TestBoardedAverageIsWhatTheSeamsAndGrainTake(t *testing.T) {
	const seams = 0.12
	got := boardedWith(0.012).Average()
	want := darkened(seams*jointShare + grainShare/2)
	if math.Abs(got.R-want) > boardedTolerance {
		t.Errorf("average red %v, want %v", got.R, want)
	}
}

// TestBoardedAverageTakesNoMoreThanAWholeSurfaceOfSeam is the edge of the
// average: seams as wide as the board cover all of it and no more, and
// with no seam only the grain takes any.
func TestBoardedAverageTakesNoMoreThanAWholeSurfaceOfSeam(t *testing.T) {
	wide := boardedWith(0.3).Average()
	if want := darkened(jointShare + grainShare/2); math.Abs(wide.R-want) >
		boardedTolerance {
		t.Errorf("wide seams average red %v, want %v", wide.R, want)
	}
	bare := boardedWith(0).Average()
	if want := darkened(grainShare / 2); math.Abs(bare.R-want) >
		boardedTolerance {
		t.Errorf("no seams average red %v, want %v", bare.R, want)
	}
}

// TestBoardedFarOffIsItsAverage checks that a pixel wider than any seam
// shows the average, wherever it looks, and that a pixel narrower than a
// seam does not.
func TestBoardedFarOffIsItsAverage(t *testing.T) {
	boarded := boardedWith(0.012)
	average := boarded.Average()
	for _, at := range []maths.Vec{maths.V(0.3, 1.7, 2.9),
		maths.V(-4, 0.01, 7), maths.V(0, 0.1, 0)} {
		if far := boarded.Within(at, boardedFar); math.Abs(far.R-average.R) >
			boardedTolerance {
			t.Errorf("at %v far off red %v, want %v", at, far.R, average.R)
		}
	}
	seam := maths.V(0.3, 0.001, 0.3)
	if near := boarded.Within(seam, boarded.Joint/boardedFar); near ==
		average {
		t.Errorf("a seam seen closely is its average %v", near)
	}
}

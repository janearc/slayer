package shape

import (
	"strings"
	"testing"
)

// tubeShareRefusedFile is the table of chord shares a ring cannot be made
// for. At 1e-6 a ring needs about two thousand sides, and at the share of
// the row too-fine about seven thousand, past tubeMostSides.
const tubeShareRefusedFile = "testdata/tube-share-refused.csv"

// tubeShareTube is the tube the tests of this file mesh, from the rows of
// numbers.yaml that say its radius, wall and length.
func tubeShareTube(t *testing.T) Tube {
	t.Helper()
	return Tube{Radius: picked(t, "tube-share-radius"),
		Thick:  picked(t, "tube-share-wall"),
		Height: picked(t, "tube-share-length")}
}

// TestATubeMeshRefusesAShareItCannotMake says that a share that is not a
// number, is not between nought and one, or would need more than
// tubeMostSides sides a ring is refused in words, and quickly: none of them
// is meshed.
func TestATubeMeshRefusesAShareItCannotMake(t *testing.T) {
	tube := tubeShareTube(t)
	for _, each := range readSheet(t, tubeShareRefusedFile) {
		share := cell(t, each, "share")
		made, err := tube.Mesh(share)
		if err == nil || made != nil {
			t.Errorf("share %v: made %v, %v; want a refusal", share, made, err)
			continue
		}
		if !strings.Contains(err.Error(), "chord share") {
			t.Errorf("share %v: error %q does not say what is wrong", share,
				err)
		}
	}
}

// TestATubeMeshMakesAShareAtTheCap is the anti-test of the one above: the
// share just allowed, and the share slayer draws with, are meshed, and the
// first has no more sides than the cap.
func TestATubeMeshMakesAShareAtTheCap(t *testing.T) {
	tube := tubeShareTube(t)
	justAllowed := picked(t, "tube-share-just-allowed")
	for _, share := range []float64{justAllowed, TubeChordShare(), half} {
		if _, err := tube.Mesh(share); err != nil {
			t.Errorf("share %v refused: %v", share, err)
		}
	}
	if sides := tubeSides(justAllowed); sides > tubeMostSides {
		t.Errorf("share %v needs %d sides, over the cap of %d", justAllowed,
			sides, tubeMostSides)
	}
}

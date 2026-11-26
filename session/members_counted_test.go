package session

import (
	"os"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies/scrooge"
)

// sightedStaging is a stage whose camera is the one of lookingNorth.
func sightedStaging(t *testing.T) *staging {
	t.Helper()
	raw, err := os.ReadFile("../sets/mangrove-study.yaml")
	if err != nil {
		t.Fatal(err)
	}
	read := readPositionSet(t, string(raw))
	stage := testStaging()
	stage.recipes = read
	eye := lookingNorth()
	stage.sight = &eye
	return stage
}

// standingAt is a place on the axis the camera looks along, so many
// metres ahead of it (negative is behind it).
func standingAt(ahead float64) maths.Subspace {
	return maths.Subspace{Origin: maths.V(0, 0, -ahead),
		Orientation: maths.Identity, Scale: 1}
}

// closeBy and behind are how far ahead, in metres, a thing near the
// camera stands, and how far behind a thing out of view.
const (
	closeBy = 3.0
	behind  = -50.0
)

// TestAMemberOutOfViewIsMadeAtTheCoarsestRungAndCountedUnseen checks that
// where scrooge is asked a thing behind the camera is made as a hull, the
// coarsest its kind has, and the tally counts it once, as out of view.
func TestAMemberOutOfViewIsMadeAtTheCoarsestRungAndCountedUnseen(
	t *testing.T) {
	t.Setenv(scrooge.Variable, "1")
	stage := sightedStaging(t)
	got, err := stage.memberDetail("plant-profile", "plant-avicennia-marina", "demo", 1, "",
		standingAt(behind), false)
	if err != nil || got != "hull" {
		t.Fatalf("a thing out of view got %q, %v; want hull", got, err)
	}
	if stage.rungs.made["hull"] != 1 || stage.rungs.unseen != 1 {
		t.Errorf("tally %v with %d unseen; want one hull, one unseen",
			stage.rungs.made, stage.rungs.unseen)
	}
}

// TestAMemberCloseByIsMadeAsFineAsTheSetSays is the anti-test of the
// ladder: a thing a few metres from the camera is made at the set's own
// complexity, and counted as seen.
func TestAMemberCloseByIsMadeAsFineAsTheSetSays(t *testing.T) {
	t.Setenv(scrooge.Variable, "1")
	stage := sightedStaging(t)
	got, err := stage.memberDetail("plant-profile", "plant-avicennia-marina", "demo", 1, "",
		standingAt(closeBy), false)
	if err != nil || got != "demo" {
		t.Fatalf("a thing close by got %q, %v; want demo", got, err)
	}
	if stage.rungs.made["demo"] != 1 || stage.rungs.unseen != 0 {
		t.Errorf("tally %v with %d unseen; want one demo, none unseen",
			stage.rungs.made, stage.rungs.unseen)
	}
}

// TestAChoiceOfRungCountsNothingUntilItIsCounted checks that the rung of
// a thing can be known ahead of it without the tally hearing of it, and
// that memberDetail then counts it once.
func TestAChoiceOfRungCountsNothingUntilItIsCounted(t *testing.T) {
	t.Setenv(scrooge.Variable, "1")
	stage := sightedStaging(t)
	chosen, err := stage.rungChosen("plant-profile", "plant-avicennia-marina", "demo", 1, "",
		standingAt(behind), false)
	if err != nil || chosen.rung != "hull" || !chosen.counted ||
		!chosen.unseen {
		t.Fatalf("chose %+v, %v; want an unseen hull to be counted",
			chosen, err)
	}
	if len(stage.rungs.made) != 0 || stage.rungs.unseen != 0 {
		t.Errorf("a choice counted %v with %d unseen", stage.rungs.made,
			stage.rungs.unseen)
	}
	if _, err := stage.memberDetail("plant-profile", "plant-avicennia-marina", "demo", 1, "",
		standingAt(behind), false); err != nil {
		t.Fatal(err)
	}
	if stage.rungs.made["hull"] != 1 {
		t.Errorf("counted %v, want the hull once", stage.rungs.made)
	}
}

// TestAChoiceWhereScroogeIsNotAskedIsTheSetsAndIsNotCounted is the
// anti-test of the choice: unasked, rungChosen gives the set's complexity
// and says nothing is to be counted, so the flag the test above reads is
// seen to be false as well as true.
func TestAChoiceWhereScroogeIsNotAskedIsTheSetsAndIsNotCounted(
	t *testing.T) {
	t.Setenv(scrooge.Variable, "")
	stage := sightedStaging(t)
	chosen, err := stage.rungChosen("plant-profile", "plant-avicennia-marina", "demo", 1, "",
		standingAt(behind), false)
	if err != nil || chosen.rung != "demo" || chosen.counted ||
		chosen.unseen {
		t.Errorf("chose %+v, %v; want the set's demo, not counted", chosen,
			err)
	}
}

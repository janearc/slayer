package manufacture

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// The folded wing of a standing bird against its toes. The frigatebird's
// is the one that hung below them.

// birdFrigatebird is the species whose folded wing is long for its legs.
const birdFrigatebird = "great-frigatebird"

// birdWingPieces are the feathers of the folded wing and the small coverts
// that ride on its bones.
func birdWingPieces(body *birdBody) []birdPiece {
	pieces := body.wingPieces()
	for _, feather := range body.feathers {
		if birdOnWing(feather.group) {
			pieces = append(pieces, birdPiece{bone: feather.name,
				form: body.forms[feather.name]})
		}
	}
	return pieces
}

// birdToePieces are the toes' bones in their forms.
func birdToePieces(body *birdBody) []birdPiece {
	pieces := []birdPiece{}
	for _, piece := range body.legPieces() {
		if strings.HasPrefix(piece.bone, "toe") {
			pieces = append(pieces, piece)
		}
	}
	return pieces
}

// TestAFrigateBirdsFoldedWingIsNotBelowItsToes is the test of the fold:
// standing, the lowest point of the feathers and coverts of the folded
// wing is no lower than the lowest point of the toes.
func TestAFrigateBirdsFoldedWingIsNotBelowItsToes(t *testing.T) {
	body := birdTestBody(t, birdFrigatebird)
	frames := body.bones.Frames(body.standing())
	wingLow, bone := birdLowestOf(body, birdWingPieces(body), frames)
	toeLow, _ := birdLowestOf(body, birdToePieces(body), frames)
	if wingLow < toeLow-birdFloor {
		t.Errorf("the wing hangs to %.4f m on %s, the toes' lowest is %.4f m",
			wingLow, bone, toeLow)
	}
}

// TestAFrigateBirdsToesAreOnTheGround: the bird is not lifted to clear
// its wing. Its toes' lowest point is on the ground, y nought, to a
// millimetre, and the lowest point of the whole bird is within a
// centimetre of it.
func TestAFrigateBirdsToesAreOnTheGround(t *testing.T) {
	body := birdTestBody(t, birdFrigatebird)
	frames := body.bones.Frames(body.standing())
	toeLow, _ := birdLowestOf(body, birdToePieces(body), frames)
	if math.Abs(toeLow) > birdMillimetre {
		t.Errorf("the toes' lowest is %.4f m, not on the ground", toeLow)
	}
	_, box, err := body.outlined(body.skinPieces(), frames, maths.Local())
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(box.Min.Y) > .01 {
		t.Errorf("the lowest point of the bird is %.4f m", box.Min.Y)
	}
}

// TestRaisingTheWingDoesNotMoveTheRestOfTheBird is an anti-test: the
// frigatebird's trunk, tail and toes are where they were with the wing
// unraised, and only its wing has moved, upward, the tip of the longest
// primary by more than three centimetres, a little under what it hung.
func TestRaisingTheWingDoesNotMoveTheRestOfTheBird(t *testing.T) {
	const gain = .03 // the tip hung .035 m below the toes; picked below that
	body := birdTestBody(t, birdFrigatebird)
	raised := body.bones.Frames(body.standing())
	flat := body.bones.Frames(body.standingRaised(0))
	for _, name := range []string{"synsacrum", "pygostyle",
		"tarsometatarsus-right", "toe3-1-right", "rectrix1", "cranium"} {
		at, _ := body.bones.Find(name)
		head := body.bones.Bone(at).Head()
		if raised[at].Apply(head).Sub(flat[at].Apply(head)).Len() > birdFloor {
			t.Errorf("%s moved when the wing was raised", name)
		}
	}
	at, _ := body.bones.Find("p9-right")
	tip := body.bones.Bone(at).Tail()
	if up := raised[at].Apply(tip).Y - flat[at].Apply(tip).Y; up < gain {
		t.Errorf("the tip of p9 rose %.4f m, want over %.2f", up, gain)
	}
}

// birdHangs is whether the folded wing of body, unraised, has a point
// lower than the lowest of its toes.
func birdHangs(body *birdBody) bool {
	frames := body.bones.Frames(body.standingRaised(0))
	wingLow, _ := birdLowestOf(body, birdWingPieces(body), frames)
	toeLow, _ := birdLowestOf(body, birdToePieces(body), frames)
	return wingLow < toeLow
}

// TestOnlyABirdWhoseWingWouldHangIsRaised is an anti-test. For each species
// the wing is raised if and only if, unraised, its lowest point is below
// the toes'. A bird whose wing clears its toes stands exactly as it did.
func TestOnlyABirdWhoseWingWouldHangIsRaised(t *testing.T) {
	birdEachBody(t, func(key string, body *birdBody) {
		pose := body.standingRaised(0)
		hangs := birdHangs(body)
		if raised := body.wingRise(pose) > 0; raised != hangs {
			t.Errorf("%s: raised %v but the wing hangs %v", key, raised, hangs)
		}
		if !hangs {
			birdSamePose(t, key, body, pose, body.standing())
		}
	})
}

// TestTheFrigateBirdsWingHangsUnraisedAndTheHeronsDoesNot is the other
// way to the test above: the frigatebird, on legs of a few centimetres,
// has a wing that hangs below its toes without the rise, and the grey
// heron, on a tarsus about six times as long, has one that does not.
func TestTheFrigateBirdsWingHangsUnraisedAndTheHeronsDoesNot(t *testing.T) {
	for key, want := range map[string]bool{birdFrigatebird: true,
		"grey-heron": false} {
		if got := birdHangs(birdTestBody(t, key)); got != want {
			t.Errorf("%s: unraised, its wing hangs %v, want %v", key, got,
				want)
		}
	}
}

// birdSamePose fails if two poses of body put any bone's ends in different
// places.
func birdSamePose(t *testing.T, key string, body *birdBody, one,
	two skeleton.Pose) {
	t.Helper()
	first, second := body.bones.Frames(one), body.bones.Frames(two)
	for i := 0; i < body.bones.Len(); i++ {
		bone := body.bones.Bone(i)
		for _, end := range []maths.Vec{bone.Head(), bone.Tail()} {
			if first[i].Apply(end).Sub(second[i].Apply(end)).Len() >
				birdFloor {
				t.Errorf("%s: %s moved though its wing clears the toes", key,
					bone.Name())
				return
			}
		}
	}
}

package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/bioforms/skeleton"
)

// The ring of bare skin round each eye. In the head's pieces each eye is
// followed by its ring.

// birdRingsOf are the pieces of the head that are rings: each piece that
// follows an eye.
func birdRingsOf(body *birdBody) []birdPiece {
	rings := []birdPiece{}
	pieces := body.headPieces()
	for i, piece := range pieces {
		if piece.region == "eye" {
			rings = append(rings, pieces[i+1])
		}
	}
	return rings
}

// TestEveryBirdHasAnEyeRingOnEachEye: the ring follows its eye in the
// head's pieces, is carried by the skull and coloured as the bare face,
// and there are two eyes.
func TestEveryBirdHasAnEyeRingOnEachEye(t *testing.T) {
	birdEachBody(t, func(key string, body *birdBody) {
		rings := birdRingsOf(body)
		if len(rings) != 2 {
			t.Fatalf("%s: %d eye rings, want 2", key, len(rings))
		}
		for _, ring := range rings {
			if ring.region != "face" || ring.bone != "cranium" {
				t.Errorf("%s: a ring on %s as %s", key, ring.bone, ring.region)
			}
		}
	})
}

// TestABirdsEyeRingIsAroundItsEyeAndLiesOutside works the ring by hand. The
// eye is a ball of radius .09 of the head's length; the ring starts at the
// ball's middle, reaches out along the side by .5 of that radius, and is
// 1.8 of it across: wider than the ball, so that a ring shows round it.
func TestABirdsEyeRingIsAroundItsEyeAndLiesOutside(t *testing.T) {
	birdEachBody(t, func(key string, body *birdBody) {
		radius := birdEyeShare * body.measures.head
		pieces := body.headPieces()
		for i, piece := range pieces {
			if piece.region != "eye" {
				continue
			}
			ball := piece.form.Middle()
			middle := ball[0].Add(ball[1]).Scale(.5)
			ring, ok := pieces[i+1].form.(skeleton.Profile)
			if !ok {
				t.Fatalf("%s: a ring that is no profile", key)
			}
			line := ring.Middle()
			if line[0].Sub(middle).Len() > birdFloor {
				t.Errorf("%s: ring starts %v, the eye's middle is %v", key,
					line[0], middle)
			}
			out := math.Copysign(1, middle.Z)
			want := middle.Z + out*.5*radius
			if math.Abs(line[1].Z-want) > birdFloor {
				t.Errorf("%s: ring ends at z %.4f, want %.4f", key, line[1].Z,
					want)
			}
			across := birdEyeRing * radius
			wide, deep := ring.Size(.5)
			if math.Abs(wide-across) > birdFloor ||
				math.Abs(deep-across) > birdFloor || across <= radius {
				t.Errorf("%s: ring %.4f by %.4f, want %.4f, the eye's "+
					"radius being %.4f", key, wide, deep, across, radius)
			}
		}
	})
}

// TestNoBirdEyeRingIsASpeck: each ring has some size along every axis.
func TestNoBirdEyeRingIsASpeck(t *testing.T) {
	birdEachBody(t, func(key string, body *birdBody) {
		frames := body.bones.Frames(skeleton.Pose{})
		for i, ring := range birdRingsOf(body) {
			if !birdSolid(birdCorners(body, ring, frames)) {
				t.Errorf("%s: ring %d is a line or a speck", key, i)
			}
		}
	})
}

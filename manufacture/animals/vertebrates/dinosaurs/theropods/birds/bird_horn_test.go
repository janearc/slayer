package birds

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/bioforms/skeleton"
)

// The claws and the scutes of the tarsus: the horny parts a bird gains on
// its feet and legs. Each is checked at rest, where the bones' own
// numbers can be worked by hand.

// birdMillimetre is a thousandth of a metre: how far two parts may differ
// in height where their true surfaces meet. A polygon of ten sides misses
// a circle by under five hundredths of its radius, and a toe's radius is
// under four millimetres here.
const birdMillimetre = 1e-3

// birdLastBone is the name of the last bone of toe n, counted from one,
// on side.
func birdLastBone(n int, side string) string {
	return fmt.Sprintf("toe%d-%d-%s", n, birdToes[n-1].phalanges, side)
}

// TestEveryBirdHasAClawOnEachToe: four toes on each of two feet, so eight
// claws, each carried by the last bone of its toe and coloured as the
// foot.
func TestEveryBirdHasAClawOnEachToe(t *testing.T) {
	birdEachBody(t, func(key string, body *birdBody) {
		claws := body.clawPieces()
		if want := 2 * len(birdToes); len(claws) != want {
			t.Fatalf("%s: %d claws, want %d", key, len(claws), want)
		}
		i := 0
		for _, side := range birdSides {
			for n := range birdToes {
				want := birdLastBone(n+1, side.name)
				if claws[i].bone != want || claws[i].region != "feet" {
					t.Errorf("%s: claw %d on %s as %s, want %s as feet", key,
						i, claws[i].bone, claws[i].region, want)
				}
				i++
			}
		}
	})
}

// TestABirdsClawCoversTheEndOfItsToeAndReachesPastIt works a claw by hand. The
// last bone of a toe is half the thickness of a small bone, since its
// thickness is 1 - .5 k / k of that, so its skin is 1.5 x .003 x length
// / 2. The claw begins half way along the bone and its tip is .6 of the
// bone's length past the bone's end, and .8 of a skin below the bone's
// axis.
func TestABirdsClawCoversTheEndOfItsToeAndReachesPastIt(t *testing.T) {
	birdEachBody(t, func(key string, body *birdBody) {
		skin := birdLegSkin * birdSmallBone * body.measures.length / 2
		for i, claw := range body.clawPieces() {
			head, tail := birdEnds(t, body, claw.bone)
			step := tail.Sub(head)
			root := head.Add(step.Scale(.5))
			tip := tail.Add(step.Scale(.6)).Add(birdUp.Scale(-.8 * skin))
			line := claw.form.Middle()
			if line[0].Sub(root).Len() > birdFloor ||
				line[1].Sub(tip).Len() > birdFloor {
				t.Errorf("%s claw %d: runs %v to %v, want %v to %v", key, i,
					line[0], line[1], root, tip)
			}
			if line[1].Sub(tail).Dot(step) <= 0 {
				t.Errorf("%s claw %d: does not reach past its toe", key, i)
			}
			if line[1].Y > tail.Y {
				t.Errorf("%s claw %d: hooks up, not down", key, i)
			}
		}
	})
}

// TestABirdsClawDoesNotStandBelowTheToes is an anti-test: standing, the lowest
// point of the claws is no lower than the lowest point of the toes, to a
// millimetre, so the claws do not dig in.
func TestABirdsClawDoesNotStandBelowTheToes(t *testing.T) {
	birdEachBody(t, func(key string, body *birdBody) {
		frames := body.bones.Frames(body.standing())
		toes := []birdPiece{}
		for _, piece := range body.legPieces() {
			if strings.HasPrefix(piece.bone, "toe") {
				toes = append(toes, piece)
			}
		}
		toeLow, _ := birdLowestOf(body, toes, frames)
		clawLow, _ := birdLowestOf(body, body.clawPieces(), frames)
		if clawLow < toeLow-birdMillimetre {
			t.Errorf("%s: claws reach %.4f m, the toes %.4f m", key, clawLow,
				toeLow)
		}
	})
}

// TestEveryBirdHasSixScutesDownEachTarsus: twelve scutes, six on each
// tarsometatarsus, coloured as the legs.
func TestEveryBirdHasSixScutesDownEachTarsus(t *testing.T) {
	birdEachBody(t, func(key string, body *birdBody) {
		scutes := body.scutePieces()
		if want := 2 * birdScutes; len(scutes) != want {
			t.Fatalf("%s: %d scutes, want %d", key, len(scutes), want)
		}
		for i, scute := range scutes {
			side := birdSides[i/birdScutes].name
			if scute.bone != "tarsometatarsus-"+side || scute.region != "legs" {
				t.Errorf("%s: scute %d on %s as %s", key, i, scute.bone,
					scute.region)
			}
		}
	})
}

// TestABirdsScuteLiesInItsPlaceOnTheFrontOfTheTarsus works each one by hand.
// The i-th of six is centred (i + .5) / 6 of the way from the ankle to
// the foot, on the tarsus's own line across the bird, and its middle is
// out in front of the bone, +x, by the radius of the bone there: the
// skin, 1.5 x .006 x the bird's length, times .7 plus .3 of (2s - 1)^4
// for s that share along.
func TestABirdsScuteLiesInItsPlaceOnTheFrontOfTheTarsus(t *testing.T) {
	birdEachBody(t, func(key string, body *birdBody) {
		skin := birdLegSkin * birdLimbBone * body.measures.length
		for i, scute := range body.scutePieces() {
			ankle, foot := birdEnds(t, body, scute.bone)
			share := (float64(i%birdScutes) + .5) / birdScutes
			middle := scute.form.Middle()[2]
			if want := ankle.Y + (foot.Y-ankle.Y)*share; math.Abs(middle.Y-
				want) > birdFloor {
				t.Errorf("%s scute %d: at %.4f m up, want %.4f", key, i,
					middle.Y, want)
			}
			if math.Abs(middle.Z-ankle.Z) > birdFloor {
				t.Errorf("%s scute %d: off the leg's line by %.4f m", key, i,
					middle.Z-ankle.Z)
			}
			radius := skin * (.7 + .3*math.Pow(2*share-1, 4))
			if math.Abs(middle.X-ankle.X-radius) > birdFloor {
				t.Errorf("%s scute %d: %.4f m in front, want %.4f", key, i,
					middle.X-ankle.X, radius)
			}
		}
	})
}

// TestABirdsScuteWrapsTheFrontAndNotTheBack is an anti-test. Each scute's ends
// lie to either side of its middle, across the leg, by the sine of fifty
// degrees of the radius, .764, and each end is nearer the front than the
// leg's side is: ahead of the bone's line.
func TestABirdsScuteWrapsTheFrontAndNotTheBack(t *testing.T) {
	birdEachBody(t, func(key string, body *birdBody) {
		for i, scute := range body.scutePieces() {
			ankle, _ := birdEnds(t, body, scute.bone)
			line := scute.form.Middle()
			first, middle, last := line[0], line[2], line[4]
			if first.Z >= middle.Z || last.Z <= middle.Z {
				t.Errorf("%s scute %d: does not span the leg", key, i)
			}
			if first.X <= ankle.X || last.X <= ankle.X {
				t.Errorf("%s scute %d: wraps round behind the leg", key, i)
			}
		}
	})
}

// TestABirdsScutesDoNotOverlapEachOther is an anti-test: each scute's corners
// lie within its own sixth of the tarsus's length, so no two touch.
func TestABirdsScutesDoNotOverlapEachOther(t *testing.T) {
	birdEachBody(t, func(key string, body *birdBody) {
		frames := body.bones.Frames(skeleton.Pose{})
		for i, scute := range body.scutePieces() {
			ankle, foot := birdEnds(t, body, scute.bone)
			room := (ankle.Y - foot.Y) / birdScutes
			top := ankle.Y - float64(i%birdScutes)*room
			for _, corner := range birdCorners(body, scute, frames) {
				if corner.Y > top+birdFloor || corner.Y < top-room-birdFloor {
					t.Errorf("%s scute %d: a corner at %.4f m, its room is "+
						"%.4f to %.4f", key, i, corner.Y, top-room, top)
					break
				}
			}
		}
	})
}

// TestNoBirdClawOrScuteIsASpeck: each has some size along every axis.
func TestNoBirdClawOrScuteIsASpeck(t *testing.T) {
	birdEachBody(t, func(key string, body *birdBody) {
		frames := body.bones.Frames(skeleton.Pose{})
		pieces := append(body.clawPieces(), body.scutePieces()...)
		for i, piece := range pieces {
			if !birdSolid(birdCorners(body, piece, frames)) {
				t.Errorf("%s: part %d on %s is a line or a speck", key, i,
					piece.bone)
			}
		}
	})
}

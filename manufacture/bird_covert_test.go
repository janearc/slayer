package manufacture

import (
	"fmt"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The lesser coverts and the scapulars. A lesser covert is the third row
// of coverts over a secondary or a tertial; the scapulars are a row along
// the shoulder blade.

// birdArmFeathers are the names, less the side, of the flight feathers of
// the arm: the secondaries and then the tertials.
func birdArmFeathers(m birdMeasures) []string {
	names := []string{}
	for j := 1; j <= m.secondaries; j++ {
		names = append(names, fmt.Sprintf("s%d", j))
	}
	for k := 1; k <= birdTertials; k++ {
		names = append(names, fmt.Sprintf("t%d", k))
	}
	return names
}

// TestEveryBirdHasALesserCovertOverEachFlightFeatherOfTheArm: a bird has
// one for each secondary and each tertial on each side, so
// 2 x (secondaries + 4) of them, the secondaries coming from its row.
func TestEveryBirdHasALesserCovertOverEachFlightFeatherOfTheArm(
	t *testing.T) {
	birdEachBody(t, func(key string, body *birdBody) {
		want := 2 * (body.measures.secondaries + birdTertials)
		if got := birdBonesWith(body, birdLesser+"-", true); got != want {
			t.Errorf("%s: %d lesser coverts, want %d", key, got, want)
		}
		for _, side := range birdSides {
			for _, flight := range birdArmFeathers(body.measures) {
				name := fmt.Sprintf("%s-%s-%s", birdLesser, flight, side.name)
				if _, ok := body.bones.Find(name); !ok {
					t.Errorf("%s: no %s", key, name)
				}
			}
		}
	})
}

// TestABirdsLesserCovertLiesOverItsFlightFeather works the place by hand. The
// covert is the third row, so its root is lifted 3 x .004 of the bird's
// length above its flight feather's root and forward of it twice that;
// it is .18 of the flight feather's length and points the same way.
func TestABirdsLesserCovertLiesOverItsFlightFeather(t *testing.T) {
	birdEachBody(t, func(key string, body *birdBody) {
		lift := birdLesserRow * birdCovertLift * body.measures.length
		offset := maths.V(birdCovertForward*lift, lift, 0)
		for _, side := range birdSides {
			for _, flight := range birdArmFeathers(body.measures) {
				root, tip := birdEnds(t, body, flight+"-"+side.name)
				head, end := birdEnds(t, body, fmt.Sprintf("%s-%s-%s",
					birdLesser, flight, side.name))
				if head.Sub(root).Sub(offset).Len() > birdFloor {
					t.Errorf("%s %s %s: root %v from the feather's, want %v",
						key, flight, side.name, head.Sub(root), offset)
				}
				want := birdLesserShare * tip.Sub(root).Len()
				if math.Abs(end.Sub(head).Len()-want) > birdFloor {
					t.Errorf("%s %s: length %.4f, want %.4f", key, flight,
						end.Sub(head).Len(), want)
				}
				if end.Sub(head).Unit().Sub(tip.Sub(root).Unit()).Len() >
					birdFloor {
					t.Errorf("%s %s: not parallel to its flight feather",
						key, flight)
				}
			}
		}
	})
}

// TestABirdsLesserCovertIsOnTheBoneOfTheArm: a secondary's covert hangs on the
// ulna and a tertial's on the humerus, and its root lies between that
// bone's ends across the bird. None is on the hand, so none is rooted
// further out than the wrist.
func TestABirdsLesserCovertIsOnTheBoneOfTheArm(t *testing.T) {
	birdEachBody(t, func(key string, body *birdBody) {
		for _, side := range birdSides {
			_, wrist := birdEnds(t, body, "ulna-"+side.name)
			for _, flight := range birdArmFeathers(body.measures) {
				carrier := "ulna-" + side.name
				if flight[0] == 't' {
					carrier = "humerus-" + side.name
				}
				name := fmt.Sprintf("%s-%s-%s", birdLesser, flight, side.name)
				if got := birdParentOf(t, body, name); got != carrier {
					t.Errorf("%s %s: on %s, want %s", key, name, got, carrier)
				}
				from, to := birdEnds(t, body, carrier)
				head, _ := birdEnds(t, body, name)
				low := math.Min(math.Abs(from.Z), math.Abs(to.Z))
				high := math.Max(math.Abs(from.Z), math.Abs(to.Z))
				if across := math.Abs(head.Z); across < low-birdFloor ||
					across > high+birdFloor {
					t.Errorf("%s %s: %.3f m out, the bone spans %.3f to %.3f",
						key, name, across, low, high)
				}
				if math.Abs(head.Z) > math.Abs(wrist.Z)+birdFloor {
					t.Errorf("%s %s: beyond the wrist", key, name)
				}
			}
		}
	})
}

// TestABirdsRowsOfCovertsGrowSmallerAndFurtherForward: over each flight
// feather the greater covert is the longest and lies nearest, then the
// median, then the lesser, which is the shortest and lies furthest
// forward: lengths .55, .32 and .18 of the feather's, and roots 1, 2 and
// 3 lifts above it.
func TestABirdsRowsOfCovertsGrowSmallerAndFurtherForward(t *testing.T) {
	birdEachBody(t, func(key string, body *birdBody) {
		rows := []string{"", birdGreater + "-", birdMedian + "-",
			birdLesser + "-"}
		for _, flight := range birdArmFeathers(body.measures) {
			lastLength, lastForward := math.Inf(1), math.Inf(-1)
			for _, row := range rows {
				head, tail := birdEnds(t, body, row+flight+"-right")
				if length := tail.Sub(head).Len(); length >= lastLength {
					t.Errorf("%s %s%s: %.4f m, not shorter than the row "+
						"below", key, row, flight, length)
				} else {
					lastLength = length
				}
				if head.X <= lastForward {
					t.Errorf("%s %s%s: not forward of the row below", key,
						row, flight)
				}
				lastForward = head.X
			}
		}
	})
}

// TestABirdsLesserCovertsFollowTheRowsSecondaries is an anti-test: the count is
// not fixed. A bird of nine secondaries has 2 x (9 + 4) = 26, one of
// eleven has 2 x (11 + 4) = 30.
func TestABirdsLesserCovertsFollowTheRowsSecondaries(t *testing.T) {
	for secondaries, want := range map[float64]int{9: 26, 11: 30} {
		body := birdWorkedUp(t, "grey-heron", "secondaries", secondaries)
		if got := birdBonesWith(body, birdLesser+"-", true); got != want {
			t.Errorf("%v secondaries: %d lesser coverts, want %d",
				secondaries, got, want)
		}
	}
}

// TestEveryBirdHasItsScapulars: six to a side, twelve in all.
func TestEveryBirdHasItsScapulars(t *testing.T) {
	birdEachBody(t, func(key string, body *birdBody) {
		want := 2 * birdScapulars
		if got := birdBonesWith(body, birdScapular, false); got != want {
			t.Errorf("%s: %d scapulars, want %d", key, got, want)
		}
	})
}

// TestABirdsScapularIsRootedAlongTheShoulderBlade works the root by hand: the
// i-th of six is i/5 of the way from the blade's head to its tail, and
// lifted .004 of the bird's length so it lies on the skin, not in it.
func TestABirdsScapularIsRootedAlongTheShoulderBlade(t *testing.T) {
	birdEachBody(t, func(key string, body *birdBody) {
		lift := maths.V(0, birdCovertLift*body.measures.length, 0)
		for _, side := range birdSides {
			from, to := birdEnds(t, body, "scapula-"+side.name)
			for i := 1; i <= birdScapulars; i++ {
				share := float64(i-1) / float64(birdScapulars-1)
				want := from.Add(to.Sub(from).Scale(share)).Add(lift)
				name := fmt.Sprintf("%s%d-%s", birdScapular, i, side.name)
				head, _ := birdEnds(t, body, name)
				if head.Sub(want).Len() > birdFloor {
					t.Errorf("%s %s: root %v, want %v", key, name, head, want)
				}
				if got := birdParentOf(t, body, name); got !=
					"scapula-"+side.name {
					t.Errorf("%s %s: on %s, want the shoulder blade", key,
						name, got)
				}
			}
		}
	})
}

// TestABirdsScapularLiesBackOverTheBackAndNotOffTheBody is an anti-test. Each
// points back, toward the tail, its length is .7 of a secondary's, and its
// tip lies no further out across the bird than the shoulder: the
// shoulder's out share, .8 of the trunk's half width, so none reaches over
// the wing.
func TestABirdsScapularLiesBackOverTheBackAndNotOffTheBody(t *testing.T) {
	birdEachBody(t, func(key string, body *birdBody) {
		m := body.measures
		reach := birdShoulderOut * m.width / 2
		length := birdScapularShare * m.secondary * body.longestPrimary()
		for _, side := range birdSides {
			for i := 1; i <= birdScapulars; i++ {
				name := fmt.Sprintf("%s%d-%s", birdScapular, i, side.name)
				head, tip := birdEnds(t, body, name)
				if tip.X >= head.X {
					t.Errorf("%s %s: does not point back", key, name)
				}
				if math.Abs(tip.Sub(head).Len()-length) > birdFloor {
					t.Errorf("%s %s: length %.4f, want %.4f", key, name,
						tip.Sub(head).Len(), length)
				}
				if math.Abs(head.Z) > reach+birdFloor ||
					math.Abs(tip.Z) > reach+birdFloor {
					t.Errorf("%s %s: beyond the shoulder, %.3f m out", key,
						name, math.Abs(tip.Z))
				}
			}
		}
	})
}

// TestABirdsScapularsAreColouredAsTheBackAndCovertsAsTheWing: the scapulars are
// the back's feathers and a lesser covert is the wing's.
func TestABirdsScapularsAreColouredAsTheBackAndCovertsAsTheWing(t *testing.T) {
	if got := birdRegionOf(birdScapular); got != "back" {
		t.Errorf("scapular coloured as %s", got)
	}
	if got := birdRegionOf(birdLesser); got != "wing" {
		t.Errorf("lesser covert coloured as %s", got)
	}
}

// TestNoBirdCovertOrScapularIsASpeck: each has some size along every axis.
func TestNoBirdCovertOrScapularIsASpeck(t *testing.T) {
	birdEachBody(t, func(key string, body *birdBody) {
		frames := body.bones.Frames(body.pose(birdBones))
		for _, feather := range body.feathers {
			if feather.group != birdLesser && feather.group != birdScapular {
				continue
			}
			piece := birdPiece{bone: feather.name,
				form: body.forms[feather.name]}
			if !birdSolid(birdCorners(body, piece, frames)) {
				t.Errorf("%s %s: a line or a speck", key, feather.name)
			}
		}
	})
}

package skeleton

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// TestTurnsAtParentAndChildCompose turns the parent a quarter about z
// at the origin and the child a quarter about y at its head (1, 0, 0).
// By hand: the child's tip (2, 0, 0) turns about y to (1, 0, -1), then
// the parent's turn takes (x, y, z) to (-y, x, z), so (0, 1, -1). The
// other order of the turns, or a shift not carried by the parent, puts
// it elsewhere.
func TestTurnsAtParentAndChildCompose(test *testing.T) {
	bones := testArm(test)
	upperTurn := NewTurn(maths.V(0, 0, 1), math.Pi/2)
	lowerTurn := NewTurn(maths.V(0, 1, 0), math.Pi/2)
	pose := Pose{}.With("upper", upperTurn).With("lower", lowerTurn)
	tip := bones.Frames(pose)[1].Apply(maths.V(2, 0, 0))
	if !tip.Near(maths.V(0, 1, -1), within) {
		test.Fatalf("the child's tip went to %v, not (0, 1, -1)", tip)
	}
}

func TestANewFrameTurnsThenShifts(test *testing.T) {
	quarter := maths.Rotation(maths.V(0, 0, 1), math.Pi/2)
	frame := NewFrame(quarter, maths.V(5, 0, 0))
	if got := frame.Apply(maths.V(1, 0, 0)); !got.Near(maths.V(5, 1, 0),
		within) {
		test.Fatalf("(1, 0, 0) went to %v", got)
	}
	if got := frame.Direction(maths.V(1, 0, 0)); !got.Near(
		maths.V(0, 1, 0), within) {
		test.Fatalf("the direction went to %v; a shift moved it", got)
	}
}

func TestAPoseOfKnownJointsHasNoUnknown(test *testing.T) {
	bones := testArm(test)
	turn := NewTurn(maths.V(0, 0, 1), 1)
	pose := Pose{}.With("lower", turn)
	if unknown := bones.Unknown(pose); len(unknown) != 0 {
		test.Fatalf("unknown %v", unknown)
	}
}

// Anti-test: a turn at a joint the skeleton lacks is named.

func TestAPoseOfAMistypedJointIsNamed(test *testing.T) {
	bones := testArm(test)
	turn := NewTurn(maths.V(0, 0, 1), 1)
	pose := Pose{}.With("lowre", turn)
	unknown := bones.Unknown(pose)
	if len(unknown) != 1 || unknown[0] != "lowre" {
		test.Fatalf("unknown %v, want [lowre]", unknown)
	}
}

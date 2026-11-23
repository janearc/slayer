package manufacture

import (
	"errors"
	"strings"
	"testing"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// orcaTestCarry is rest point p carried in pose by the bones it is
// bound to.
func orcaTestCarry(test *testing.T, pose skeleton.Pose,
	p maths.Vec,
) maths.Vec {
	test.Helper()
	bones := orcaTestSkeleton(test)
	frames := bones.Frames(pose)
	axis := orcaAxis(bones)
	return skeleton.Carry(p, frames,
		orcaBind(bones, axis, p))
}

func TestEveryPointOfTheSkinIsBound(test *testing.T) {
	binding, err := orcaBinding()
	if err != nil {
		test.Fatal(err)
	}
	for i, hold := range binding {
		total := 0.
		for _, each := range hold {
			total += each.Weight()
		}
		if total <= 0 {
			test.Fatalf("point %d is bound to no bone", i)
		}
	}
}

func TestTheFlipperTipFollowsTheHumerus(test *testing.T) {
	bones := orcaTestSkeleton(test)
	tip := maths.V(orcaFlipperRootX, orcaFlipperRootY, orcaFlipperRootZ).
		Add(orcaFlipperSpan.Scale(orcaFlipperAt + orcaFlipperReach/2))
	hold := orcaBind(bones, orcaAxis(bones), tip)
	humerus := orcaMust(bones, "right-humerus")
	last := hold[len(hold)-1]
	if last.Bone() != humerus || last.Weight() < .99 {
		test.Fatalf("the flipper's tip is held by %v", hold)
	}
}

func TestTheSpyHopRaisesTheHeadAboveTheFlukes(test *testing.T) {
	pose := orcaSpyHopPose(orcaTestSkeleton(test))
	snout := orcaTestCarry(test, pose, maths.V(3, 0, 0))
	flukes := orcaTestCarry(test, pose, maths.V(-2.8, 0, 0))
	if snout.Y-flukes.Y < 4 {
		test.Fatalf("snout at %v, flukes at %v: not upright", snout, flukes)
	}
}

func TestTheSpyHopTurnsTheFlippersOut(test *testing.T) {
	pose := orcaSpyHopPose(orcaTestSkeleton(test))
	upright := skeleton.Pose{}.Carried(skeleton.TurnAbout(orcaSpyHopPivot,
		orcaAcross, orcaSpyHopTilt))
	tip := maths.V(orcaFlipperRootX, orcaFlipperRootY, orcaFlipperRootZ).
		Add(orcaFlipperSpan.Scale(orcaFlipperAt + orcaFlipperReach/2))
	out := orcaTestCarry(test, pose, tip)
	still := orcaTestCarry(test, upright, tip)
	if out.Z <= still.Z {
		test.Fatalf("the right flipper did not turn out: %v, %v", out, still)
	}
}

// Anti-tests: what a pose must not do.

func TestTheRestPoseMovesNoSkin(test *testing.T) {
	for _, p := range orcaRest().points[:500] {
		if moved := orcaTestCarry(test, skeleton.Pose{}, p); !moved.Near(p,
			1e-9) {
			test.Fatalf("%v moved to %v at rest", p, moved)
		}
	}
}

func TestTheSwimDoesNotMoveTheSkull(test *testing.T) {
	bones := orcaTestSkeleton(test)
	frames := bones.Frames(orcaSwimPose(bones, .8))
	cranium := orcaMust(bones, "cranium")
	tip := bones.Bone(cranium).Tail()
	if !frames[cranium].Apply(tip).Near(tip, 1e-9) {
		test.Fatal("the swim moved the skull")
	}
}

// orcaTestMoved is how far frames carry the tail of the bone named name.
func orcaTestMoved(bones *skeleton.Skeleton, frames []skeleton.Frame,
	name string) float64 {
	at := orcaMust(bones, name)
	tail := bones.Bone(at).Tail()
	return frames[at].Apply(tail).Sub(tail).Len()
}

// TestTheSwimBendsOnlyBehindTheChest checks what the swim pose decides:
// the thoracics, which lie ahead of where the bending begins, stay
// where they are, and the last caudal's tail moves further than the
// first lumbar's, as the stroke grows toward the flukes.
func TestTheSwimBendsOnlyBehindTheChest(test *testing.T) {
	bones := orcaTestSkeleton(test)
	frames := bones.Frames(orcaSwimPose(bones, 2.1))
	if d := orcaTestMoved(bones, frames, "T10"); !(d < 1e-9) {
		test.Fatalf("the last thoracic moved %v", d)
	}
	lumbar := orcaTestMoved(bones, frames, "L1")
	caudal := orcaTestMoved(bones, frames, "Ca24")
	if !(caudal > lumbar) {
		test.Fatalf("the last caudal moved %v, the first lumbar %v", caudal,
			lumbar)
	}
}

// Anti-test: a pose that names a joint the skeleton lacks is refused,
// naming it, and is not drawn as if the joint had turned.

func TestAPoseOfAMissingJointIsRefused(test *testing.T) {
	bones := orcaTestSkeleton(test)
	pose := orcaSpyHopPose(bones).With("right-humerous",
		skeleton.NewTurn(orcaAlong, 1))
	_, err := orcaPosed(bones, pose, maths.Local())
	if !errors.Is(err, errOrcaPose) ||
		!strings.Contains(err.Error(), "right-humerous") {
		test.Fatalf("a mistyped joint gave %v", err)
	}
}

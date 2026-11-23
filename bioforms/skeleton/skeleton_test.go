package skeleton

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// within is how near two places are for a test to call them one.
const within = 1e-9

// testArm is a root bone and a child along x, each one long.
func testArm(test *testing.T) *Skeleton {
	test.Helper()
	bones := New()
	if err := bones.Add("upper", "", maths.V(0, 0, 0), maths.V(1, 0, 0),
		.1, .1); err != nil {
		test.Fatal(err)
	}
	if err := bones.Add("lower", "upper", maths.V(1, 0, 0),
		maths.V(2, 0, 0), .1, .1); err != nil {
		test.Fatal(err)
	}
	return bones
}

func TestTheRestPoseMovesNothing(test *testing.T) {
	bones := testArm(test)
	for _, frame := range bones.Frames(Pose{}) {
		p := maths.V(.3, .2, -.4)
		if !frame.Apply(p).Near(p, within) {
			test.Fatalf("rest moved %v to %v", p, frame.Apply(p))
		}
	}
}

func TestATurnMovesTheChildAndNotTheParent(test *testing.T) {
	bones := testArm(test)
	turn := NewTurn(maths.V(0, 0, 1), math.Pi/2)
	pose := Pose{}.With("lower", turn)
	frames := bones.Frames(pose)
	tip := frames[1].Apply(maths.V(2, 0, 0))
	if !tip.Near(maths.V(1, 1, 0), within) {
		test.Fatalf("the child's tip went to %v", tip)
	}
	if !frames[0].Apply(maths.V(1, 0, 0)).Near(maths.V(1, 0, 0), within) {
		test.Fatal("the parent moved")
	}
}

func TestAParentsTurnCarriesTheChild(test *testing.T) {
	bones := testArm(test)
	turn := NewTurn(maths.V(0, 0, 1), math.Pi/2)
	pose := Pose{}.With("upper", turn)
	tip := bones.Frames(pose)[1].Apply(maths.V(2, 0, 0))
	if !tip.Near(maths.V(0, 2, 0), within) {
		test.Fatalf("the child's tip went to %v", tip)
	}
}

func TestATurnKeepsABonesLength(test *testing.T) {
	bones := testArm(test)
	upperTurn := NewTurn(maths.V(1, 2, 3), .7)
	lowerTurn := NewTurn(maths.V(-2, 1, 0), 1.1)
	pose := Pose{}.With("upper", upperTurn).With("lower", lowerTurn)
	frames := bones.Frames(pose)
	head := frames[1].Apply(bones.Bone(1).Head())
	tail := frames[1].Apply(bones.Bone(1).Tail())
	if math.Abs(tail.Sub(head).Len()-1) > within {
		test.Fatalf("length %v", tail.Sub(head).Len())
	}
}

func TestCarryBlendsByWeight(test *testing.T) {
	bones := testArm(test)
	turn := NewTurn(maths.V(0, 0, 1), math.Pi/2)
	pose := Pose{}.With("lower", turn)
	frames := bones.Frames(pose)
	p := maths.V(2, 0, 0)
	half := Carry(p, frames, []Influence{NewInfluence(0, 1),
		NewInfluence(1, 1)})
	if !half.Near(maths.V(1.5, .5, 0), within) {
		test.Fatalf("half and half went to %v", half)
	}
	if !Carry(p, frames, nil).Near(p, within) {
		test.Fatal("a point no bone moves was moved")
	}
}

func TestAColumnNamesAndCountsItsVertebrae(test *testing.T) {
	bones := New()
	path := func(share float64) (maths.Vec, float64) {
		return maths.V(-share*4, 0, 0), .05
	}
	err := bones.Column("", []Region{NewRegion("C", 7, .2, .2),
		NewRegion("T", 10, 1, 1)}, path)
	if err != nil {
		test.Fatal(err)
	}
	if bones.Count("C") != 7 || bones.Count("T") != 10 {
		test.Fatalf("counts %d and %d", bones.Count("C"), bones.Count("T"))
	}
	t1, _ := bones.Find("T1")
	c7, _ := bones.Find("C7")
	if bones.Bone(t1).Parent() != c7 ||
		!bones.Bone(t1).Head().Near(bones.Bone(c7).Tail(), within) {
		test.Fatal("T1 does not follow C7")
	}
	if last, _ := bones.Find("T10"); math.Abs(
		bones.Bone(last).Tail().X+4) > within {
		test.Fatal("the column does not end at the path's end")
	}
}

func TestAHandHasItsDigitsBones(test *testing.T) {
	bones := New()
	hand := NewHand(maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0),
		.05).Arm(.2, .2, .05).Digit(.1, .2, .1, .1).Digit(-.1, -.2, .1)
	if err := bones.Hand("right-", "", hand); err != nil {
		test.Fatal(err)
	}
	for _, name := range []string{"right-humerus", "right-radius",
		"right-ulna", "right-carpal1", "right-metacarpal1",
		"right-phalanx1-1", "right-carpal2", "right-metacarpal2"} {
		if _, ok := bones.Find(name); !ok {
			test.Fatalf("no %s", name)
		}
	}
	if bones.Count("right-phalanx2") != 0 {
		test.Fatal("the second digit has a phalanx it was not given")
	}
}

func TestEachBoneIsDrawnClosed(test *testing.T) {
	bones := testArm(test)
	points, faces := bones.Mesh(bones.Frames(Pose{}))
	made, err := mesh.New(points, faces, nil, false)
	if err != nil {
		test.Fatal(err)
	}
	if !made.Closed() {
		test.Fatal("the bones are not closed")
	}
}

// Anti-tests: what a skeleton refuses.

func TestABoneOfNoLengthIsRefused(test *testing.T) {
	err := New().Add("a", "", maths.V(1, 1, 1), maths.V(1, 1, 1), .1, .1)
	if !errors.Is(err, errShortBone) {
		test.Fatalf("got %v", err)
	}
}

func TestABoneOfNoThicknessIsRefused(test *testing.T) {
	err := New().Add("a", "", maths.V(0, 0, 0), maths.V(1, 0, 0), 0, .1)
	if !errors.Is(err, errThickness) {
		test.Fatalf("got %v", err)
	}
}

func TestAMissingParentIsRefused(test *testing.T) {
	err := New().Add("a", "b", maths.V(0, 0, 0), maths.V(1, 0, 0), .1, .1)
	if !errors.Is(err, errNoParent) {
		test.Fatalf("got %v", err)
	}
}

func TestANameUsedTwiceIsRefused(test *testing.T) {
	bones := testArm(test)
	err := bones.Add("lower", "", maths.V(0, 0, 0), maths.V(1, 0, 0), .1, .1)
	if !errors.Is(err, errSameName) {
		test.Fatalf("got %v", err)
	}
}

func TestAnEmptyRegionIsRefused(test *testing.T) {
	path := func(float64) (maths.Vec, float64) { return maths.Vec{}, .1 }
	err := New().Column("", []Region{NewRegion("L", 0, 1, 1)}, path)
	if !errors.Is(err, errRegion) {
		test.Fatalf("got %v", err)
	}
	if err := New().Column("", nil, path); !errors.Is(err, errNoVertebrae) {
		test.Fatalf("got %v", err)
	}
}

func TestADigitWithNoBonesIsRefused(test *testing.T) {
	hand := NewHand(maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0),
		.05).Arm(.2, .2, .05).Digit(0, 0)
	if err := New().Hand("", "", hand); !errors.Is(err, errNoDigit) {
		test.Fatalf("got %v", err)
	}
}

package skeleton

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// jointedArm is a root bone along minus x and a hinged child beyond it,
// with joint ranges on both.
func jointedArm(t *testing.T) (*Skeleton, Joints) {
	t.Helper()
	made := New()
	if err := made.Add("upper", "", maths.Vec{}, maths.V(-1, 0, 0), .05,
		.04); err != nil {
		t.Fatal(err)
	}
	if err := made.Add("lower", "upper", maths.V(-1, 0, 0),
		maths.V(-2, 0, 0), .04, .03); err != nil {
		t.Fatal(err)
	}
	joints := Joints{}.With("upper", Hinge(-math.Pi/2, math.Pi/2)).
		With("lower", Hinge(0, 1))
	return made, joints
}

func TestAFlexLiftsABoneThatPointsBack(t *testing.T) {
	made, joints := jointedArm(t)
	pose, err := joints.Turned(made, Pose{}, "upper", math.Pi/2, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	at, _ := made.Find("lower")
	moved := made.Frames(pose)[at].Apply(maths.V(-2, 0, 0))
	if math.Abs(moved.Y-2) > 1e-9 || math.Abs(moved.X) > 1e-9 {
		t.Fatalf("a quarter flex left the tail at %v", moved)
	}
}

func TestATurnReplacesTheBonesEarlierTurn(t *testing.T) {
	made, joints := jointedArm(t)
	pose, err := joints.Turned(made, Pose{}, "lower", 1, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	pose, err = joints.Turned(made, pose, "lower", 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	at, _ := made.Find("lower")
	if made.Frames(pose)[at].Apply(maths.V(-2, 0, 0)).
		Sub(maths.V(-2, 0, 0)).Len() > 1e-9 {
		t.Fatal("a turn of nothing kept the earlier turn")
	}
}

func TestATurnPastItsJointIsRefused(t *testing.T) {
	made, joints := jointedArm(t)
	for _, angle := range [...]float64{-.1, 1.1, math.NaN()} {
		_, err := joints.Turned(made, Pose{}, "lower", angle, 0, 0)
		if !errors.Is(err, errPastJoint) {
			t.Fatalf("flex %v gave %v", angle, err)
		}
	}
	if _, err := joints.Turned(made, Pose{}, "lower", 0, .1,
		0); !errors.Is(err, errPastJoint) {
		t.Fatalf("a hinge bent sideways: %v", err)
	}
}

func TestATurnOfABoneWithNoRangeOrNotHereIsRefused(t *testing.T) {
	made, _ := jointedArm(t)
	if _, err := (Joints{}).Turned(made, Pose{}, "upper", 0, 0,
		0); !errors.Is(err, errUnjointed) {
		t.Fatalf("a bone with no range gave %v", err)
	}
	if _, err := (Joints{}).Turned(made, Pose{}, "wing", 0, 0,
		0); !errors.Is(err, errNoBone) {
		t.Fatalf("a bone not here gave %v", err)
	}
}

func TestLimitIsTheJointsRange(t *testing.T) {
	_, joints := jointedArm(t)
	low, high, found := joints.Limit("lower", Flex)
	if !found || low != 0 || high != 1 {
		t.Fatalf("the elbow's range is %v to %v, found %v", low, high, found)
	}
	if _, _, found := joints.Limit("lower", jointAxes); found {
		t.Fatal("a fourth axis was found")
	}
}

func TestHoldersFollowTheirBonesAndRefuseStrangers(t *testing.T) {
	made, joints := jointedArm(t)
	holders, err := made.Holders([]string{"upper", "lower"}, nil, .1)
	if err != nil {
		t.Fatal(err)
	}
	point := maths.V(-1.9, .05, 0)
	pose, err := joints.Turned(made, Pose{}, "lower", 1, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	moved := Carry(point, made.Frames(pose), Nearness(point, holders))
	if moved.Sub(point).Len() < .3 {
		t.Fatalf("the elbow's turn left the skin at %v", moved)
	}
	if _, err := made.Holders([]string{"wing"}, nil,
		.1); !errors.Is(err, errNoBone) {
		t.Fatalf("a bone not here gave %v", err)
	}
	if _, err := made.Holders([]string{"upper"}, nil,
		0); !errors.Is(err, errReach) {
		t.Fatalf("no reach gave %v", err)
	}
}

func TestARayRunsItsLengthsEndToEnd(t *testing.T) {
	made, _ := jointedArm(t)
	names, err := made.Ray("digit", "lower", maths.V(-2, 0, 0),
		maths.V(-1, 0, 0), []float64{.05, .03, .02}, .005)
	if err != nil {
		t.Fatal(err)
	}
	at, found := made.Find(names[2])
	if !found || math.Abs(made.Bone(at).Tail().X+2.1) > 1e-9 {
		t.Fatalf("the ray ends at %v", made.Bone(at).Tail())
	}
	if _, err := made.Ray("toe", "lower", maths.Vec{}, maths.V(1, 0, 0),
		nil, .005); !errors.Is(err, errRay) {
		t.Fatalf("no lengths gave %v", err)
	}
}

func TestALoftIsClosedAndLiftedBetweenItsSections(t *testing.T) {
	loft := NewLoft(maths.Vec{}, maths.V(1, 0, 0), maths.Up,
		NewLoftSection(.1, .1, .05, 0), NewLoftSection(.9, .02, .02, .1))
	if middle := loft.at(.5); math.Abs(middle.lift-.05) > 1e-9 ||
		math.Abs(middle.wide-.06) > 1e-9 {
		t.Fatalf("half way the loft is %+v", middle)
	}
	points, faces := loft.Build(nil, nil,
		func(point maths.Vec) maths.Vec { return point })
	built, err := mesh.New(points, faces, nil, false)
	if err != nil || !built.Closed() {
		t.Fatalf("the loft is not a closed piece: %v", err)
	}
	if points, _ := NewLoft(maths.Vec{}, maths.V(1, 0, 0),
		maths.Up).Build(nil, nil, func(p maths.Vec) maths.Vec {
		return p
	}); len(points) != 0 {
		t.Fatal("a loft of no sections added points")
	}
}

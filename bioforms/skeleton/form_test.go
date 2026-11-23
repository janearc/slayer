package skeleton

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// testForms is a form for each bone of testArm: a rod, and a vertebra.
func testForms() map[string]BoneForm {
	return map[string]BoneForm{
		"upper": NewRod(maths.V(0, 0, 0), maths.V(1, 0, 0), .1, .6),
		"lower": NewCentrum(maths.V(1, 0, 0), maths.V(2, 0, 0), .1, .1, .1),
	}
}

func TestDrawnFormsAreClosedAndOutward(test *testing.T) {
	bones := testArm(test)
	points, faces := bones.Draw(bones.Frames(Pose{}), testForms())
	drawn, err := shape.NewMesh(points, faces, nil, false)
	if err != nil {
		test.Fatal(err)
	}
	if !drawn.Closed() {
		test.Fatal("the drawn bones are open")
	}
}

func TestABoneWithNoFormIsDrawnAsASpindle(test *testing.T) {
	bones := testArm(test)
	frames := bones.Frames(Pose{})
	plain, _ := bones.Mesh(frames)
	drawn, _ := bones.Draw(frames, nil)
	if len(plain) != len(drawn) {
		test.Fatalf("%d points drawn, %d as spindles", len(drawn), len(plain))
	}
}

func TestAnArcPassesThroughItsEnds(test *testing.T) {
	arc := NewArc([4]maths.Vec{maths.V(0, 0, 0), maths.V(1, 1, 0),
		maths.V(2, 1, 0), maths.V(3, 0, 0)}, .01, .02, maths.V(0, 0, 1))
	if !arc.At(0).Near(maths.V(0, 0, 0), within) || !arc.At(1).Near(maths.V(3, 0, 0), within) {
		test.Fatal("the arc does not run end to end")
	}
}

func TestTwoQuarterTurnsComposeToAHalf(test *testing.T) {
	bones := testArm(test)
	quarter := NewTurn(maths.V(0, 0, 1), math.Pi/2)
	pose := Pose{}.Compose("lower", quarter).Compose("lower", quarter)
	frames := bones.Frames(pose)
	at, _ := bones.Find("lower")
	if got := frames[at].Apply(maths.V(2, 0, 0)); !got.Near(maths.V(0, 0, 0), 1e-9) {
		test.Fatalf("two quarter turns took the tip to %v", got)
	}
}

func TestComposeOnAFreeJointIsWith(test *testing.T) {
	turn := NewTurn(maths.V(0, 1, 0), .4)
	bones := testArm(test)
	a := bones.Frames(Pose{}.Compose("upper", turn))
	b := bones.Frames(Pose{}.With("upper", turn))
	p := maths.V(1.5, .2, .1)
	if !a[1].Apply(p).Near(b[1].Apply(p), within) {
		test.Fatal("composing onto no turn differs from With")
	}
}

func TestAimPointsTheBone(test *testing.T) {
	bones := testArm(test)
	turn := NewTurn(maths.V(0, 0, 1), .5)
	pose := Pose{}.With("upper", turn)
	way := maths.V(0, -1, 1)
	pose = bones.Aim(pose, "lower", way)
	frames := bones.Frames(pose)
	at, _ := bones.Find("lower")
	bone := bones.Bone(at)
	tail := frames[at].Apply(bone.Tail())
	head := frames[at].Apply(bone.Head())
	got := tail.Sub(head).Unit()
	if !got.Near(way.Unit(), 1e-9) {
		test.Fatalf("the bone points %v, not %v", got, way.Unit())
	}
}

func TestAimAtNoBoneLeavesThePose(test *testing.T) {
	bones := testArm(test)
	turn := NewTurn(maths.V(0, 0, 1), .5)
	pose := Pose{}.With("upper", turn)
	aimed := bones.Aim(pose, "tail", maths.V(0, 1, 0))
	p := maths.V(2, 0, 0)
	if !bones.Frames(aimed)[1].Apply(p).Near(bones.Frames(pose)[1].Apply(p), within) {
		test.Fatal("aiming a bone there is not changed the pose")
	}
}

func TestNearnessFavoursTheNearerBone(test *testing.T) {
	holders := []Holder{
		NewHolder(0, []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0)}, 1),
		NewHolder(1, []maths.Vec{maths.V(1, 0, 0), maths.V(2, 0, 0)}, 1),
	}
	near := Nearness(maths.V(.2, .3, 0), holders)
	if near[0].Weight() <= near[1].Weight() {
		test.Fatalf("weights %g and %g", near[0].Weight(), near[1].Weight())
	}
	on := Nearness(maths.V(.5, 0, 0), holders)
	if math.IsInf(on[0].Weight(), 0) || math.IsNaN(on[0].Weight()) {
		test.Fatal("a point on a bone has no finite weight")
	}
}

func TestAShortReachHoldsLess(test *testing.T) {
	line := []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0)}
	p := maths.V(.5, .3, 0)
	wide := Nearness(p, []Holder{NewHolder(0, line, 1)})[0].Weight()
	short := Nearness(p, []Holder{NewHolder(0, line, .4)})[0].Weight()
	if short >= wide {
		test.Fatalf("a short reach holds %g, a full one %g", short, wide)
	}
}

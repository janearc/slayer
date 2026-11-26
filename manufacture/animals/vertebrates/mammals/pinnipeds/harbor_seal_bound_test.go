package pinnipeds

import (
	"math"
	"testing"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

func sealRigOnBonesForTest(t *testing.T) *SealRig {
	t.Helper()
	rig, err := NewSealRigOnBones()
	if err != nil {
		t.Fatal(err)
	}
	return rig
}

func TestEveryMassSitsOnABoneOnEachOfItsSides(t *testing.T) {
	body, err := newSealBody(sealBonesForTest(t))
	if err != nil {
		t.Fatal(err)
	}
	want := 0
	for _, mass := range sealMasses {
		want++
		if mass.mirrored {
			want++
		}
	}
	if len(body.centres) != want || len(body.halves) != want {
		t.Fatalf("%d masses placed, want %d", len(body.centres), want)
	}
}

func TestAMassOnABoneThatIsNotHereIsRefused(t *testing.T) {
	bones := &sealBones{bones: skeleton.New()}
	if _, err := newSealBody(bones); err == nil {
		t.Fatal("masses were placed on a skeleton of no bones")
	}
}

func TestARayLeavesAUnitBallAtOne(t *testing.T) {
	out, through := sealExit(maths.Vec{}, maths.V(1, 0, 0), maths.Vec{},
		maths.V(1, 1, 1))
	if !through || math.Abs(out-1) > 1e-12 {
		t.Fatalf("left at %v, through %v", out, through)
	}
}

func TestARayPastABallMissesIt(t *testing.T) {
	_, through := sealExit(maths.Vec{}, maths.V(1, 0, 0), maths.V(0, 3, 0),
		maths.V(1, 1, 1))
	if through {
		t.Fatal("a ray passing three radii away went through")
	}
	_, through = sealExit(maths.V(5, 0, 0), maths.V(1, 0, 0), maths.Vec{},
		maths.V(1, 1, 1))
	if through {
		t.Fatal("a ray leaving away from a ball behind it went through")
	}
}

func TestBlubberIsThickOnTheTrunkAndGoneAtTheNose(t *testing.T) {
	if math.Abs(sealBlubber(0)-sealTrunkBlubber) > 1e-9 {
		t.Fatalf("trunk blubber %v", sealBlubber(0))
	}
	if sealBlubber(.7) >= sealTrunkBlubber || sealBlubber(sealNoseTip) > 1e-9 {
		t.Fatal("the head's blubber is not thinner, or the nose not bare")
	}
}

func TestARigOnBonesRestsWhereItWasBuilt(t *testing.T) {
	rig := sealRigOnBonesForTest(t)
	points := rig.posed(SealRigPose{})
	for index, point := range points {
		if point.Sub(rig.rest[index]).Len() > 1e-9 {
			t.Fatalf("vertex %d moved at rest by %v", index,
				point.Sub(rig.rest[index]).Len())
		}
	}
}

func TestEveryRigPoseStaysWithinTheBonesJoints(t *testing.T) {
	rig := sealRigOnBonesForTest(t)
	for _, sign := range [...]float64{-1, 1} {
		pose := SealRigPose{Yaw: sign * math.Pi / 2, Pitch: sign * .65,
			HeadPitch: sign * .92, Fore: [2]float64{sign * 1.15, -sign * 1.15},
			Hind: [2]float64{sign * .8, -sign * .8}}
		if _, err := rig.bound.turns(pose, rig.flexStart); err != nil {
			t.Fatalf("the rig's bound pose %v broke a joint: %v", pose, err)
		}
	}
}

func TestTheMuzzleRisesWithTheNod(t *testing.T) {
	rig := sealRigOnBonesForTest(t)
	muzzle := maths.V(.80, .05, 0)
	up := rig.HeadPoint(SealRigPose{HeadPitch: .5}, muzzle)
	down := rig.HeadPoint(SealRigPose{HeadPitch: -.5}, muzzle)
	if !(up.Y > muzzle.Y && down.Y < muzzle.Y) {
		t.Fatalf("a nod up left the muzzle at %v and down at %v", up, down)
	}
}

func TestAHindFlipperFeathersAndTheForeStaysPut(t *testing.T) {
	rig := sealRigOnBonesForTest(t)
	still := rig.posed(SealRigPose{})
	feathered := rig.posed(SealRigPose{Hind: [2]float64{.8, 0}})
	tip := rig.limbs[2].pole
	if feathered[tip].Sub(still[tip]).Len() < .01 {
		t.Fatal("feathering the hind flipper left its tip in place")
	}
	nose := len(rig.rest) - 1
	for index, vertex := range rig.vertices {
		if vertex.point.X > rig.vertices[nose].point.X {
			nose = index
		}
	}
	if feathered[nose].Sub(still[nose]).Len() > 1e-9 {
		t.Fatal("feathering a hind flipper moved the nose")
	}
}

func TestABadRigPoseOnBonesGivesNoNumbers(t *testing.T) {
	rig := sealRigOnBonesForTest(t)
	point := rig.HeadPoint(SealRigPose{HeadPitch: 5}, maths.V(.8, .05, 0))
	if !math.IsNaN(point.X) {
		t.Fatalf("a nod past the neck's joints gave %v", point)
	}
	if _, err := rig.Skin(SealRigPose{HeadPitch: 5}); err == nil {
		t.Fatal("a nod past the rig's bound made a skin")
	}
}

func TestTheWhiskerRowsHoldTheArraysCountInItsGrid(t *testing.T) {
	total := 0
	for row, count := range sealWhiskersInRow {
		if count < 1 || count > sealWhiskerColumns {
			t.Fatalf("row %d holds %d strands in %d columns", row, count,
				sealWhiskerColumns)
		}
		total += count
	}
	if total != sealMystacials {
		t.Fatalf("the rows hold %d strands, want %d", total, sealMystacials)
	}
}

func TestAMarkIsASpotARingOrABlotchByItsShare(t *testing.T) {
	cases := map[float64]int{0: sealMarkRing,
		sealRingShare + sealBlotchShare/2: sealMarkBlotch, .9: sealMarkSpot}
	for share, want := range cases {
		if got := sealMarkKind(share); got != want {
			t.Fatalf("share %v gave kind %d, want %d", share, got, want)
		}
	}
}

func TestTheCoatHasMarksOnTheBackAndBlendsThemFromFar(t *testing.T) {
	const seed = 407
	marked := 0
	for step := range 400 {
		point := maths.V(-.6+.003*float64(step), .2, .05)
		if mark, _ := sealMarks(seed, point, 0); mark > .5 {
			marked++
		}
	}
	if marked == 0 || marked == 400 {
		t.Fatalf("%d of 400 points along the back are marked", marked)
	}
	mark, lift := sealMarks(seed, maths.V(0, .2, 0), 1)
	if math.Abs(mark-sealBackMarked*sealMeanCover) > 1e-9 || lift != 0 {
		t.Fatalf("from far the back is %v dark and %v lifted", mark, lift)
	}
}

func TestTheCoatsMarksRepeatForASeed(t *testing.T) {
	point := maths.V(.1, .15, .07)
	first, _ := sealMarks(9, point, 0)
	again, _ := sealMarks(9, point, 0)
	if first != again {
		t.Fatal("the same seed and point gave two marks")
	}
}

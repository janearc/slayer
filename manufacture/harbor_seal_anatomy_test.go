package manufacture

import (
	"math"
	"slices"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

func TestSealDigitStationsRunAlongTheOutline(t *testing.T) {
	for _, test := range []struct {
		loop int
		want []int
	}{
		{16, []int{0, 0, 0, 0, 0, 1, 2, 3, 4, 4, 4, 4, 4, 3, 2, 1}},
		{18, []int{0, 0, 0, 0, 0, 0, 1, 2, 3, 4, 4, 4, 4, 4, 4, 3, 2, 1}},
	} {
		if got := sealDigitStations(test.loop); !slices.Equal(got, test.want) {
			t.Fatalf("loop %d: stations %v, want %v", test.loop, got, test.want)
		}
	}
}

func TestSealDigitLabelsHaveWebBetweenOutlineVertices(t *testing.T) {
	labels := foreHand.labels(16)
	if len(labels) != 32 {
		t.Fatalf("%d labels for a 16 vertex outline", len(labels))
	}
	seen := map[int]bool{}
	for place, label := range labels {
		if place%2 == 1 && label != -1 {
			t.Fatalf("web vertex %d labelled digit %d", place, label)
		}
		seen[label] = true
	}
	for digit := range sealDigits {
		if !seen[digit] {
			t.Fatalf("no vertex ends digit %d", digit)
		}
	}
}

func TestSealHandWebFallsShortOfItsDigits(t *testing.T) {
	for _, hand := range []sealHand{foreHand, hindHand} {
		for digit := range sealDigits - 1 {
			web := hand.reachAt(float64(digit) + .5)
			if web >= min(hand.reach[digit], hand.reach[digit+1]) {
				t.Fatalf("web %.4f m is not shorter than digits %d and %d", web, digit, digit+1)
			}
			if hand.thickness(float64(digit)+.5) >= hand.thickness(float64(digit)) {
				t.Fatalf("web is not thinner than the digit at %d", digit)
			}
		}
	}
}

func TestSealRigFlippersEndInFiveSeparateDigits(t *testing.T) {
	rig := NewHarborSealRig()
	points := rig.posed(SealRigPose{})
	for index, limb := range rig.limbs {
		var tips [sealDigits]maths.Vec
		for digit := range tips {
			tips[digit] = sealDigitMean(points, limb.outer, limb.digits, digit)
		}
		for digit := range sealDigits - 1 {
			if apart := tips[digit].Sub(tips[digit+1]).Len(); apart < .012 {
				t.Fatalf("limb %d: digits %d and %d are %.4f m apart", index, digit, digit+1, apart)
			}
		}
	}
}

func TestSealRigHindFlippersTrailAtEveryLegalPose(t *testing.T) {
	rig := NewHarborSealRig()
	for step := -8; step <= 8; step++ {
		share := float64(step) / 8
		pose := SealRigPose{Yaw: share, Hind: [2]float64{.8 * share, -.8 * share}}
		chains, err := rig.Skeleton(pose)
		if err != nil {
			t.Fatal(err)
		}
		for _, chain := range chains[3:] {
			if chain[2].X >= chain[1].X {
				t.Fatalf("step %d: a hindflipper tip lies forward of its root", step)
			}
		}
	}
	if _, err := rig.Skeleton(SealRigPose{Hind: [2]float64{.9, 0}}); err == nil {
		t.Fatal("accepted a hind flipper turned beyond the legal range")
	}
}

func TestSealRigTailEndsBehindTheTrunkAndKeepsOneClosedSkin(t *testing.T) {
	rig := NewHarborSealRig()
	skin, err := rig.Skin(SealRigPose{})
	if err != nil {
		t.Fatal(err)
	}
	if !skin.Closed() || !sealSkinConnected(skin) {
		t.Fatal("the skin with its tail and digits is not one closed mesh")
	}
	last := sealTailRings[len(sealTailRings)-1].back
	if skin.Bounds().Min.X > sealHip+last+.01 {
		t.Fatalf("tail does not reach %.3f m behind the hip: %.3f", last, skin.Bounds().Min.X)
	}
	if last < -.12 || last > -.05 {
		t.Fatalf("tail %.3f m is not short", last)
	}
}

func TestSealRigRestHasNoFoldsAndTheNeckRollsOnTheInside(t *testing.T) {
	rig := NewHarborSealRig()
	rest := rig.posed(SealRigPose{})
	for index, point := range rest {
		if point.Sub(rig.rest[index]).Len() > 1e-9 {
			t.Fatalf("vertex %d is off its rest place with no pose", index)
		}
	}
	belly, back := maths.V(0, -1, 0), maths.V(0, 1, 0)
	crest := sealNeckStart + sealNeckPeriod/2
	at := maths.V(crest, 0, 0)
	if sealNeckFold(at, belly, -.8) <= 0 {
		t.Fatal("no roll under the chin when the head nods down")
	}
	if sealNeckFold(at, back, -.8) != 0 {
		t.Fatal("the nape rolled when the head nodded down")
	}
	if sealNeckFold(at, back, .8) <= 0 {
		t.Fatal("no roll at the nape when the head tips back")
	}
	if sealNeckFold(at, belly, 0) != 0 {
		t.Fatal("a roll with the head level")
	}
	if sealNeckFold(maths.V(sealNeckEnd+.05, 0, 0), belly, -.8) != 0 {
		t.Fatal("the neck rolled past the skull")
	}
}

func TestSealRootFoldNeedsATurnAndStaysNearTheRoot(t *testing.T) {
	ring := sealRootPeriod / 2
	for ring < sealRootNear+sealRootBlend {
		ring += sealRootPeriod
	}
	if sealRootFold(ring, 0) != 0 {
		t.Fatal("a ring round the root with the flipper at rest")
	}
	if sealRootFold(ring, sealRootTurn) <= 0 {
		t.Fatal("no ring at a full turn")
	}
	if sealRootFold(sealRootFar+.1, sealRootTurn) != 0 {
		t.Fatal("a ring far from the root")
	}
}

func TestSealRigFoldsStayWithinTheirDepth(t *testing.T) {
	rig := NewHarborSealRig()
	pose := SealRigPose{HeadPitch: -.9, Fore: [2]float64{-1, 1}, Hind: [2]float64{.7, -.7}}
	most := math.Max(sealNeckDepth, sealRootDepth)
	for index := range rig.vertices {
		if fold := rig.fold(index, pose).Len(); fold > sealNeckDepth+sealRootDepth+most {
			t.Fatalf("vertex %d is folded %.4f m", index, fold)
		}
	}
}

func TestSealRestHitFindsTheSkinFromOutside(t *testing.T) {
	rig := NewHarborSealRig()
	point, normal, found := rig.restHit(maths.V(0, .5, 0), maths.V(0, -1, 0))
	if !found || point.Y < .1 || point.Y > .3 || normal.Y < .9 {
		t.Fatalf("back at the middle: %v %v %v", point, normal, found)
	}
	if _, _, found := rig.restHit(maths.V(0, 3, 0), maths.V(0, 1, 0)); found {
		t.Fatal("a ray leaving the animal hit it")
	}
}

func TestSealPartsBuildEveryLandmarkOnTheSkin(t *testing.T) {
	rig := NewHarborSealRig()
	var molds sealMolds
	pose := harborSealRelaxed
	look := SealLook{Wet: .5, Nostrils: .5}
	for _, side := range [...]float64{-1, 1} {
		for _, build := range []func(SealRigPose, float64, SealLook, *sealMolds) error{
			rig.eye, rig.ear, rig.nostril, rig.whiskers, rig.brow,
		} {
			if err := build(pose, side, look, &molds); err != nil {
				t.Fatal(err)
			}
		}
	}
	strands := 2 * (sealMystacials + sealBrowCount)
	const sides = 3
	perStrand := 2*sealWhiskerSteps*sides + 2*sides
	if got := len(molds.hair.faces); got != strands*perStrand {
		t.Fatalf("%d whisker faces, want %d for %d separate strands", got, strands*perStrand, strands)
	}
	if len(molds.cornea.faces) == 0 || len(molds.rim.faces) == 0 || len(molds.dark.faces) == 0 {
		t.Fatal("eyes, lids, ears or nostrils are missing")
	}
	if err := rig.umbilicus(pose, &molds); err != nil {
		t.Fatal(err)
	}
}

func TestSealPartsHaveNoPinnaAndTheEarIsAnOpeningAtSkinLevel(t *testing.T) {
	rig := NewHarborSealRig()
	var molds sealMolds
	if err := rig.ear(SealRigPose{}, 1, SealLook{}, &molds); err != nil {
		t.Fatal(err)
	}
	box := sealMoldBox(&molds.rim)
	if box.Max.Sub(box.Min).Len() > 4*sealEarLong {
		t.Fatalf("an ear structure %.3f m across stands on the head", box.Max.Sub(box.Min).Len())
	}
}

func sealMoldBox(mold *sealMold) maths.Box {
	box := maths.Box{Min: maths.V(math.Inf(1), math.Inf(1), math.Inf(1)),
		Max: maths.V(math.Inf(-1), math.Inf(-1), math.Inf(-1))}
	for _, one := range mold.faces {
		for _, point := range one {
			box.Min = maths.V(math.Min(box.Min.X, point.X), math.Min(box.Min.Y, point.Y),
				math.Min(box.Min.Z, point.Z))
			box.Max = maths.V(math.Max(box.Max.X, point.X), math.Max(box.Max.Y, point.Y),
				math.Max(box.Max.Z, point.Z))
		}
	}
	return box
}

func TestSealNostrilsCloseToASeam(t *testing.T) {
	rig := NewHarborSealRig()
	widths := make([]float64, 0, 2)
	for _, shut := range []float64{0, 1} {
		var molds sealMolds
		if err := rig.nostril(SealRigPose{}, 1, SealLook{Nostrils: shut}, &molds); err != nil {
			t.Fatal(err)
		}
		box := sealMoldBox(&molds.dark)
		widths = append(widths, box.Max.Sub(box.Min).Len())
	}
	if widths[1] >= widths[0] {
		t.Fatalf("a shut nostril %.5f m is not narrower than an open one %.5f m", widths[1], widths[0])
	}
}

func TestSealNostrilsMirrorAsAV(t *testing.T) {
	rig := NewHarborSealRig()
	var left, right sealMolds
	if err := rig.nostril(SealRigPose{}, -1, SealLook{}, &left); err != nil {
		t.Fatal(err)
	}
	if err := rig.nostril(SealRigPose{}, 1, SealLook{}, &right); err != nil {
		t.Fatal(err)
	}
	a, b := sealMoldBox(&left.dark), sealMoldBox(&right.dark)
	if a.Max.Z >= 0 || b.Min.Z <= 0 {
		t.Fatalf("nostrils are not on opposite sides: %v %v", a, b)
	}
	if math.Abs((a.Max.Y-a.Min.Y)-(b.Max.Y-b.Min.Y)) > 1e-6 {
		t.Fatal("the two slits are not mirror images")
	}
}

// sealHeadFrom is where the head begins along the rest body, in metres:
// from the neck forward, where the landmarks are set by the skin's normals.
const sealHeadFrom = .6

// sealMirrorTolerance is how far a normal and its mirror image may part:
// the rounding of a sum of areas.
const sealMirrorTolerance = 1e-9

func TestSealHeadNormalsMirrorAcrossTheMidline(t *testing.T) {
	rig := NewHarborSealRig()
	checked := 0
	for index, point := range rig.rest {
		if point.X < sealHeadFrom {
			continue
		}
		mirror := -1
		for other, candidate := range rig.rest {
			if math.Abs(candidate.X-point.X)+math.Abs(candidate.Y-point.Y)+
				math.Abs(candidate.Z+point.Z) < sealMirrorTolerance {
				mirror = other
				break
			}
		}
		if mirror < 0 {
			t.Fatalf("the head's point %v has no mirror image", point)
		}
		mine, theirs := rig.normals[index], rig.normals[mirror]
		if math.Abs(mine.X-theirs.X)+math.Abs(mine.Y-theirs.Y)+
			math.Abs(mine.Z+theirs.Z) > sealMirrorTolerance {
			t.Fatalf("the normal at %v is %v, its mirror image's is %v",
				point, mine, theirs)
		}
		checked++
	}
	if checked < sealSections {
		t.Fatalf("only %d points of the head were checked", checked)
	}
}

func TestSealClawsEndEveryDigit(t *testing.T) {
	rig := NewHarborSealRig()
	var molds sealMolds
	rig.claws(rig.posed(SealRigPose{}), &molds)
	box := sealMoldBox(&molds.horn)
	if len(molds.horn.faces) < 4*sealDigits*sealClawSteps {
		t.Fatalf("%d claw faces for twenty digits", len(molds.horn.faces))
	}
	if box.Max.Sub(box.Min).Len() < sealForeClawLong {
		t.Fatal("claws are smaller than one claw")
	}
}

func TestSealFurIsPlentifulAndWetHairLiesFlatter(t *testing.T) {
	rig := NewHarborSealRig()
	pose := harborSealRelaxed
	dry, err := rig.Fur(pose, SealLook{Seed: 7})
	if err != nil {
		t.Fatal(err)
	}
	wet, err := rig.Fur(pose, SealLook{Wet: 1, Seed: 7})
	if err != nil {
		t.Fatal(err)
	}
	if dry.FaceCount() < 50000 {
		t.Fatalf("%d hair faces is not a coat", dry.FaceCount())
	}
	if wet.FaceCount() >= dry.FaceCount() {
		t.Fatalf("a soaked coat has %d faces and a dry one %d: wet hair clumps into fewer blades",
			wet.FaceCount(), dry.FaceCount())
	}
	if sealFurFor(1).lift >= sealFurFor(0).lift || sealFurFor(1).length >= sealFurFor(0).length {
		t.Fatal("wet hair does not lie flatter and shorter")
	}
}

func TestSealFurIsRepeatableAndTakesItsSeed(t *testing.T) {
	rig := NewHarborSealRig()
	one, err := rig.Fur(SealRigPose{}, SealLook{Seed: 3})
	if err != nil {
		t.Fatal(err)
	}
	again, _ := rig.Fur(SealRigPose{}, SealLook{Seed: 3})
	other, _ := rig.Fur(SealRigPose{}, SealLook{Seed: 4})
	if !slices.Equal(one.Points(), again.Points()) {
		t.Fatal("the same seed grew different hair")
	}
	if slices.Equal(one.Points(), other.Points()) {
		t.Fatal("another seed grew the same hair")
	}
}

func TestSealFurIsBareOnTheNoseAndAroundTheEye(t *testing.T) {
	rig := NewHarborSealRig()
	discs, err := rig.bareDiscs()
	if err != nil {
		t.Fatal(err)
	}
	if !bareAt(discs, maths.V(.82, .05, 0)) {
		t.Fatal("hair on the nose pad")
	}
	eyelid := discs[0].centre.Add(maths.V(.01, 0, 0))
	if !bareAt(discs, eyelid) {
		t.Fatal("hair on the eyelid")
	}
	if bareAt(discs, maths.V(0, 0, .1)) {
		t.Fatal("no hair on the flank")
	}
}

func TestSealFurRefusesAPoseOutsideItsBounds(t *testing.T) {
	rig := NewHarborSealRig()
	if _, err := rig.Fur(SealRigPose{HeadPitch: 2}, SealLook{}); err == nil {
		t.Fatal("grew hair on an impossible pose")
	}
	if _, err := rig.Fur(SealRigPose{}, SealLook{Wet: 2}); err == nil {
		t.Fatal("grew hair on a coat more than soaked")
	}
	if _, err := rig.Parts(SealRigPose{}, SealLook{Nostrils: math.NaN()}); err == nil {
		t.Fatal("made parts for a nostril that is no number")
	}
}

func TestSealFlowRunsTailwardAndAwayFromTheFlipperRoots(t *testing.T) {
	rig := NewHarborSealRig()
	field := rig.flowField()
	back := field.at(maths.V(-.2, .1, 0))
	if back.X > -.9 {
		t.Fatalf("the coat on the back does not run to the tail: %v", back)
	}
	for index, limb := range rig.limbs {
		beyond := limb.tip.Add(limb.tip.Sub(limb.root).Scale(.1))
		away := field.at(beyond).Dot(beyond.Sub(limb.root).Unit())
		if away < .5 {
			t.Fatalf("limb %d: the coat does not run away from the root (%.2f)", index, away)
		}
	}
	head := field.at(maths.V(.78, .06, .05))
	if head.Z <= 0 {
		t.Fatalf("the head's coat does not fan out from the midline: %v", head)
	}
}

func TestSealPelageDarkensWhenWetAndKeepsANosePad(t *testing.T) {
	rig := NewHarborSealRig()
	dry, wet := rig.Pelage(SealLook{Seed: 5}), rig.Pelage(SealLook{Wet: 1, Seed: 5})
	for _, point := range []maths.Vec{maths.V(0, .2, 0), maths.V(-.4, -.1, .12), maths.V(.3, .05, -.1)} {
		if sealLuminance(wet.At(point)) >= sealLuminance(dry.At(point)) {
			t.Fatalf("soaked coat is not darker at %v", point)
		}
	}
	pad := maths.V(.81, .059, .01)
	if sealNose(pad) < .5 || sealNose(maths.V(0, 0, 0)) != 0 {
		t.Fatal("the nose pad is not where the muzzle ends")
	}
	if sealLuminance(dry.At(pad)) > sealLuminance(sealNoseColour)*2 {
		t.Fatal("the nose pad is not dark")
	}
}

func sealLuminance(colour maths.Light) float64 {
	return colour.R + colour.G + colour.B
}

func TestSealPelageStreaksFollowTheFlowAndFadeWithFootprint(t *testing.T) {
	rig := NewHarborSealRig()
	pelage := rig.Pelage(SealLook{Seed: 5}).(sealPelage)
	// A point between the noise's lattice nodes: at a node, as (-.2, .1,
	// .05) is for a coat that flows tailward, gradient noise is nought.
	point := maths.V(-.2013, .1007, .0511)
	if pelage.streak(point, 0) == 0 {
		t.Fatal("no streaks at a lone ray")
	}
	if pelage.streak(point, sealStreakFar) != 0 {
		t.Fatal("streaks glitter at a far footprint")
	}
}

func TestHarborSealKindsAreRegisteredAndMakeTheWholeAnimal(t *testing.T) {
	for _, name := range []string{"harbor-seal", "harbor-seal-dry"} {
		kind, err := KindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		made, err := MakeWithComplexity(kind, Sow(407), "quick")
		if err != nil {
			t.Fatal(err)
		}
		parts, err := made.Things(maths.Local())
		if err != nil {
			t.Fatal(err)
		}
		if len(parts) < 6 {
			t.Fatalf("%s: %d parts", name, len(parts))
		}
		skin, ok := parts[0].Shape.(*mesh.Mesh)
		if !ok || !skin.Closed() {
			t.Fatalf("%s: the first part is not a closed skin", name)
		}
		if parts[0].Material.Name != "harbor seal skin" || parts[1].Material.Name != "harbor seal hair" {
			t.Fatalf("%s: skin and hair are not the first two parts", name)
		}
	}
	wet, _ := KindNamed("harbor-seal")
	dry, _ := KindNamed("harbor-seal-dry")
	if wet == dry {
		t.Fatal("wet and dry are one kind")
	}
	if _, err := KindNamed("harbor-seal-fur"); err == nil {
		t.Fatal("found a kind nobody made")
	}
}

// Both looks are made at the seeds the catalogue and the set tests draw
// from, not only the one above: a nostril shut as far as a look can shut it
// is a ring the mesh takes, at every seed.
func TestAShutHarborSealIsMadeAtEverySeed(t *testing.T) {
	for _, name := range []string{"harbor-seal", "harbor-seal-dry"} {
		kind, err := KindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, number := range []uint64{0, 1, 2, 407} {
			if _, err := MakeWithComplexity(kind, Sow(number),
				"quick"); err != nil {
				t.Errorf("%s at seed %d: %v", name, number, err)
			}
		}
	}
}

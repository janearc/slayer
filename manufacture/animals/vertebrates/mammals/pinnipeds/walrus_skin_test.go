package pinnipeds

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// walrusStrandFaces is the faces in one bristle: two for each flat side of
// each segment, and a fan at each end.
const walrusStrandFaces = 2*walrusStrandSides*walrusStrandSegments + 2*walrusStrandSides

// testWalrusSkin is the hide's binding and the head's motion in a pose,
// bent and stroked, failing the test on an error.
func testWalrusSkin(t *testing.T, pose walrusPose, yaw, phase float64) (func(maths.Vec) maths.Vec,
	func(maths.Vec) maths.Vec) {
	t.Helper()
	hide, head, err := defaultWalrusRig().skin(pose, yaw, phase)
	if err != nil {
		t.Fatal(err)
	}
	return hide, head
}

func TestWalrusSkinWeightsAddToOne(t *testing.T) {
	// Turn the root, which carries every bone alike: if the weights add to
	// one, every point of the hide goes exactly where the root takes it.
	body := walrusRest
	turn := skeleton.NewTurn(maths.V(0, 1, 0), .7)
	pose := skeleton.Pose{}.With("thoracic 14", turn)
	frames := body.tree.Frames(pose)
	holders := walrusHolders(body)
	for _, p := range []maths.Vec{maths.V(1.6, .1, .14), maths.V(-1.2, -.1, 0), maths.V(.4, -.3, .8)} {
		moved := skeleton.Carry(p, frames, skeleton.Nearness(p, holders))
		if off := moved.Sub(frames[0].Apply(p)).Len(); !(off <= 1e-9) {
			t.Fatalf("%v is %g off where the whole body takes it", p, off)
		}
	}
}

func TestWalrusSkinRestIsIdentity(t *testing.T) {
	skin, head := testWalrusSkin(t, walrusLying, 0, 0)
	for _, p := range []maths.Vec{maths.V(1.6, .1, .14), maths.V(-1.2, -.1, 0), maths.V(.4, -.3, .8)} {
		if moved := skin(p).Sub(p).Len(); !(moved <= 1e-12) {
			t.Fatalf("rest pose moves %v by %g", p, moved)
		}
		if moved := head(p).Sub(p).Len(); !(moved <= 1e-12) {
			t.Fatalf("rest pose moves the head at %v by %g", p, moved)
		}
	}
}

func TestWalrusSkinHeadIsRigid(t *testing.T) {
	_, bent := testWalrusSkin(t, walrusLying, .8, 0)
	_, stroked := testWalrusSkin(t, walrusLying, 0, math.Pi/2)
	for _, p := range []maths.Vec{maths.V(1.6, .1, .14), maths.V(1.7, .2, -.1)} {
		if !(bent(p).Sub(p).Len() <= 1e-12) || !(stroked(p).Sub(p).Len() <= 1e-12) {
			t.Fatalf("head point %v moved", p)
		}
	}
	_, raised := testWalrusSkin(t, walrusHauledOut, 0, 0)
	a, b := maths.V(1.6, .1, .14), maths.V(1.7, .2, -.1)
	if gap := raised(a).Sub(raised(b)).Len() - a.Sub(b).Len(); math.Abs(gap) > 1e-9 {
		t.Fatalf("a raised head stretches by %g", gap)
	}
}

func TestWalrusSkinTailAndFlippersMove(t *testing.T) {
	tail := maths.V(-1.25, -.02, 0)
	bent, _ := testWalrusSkin(t, walrusLying, .8, 0)
	if bent(tail).Sub(tail).Len() < .2 {
		t.Fatal("a bent spine leaves the tail where it was")
	}
	hind := walrusRayLines[3][2]
	tip := hind[len(hind)-2]
	stroked, _ := testWalrusSkin(t, walrusLying, 0, math.Pi/2)
	if stroked(tip).Sub(tip).Len() < .02 {
		t.Fatal("a stroke leaves the hind flipper where it was")
	}
}

func TestWalrusSkinKeepsLengthsAcrossAJoint(t *testing.T) {
	// Two points a short distance apart on either side of a joint must stay
	// about as far apart when the spine bends: skin is blended, not torn.
	skin, _ := testWalrusSkin(t, walrusLying, .8, 0)
	for _, name := range walrusBendJoints {
		bone, _ := walrusRest.bone(name)
		joint := bone.Head()
		a, b := joint.Add(maths.V(.05, 0, .3)), joint.Add(maths.V(-.05, 0, .3))
		before, after := a.Sub(b).Len(), skin(a).Sub(skin(b)).Len()
		if !(math.Abs(after-before) <= .5*before) {
			t.Fatalf("joint %s: %.3f becomes %.3f", name, before, after)
		}
	}
}

func TestWalrusPosesKeepTheAnkleInItsReach(t *testing.T) {
	for _, pose := range []walrusPose{walrusLying, walrusHauledOut, walrusSwimming} {
		if _, _, err := defaultWalrusRig().skin(pose, 0, 0); err != nil {
			t.Fatalf("pose %d: %v", pose, err)
		}
	}
	// Turn the right sole about the line square to both the shank and the
	// sole, the way that opens the angle between them: by the room left
	// to the limit and a little more it is refused, a little less it is
	// not.
	rest := walrusRest.tree.Frames(skeleton.Pose{})
	knee, ankle := walrusRest.rod("tibia right")
	base, tip := walrusRest.rod("metatarsal 3 right")
	shank, sole := ankle.Sub(knee).Unit(), tip.Sub(base).Unit()
	opens := shank.Cross(sole).Unit()
	room := walrusRest.profile.AnkleForward - walrusAnkleTurn(walrusRest, rest, "right")
	for _, step := range []struct {
		angle   float64
		refused bool
	}{{room + .1, true}, {room - .1, false}} {
		pose := skeleton.Pose{}.With("astragalus right",
			skeleton.NewTurn(opens, step.angle))
		_, err := walrusPosedFrames(walrusRest, pose)
		if errors.Is(err, errWalrusAnkle) != step.refused {
			t.Fatalf("an ankle opened %.2f rad gave %v", step.angle, err)
		}
	}
}

func TestWalrusFlipperThatCannotReachTheGroundIsRefused(t *testing.T) {
	// Turning the skull moves no flipper, so no angle of it rests one.
	_, err := walrusRestOnGround(walrusRest, skeleton.Pose{}, "skull",
		"metacarpal 3 left")
	if !errors.Is(err, errWalrusGround) {
		t.Fatalf("a flipper the turn cannot move gave %v", err)
	}
	for _, names := range [][2]string{{"humerus left", "no such ray"},
		{"no such bone", "metacarpal 3 left"}} {
		_, err := walrusRestOnGround(walrusRest, skeleton.Pose{}, names[0],
			names[1])
		if !errors.Is(err, errWalrusGround) {
			t.Fatalf("turning %q for %q gave %v", names[0], names[1], err)
		}
	}
}

func TestWalrusPoseThatTurnsNoBoneIsRefused(t *testing.T) {
	pose := skeleton.Pose{}.With("no such joint",
		skeleton.NewTurn(walrusSideAxis, .1))
	if _, err := walrusPosedFrames(walrusRest, pose); !errors.Is(err,
		skeleton.ErrNoJoint) {
		t.Fatalf("a turn of a joint the skeleton lacks gave %v", err)
	}
}

func TestWalrusSwimmingTrailsTheHindFlippers(t *testing.T) {
	frames, err := walrusPosedFrames(walrusRest,
		walrusPoseOf(walrusRest, walrusSwimming))
	if err != nil {
		t.Fatal(err)
	}
	for _, side := range []string{"left", "right"} {
		if turn := walrusAnkleTurn(walrusRest, frames, side); turn > 1 {
			t.Fatalf("%s sole turned %.2f rad from the shank in the water", side, turn)
		}
	}
}

func TestWalrusHauledOutRestsOnItsForeFlippers(t *testing.T) {
	frames := walrusRest.tree.Frames(walrusPoseOf(walrusRest, walrusHauledOut))
	for _, side := range []string{"left", "right"} {
		bone, at := walrusRest.bone("metacarpal 3 " + side)
		height := frames[at].Apply(bone.Head()).Y
		if !(math.Abs(height-walrusFloor-.05) <= .01) {
			t.Fatalf("%s fore flipper at %.3f, not on the ground", side, height)
		}
	}
	_, skull := walrusRest.bone("skull")
	if lift := frames[skull].Apply(walrusFrame.condyle).Y - walrusFrame.condyle.Y; lift < .1 {
		t.Fatalf("the head rose only %.3f m", lift)
	}
}

func TestWalrusUndersideIsFlat(t *testing.T) {
	parts := testWalrusStudy(t, 0, 0)
	hide, _, _ := testWalrusMesh(t, parts[walrusHidePart])
	lowest, resting := math.Inf(1), 0
	for _, v := range hide {
		lowest = math.Min(lowest, v.Y)
		if v.Y < walrusFloor+.005 {
			resting++
		}
	}
	if lowest < walrusFloor-.01 {
		t.Fatalf("hide sinks %.3f below the floor", walrusFloor-lowest)
	}
	if resting < 1000 {
		t.Fatalf("only %d vertices lie on the floor", resting)
	}
}

func TestWalrusSkeletonWithNoSkullIsFaulted(t *testing.T) {
	empty := &walrusBones{tree: skeleton.New(),
		forms: map[string]skeleton.BoneForm{}}
	walrusSkullOf(empty)
	if !errors.Is(empty.fault, errWalrusForm) {
		t.Fatalf("a skeleton with no skull has the fault %v", empty.fault)
	}
	if _, err := walrusPosedFrames(empty, skeleton.Pose{}); !errors.Is(err,
		errWalrusForm) {
		t.Fatalf("posing a skeleton with no skull gave %v", err)
	}
	whole := walrusSkeleton()
	walrusSkullOf(whole)
	if whole.fault != nil {
		t.Fatalf("the walrus's own skeleton has the fault %v", whole.fault)
	}
}

func TestWalrusFollicleOffTheHideIsRefused(t *testing.T) {
	outside := func(maths.Vec) float64 { return 1 }
	still := func(p maths.Vec) maths.Vec { return p }
	if _, err := defaultWalrusRig().walrusVibrissae(1, outside, still); !errors.Is(err,
		errWalrusLandmark) {
		t.Fatalf("bristles with no hide under them gave %v", err)
	}
}

func TestWalrusEyesAndNostrilsLieOnTheOutline(t *testing.T) {
	const near = .003
	outline := testWalrusOutline(t)
	for _, side := range []float64{-1, 1} {
		eye, err := defaultWalrusRig().walrusEyeOn(outline, side)
		if err != nil || !(math.Abs(outline.distance(eye)) <= near) {
			t.Fatalf("side %g: the eye at %v, fault %v", side, eye, err)
		}
		nostril, err := defaultWalrusRig().walrusNostrilOn(outline, side)
		if err != nil || !(math.Abs(outline.distance(nostril)) <= near) {
			t.Fatalf("side %g: the nostril at %v, fault %v", side, nostril, err)
		}
	}
}

func TestWalrusFaceOffTheOutlineIsRefused(t *testing.T) {
	// An outline that ends behind the head has no flank beside the eyes.
	short, err := walrusOutlineOf(walrusSections(walrusStations, 0))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := defaultWalrusRig().walrusFaceOf(short); !errors.Is(err, errWalrusLandmark) {
		t.Fatalf("a face on an outline without a head gave %v", err)
	}
	if _, err := defaultWalrusRig().walrusLids(short); !errors.Is(err, errWalrusLandmark) {
		t.Fatalf("lids on an outline without a head gave %v", err)
	}
	if _, err := defaultWalrusRig().walrusField(short, nil, true); !errors.Is(err, errWalrusLandmark) {
		t.Fatalf("a field on an outline without a head gave %v", err)
	}
}

func TestWalrusVibrissaeInRows(t *testing.T) {
	parts := testWalrusStudy(t, 0, 0)
	for side := 0; side < 2; side++ {
		index := walrusPartIndex(side, walrusWhiskerPart)
		_, faces, _ := testWalrusMesh(t, parts[index])
		strands := len(faces) / walrusStrandFaces
		if strands < 200 || len(faces)%walrusStrandFaces != 0 {
			t.Fatalf("part %d: %d faces make %d strands", index, len(faces), strands)
		}
	}
}

func TestWalrusTuskStandsInItsSocket(t *testing.T) {
	parts := testWalrusStudy(t, 0, 0)
	hide, _, _ := testWalrusMesh(t, parts[walrusHidePart])
	nearest := func(p maths.Vec) float64 {
		best := math.Inf(1)
		for _, v := range hide {
			best = math.Min(best, v.Sub(p).Len())
		}
		return best
	}
	for _, side := range []float64{-1, 1} {
		mouth := walrusTuskPath(side, .28)
		if gap := nearest(mouth); gap > .06 {
			t.Fatalf("side %v: tusk leaves the jaw %.3f m from any hide", side, gap)
		}
	}
}

func TestWalrusFollicleLattice(t *testing.T) {
	drop, across := walrusFollicle(3, 7, -1)
	on := walrusFrame.at(1, across, drop)
	if across >= 0 || defaultWalrusRig().walrusFollicleDistance(on) > 1e-9 {
		t.Fatalf("follicle at %g,%g is not on the lattice", drop, across)
	}
	off := walrusFrame.at(1, across, drop+.005)
	if defaultWalrusRig().walrusFollicleDistance(off) < .003 {
		t.Fatal("a point off the lattice counts as a follicle")
	}
}

func TestWalrusWhiskersAreTheSourcedCount(t *testing.T) {
	if sites := 2 * walrusFollicleRows * walrusFollicleColumns; sites < 400 || sites > 700 {
		t.Fatalf("%d follicle sites, outside 400 to 700", sites)
	}
	if walrusFollicleRows < 13 || walrusFollicleRows > 15 {
		t.Fatalf("%d rows, outside 13 to 15", walrusFollicleRows)
	}
}

func TestWalrusPadShowsBetweenWhiskers(t *testing.T) {
	// A bristle 3 mm thick on a lattice about a centimetre apart covers
	// well under half the pad.
	covered := 2 * walrusStrandRadius / walrusFollicleColumnStep
	if covered > .5 {
		t.Fatalf("bristles cover %.2f of a row", covered)
	}
}

func TestWalrusHideColoursAreReflectances(t *testing.T) {
	texture := walrusHideTexture{}
	for x := -1.8; x < 1.8; x += .07 {
		for y := -.4; y < .4; y += .07 {
			for z := -.9; z < .9; z += .07 {
				c := texture.At(maths.V(x, y, z))
				for _, part := range []float64{c.R, c.G, c.B} {
					if part < 0 || part > 1 || math.IsNaN(part) {
						t.Fatalf("colour %v at %g,%g,%g", c, x, y, z)
					}
				}
			}
		}
	}
}

func TestWalrusHideDiffersByRegion(t *testing.T) {
	texture := walrusHideTexture{}
	back, belly := texture.At(maths.V(-.3, .38, 0)), texture.At(maths.V(-.3, -.40, 0))
	if belly.R <= back.R {
		t.Fatalf("belly %v is not paler than back %v", belly, back)
	}
	fin := texture.At(walrusRayLines[2][2][2])
	if fin.R >= back.R {
		t.Fatalf("flipper %v is not darker than back %v", fin, back)
	}
}

func TestWalrusIvoryPaleAtTheTip(t *testing.T) {
	root, tip := walrusIvoryTexture{}.At(maths.V(1.6, .16, .1)), walrusIvoryTexture{}.At(maths.V(1.6, -.3, .1))
	if tip.B <= root.B {
		t.Fatalf("tip %v is not paler than root %v", tip, root)
	}
}

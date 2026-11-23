package manufacture

import (
	"errors"
	"fmt"
	"strings"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// walrusPose is a posture of the living animal, made by turning joints of
// the skeleton laid out at rest.
type walrusPose int

const (
	// walrusLying is the rest pose: lying on land, belly on the ground,
	// hind flippers turned forward under the body.
	walrusLying walrusPose = iota
	// walrusHauledOut is lying as at rest, the chest on the fore flippers,
	// with the neck raised and the head lifted and held level.
	walrusHauledOut
	// walrusSwimming is the body straight, the head in line with it, the
	// fore flippers laid back along the sides and the hind flippers
	// trailing behind, swept to one side in the stroke.
	walrusSwimming
)

// errWalrusAnkle is a pose that turns an ankle past what it can turn.
var errWalrusAnkle = errors.New("walrus: the pose turns an ankle past its reach")

// walrusSideAxis and walrusUpAxis are the axes most joints turn about: z,
// from side to side, and y, up.
var (
	walrusSideAxis = maths.V(0, 0, 1)
	walrusUpAxis   = maths.V(0, 1, 0)
)

// errWalrusGround is a flipper that no turn of its bone rests on the
// ground.
var errWalrusGround = errors.New("walrus: no turn rests the flipper on the ground")

// walrusPoseOf is the pose made from rest by turning joints. Angles are
// chosen.
func walrusPoseOf(body *walrusBones, pose walrusPose) skeleton.Pose {
	switch pose {
	case walrusHauledOut:
		return walrusHauledOutPose()
	case walrusSwimming:
		return walrusSwimmingPose(body)
	}
	return skeleton.Pose{}
}

// walrusHauledOutPose raises the neck at its root and again halfway up,
// and turns the head back down to level. The chest and the fore flippers
// stay where they rest: the fore limb at rest is at its full reach
// downward, so a lifted chest would take the flippers off the ground.
func walrusHauledOutPose() skeleton.Pose {
	return skeleton.Pose{}.
		With("cervical 7", skeleton.NewTurn(walrusSideAxis, .28)).
		With("cervical 4", skeleton.NewTurn(walrusSideAxis, .16)).
		With("skull", skeleton.NewTurn(walrusSideAxis, -.26))
}

// walrusSwimmingPose lowers the head into line, lays the fore flippers
// back along the body, trails the hind flippers behind and bends the
// loins to one side, as in a stroke.
func walrusSwimmingPose(body *walrusBones) skeleton.Pose {
	pose := skeleton.Pose{}.
		With("cervical 7", skeleton.NewTurn(walrusSideAxis, -.30)).
		With("skull", skeleton.NewTurn(walrusSideAxis, .18)).
		With("lumbar 3", skeleton.NewTurn(walrusUpAxis, .08)).
		With("sacral 1", skeleton.NewTurn(walrusUpAxis, .10))
	for _, side := range []float64{-1, 1} {
		name := walrusSideName(side)
		for _, aim := range walrusSwimmingAims(side) {
			pose = body.tree.Aim(pose, aim.bone+" "+name, aim.way)
		}
	}
	return pose
}

// walrusAim is a bone to point and the way to point it.
type walrusAim struct {
	bone string
	way  maths.Vec
}

// walrusSwimmingAims is the bones the swimming pose points on one side, in
// order: fore flippers back along the flanks, thighs and shanks back, the
// soles trailing straight on from the shanks.
func walrusSwimmingAims(side float64) []walrusAim {
	return []walrusAim{
		{"humerus", maths.V(-.8, -.45, side*.35)},
		{"radius", maths.V(-.9, -.15, side*.40)},
		{"scapholunar", maths.V(-1, -.05, side*.15)},
		{"femur", maths.V(-.6, -.3, side*.5)},
		{"tibia", maths.V(-1, -.1, side*.25)},
		{"astragalus", maths.V(-1, -.02, side*.08)},
	}
}

// walrusRestOnGround turns the bone named about the side axis through its
// joint until the joint at the base of the ray named lies on the ground,
// found by halving the angle between a lift and a drop. It refuses a bone
// or a ray the skeleton does not have, and a swing whose two ends lie on
// the same side of the ground, which has no such angle between them.
func walrusRestOnGround(body *walrusBones, pose skeleton.Pose, bone,
	ray string) (skeleton.Pose, error) {
	const (
		steps       = 30
		clear       = .05
		widest      = .9
		groundLevel = walrusFloor + clear
	)
	at, there := body.tree.Find(ray)
	if _, turned := body.tree.Find(bone); !there || !turned {
		return skeleton.Pose{}, fmt.Errorf("%w: the skeleton lacks %q or %q",
			errWalrusGround, bone, ray)
	}
	base := body.tree.Bone(at)
	height := func(angle float64) float64 {
		tried := pose.Compose(bone, skeleton.NewTurn(walrusSideAxis, angle))
		return body.tree.Frames(tried)[at].Apply(base.Head()).Y - groundLevel
	}
	low, high := -widest, widest
	lowAbove := height(low)
	if highAbove := height(high); (lowAbove > 0) == (highAbove > 0) {
		return skeleton.Pose{}, fmt.Errorf("%w: turning %q leaves %q %.3f "+
			"and %.3f m off it at the ends of its swing", errWalrusGround, bone,
			ray, lowAbove, highAbove)
	}
	for i := 0; i < steps; i++ {
		middle := (low + high) / 2
		above := height(middle)
		if (above > 0) == (lowAbove > 0) {
			low, lowAbove = middle, above
		} else {
			high = middle
		}
	}
	return pose.Compose(bone, skeleton.NewTurn(walrusSideAxis, (low+high)/2)),
		nil
}

// walrusPosedFrames is the skeleton's frames in the pose, refused when
// the pose turns a joint the skeleton lacks or either ankle is turned
// past walrusAnkleForward.
func walrusPosedFrames(body *walrusBones, pose skeleton.Pose) ([]skeleton.Frame, error) {
	if body.fault != nil {
		return nil, body.fault
	}
	if err := body.tree.Check(pose); err != nil {
		return nil, err
	}
	frames := body.tree.Frames(pose)
	for _, side := range []string{"left", "right"} {
		if turn := walrusAnkleTurn(body, frames, side); turn > walrusAnkleForward {
			return nil, fmt.Errorf("%w: %s %.2f rad", errWalrusAnkle, side, turn)
		}
	}
	return frames, nil
}

// walrusHolders is every bone that holds the hide, with its reach: the
// column, the ribs and the breastbone, the skull and the jaw at full reach;
// the bones of the palm, sole and digits, which lie in the flippers, at
// ray; the other limb bones, which lie inside the body wall, at a short
// reach, so that only the skin near them follows them. The tusks hold no
// hide. Reaches chosen.
func walrusHolders(body *walrusBones) []skeleton.Holder {
	const axial, limb, ray = 1., .45, .7
	var holders []skeleton.Holder
	for at := 0; at < body.tree.Len(); at++ {
		name := body.tree.Bone(at).Name()
		if strings.HasPrefix(name, "tusk") {
			continue
		}
		reach := walrusReachOf(name, axial, limb, ray)
		holders = append(holders, skeleton.NewHolder(at,
			body.forms[name].Middle(), reach))
	}
	return holders
}

// walrusReachOf is the reach for the bone of that name: axial bones at
// full reach, the palm, sole and digit bones at ray, the other limb bones
// at limb.
func walrusReachOf(name string, axial, limb, ray float64) float64 {
	for _, prefix := range []string{"cervical", "thoracic", "lumbar", "sacral",
		"caudal", "rib", "sternebra", "skull", "mandible"} {
		if strings.HasPrefix(name, prefix) {
			return axial
		}
	}
	for _, prefix := range []string{"metacarpal", "metatarsal", "fore phalanx",
		"hind phalanx"} {
		if strings.HasPrefix(name, prefix) {
			return ray
		}
	}
	return limb
}

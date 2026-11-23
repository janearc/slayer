package manufacture

import (
	"math"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// The hide is bound to the skeleton: every point of it at rest is held by
// the bones near it and goes where they go in the pose. On top of a pose,
// yaw bends the loins side to side, shared over the lumbar and first
// sacral joints, and phase is the stroke of the flippers.
const (
	// walrusFlipperSwing is the angle a flipper swings through, in radians
	// at the top of the stroke. Chosen.
	walrusFlipperSwing = .30
)

// walrusBendJoints are the joints a yaw is shared over.
var walrusBendJoints = []string{"lumbar 1", "lumbar 2", "lumbar 3", "lumbar 4",
	"lumbar 5", "lumbar 6", "sacral 1"}

// walrusStroke is pose with the loins bent by yaw and the flippers at
// phase of their stroke: the hind flippers sweep side to side at the
// ankle, the fore flippers at the shoulder.
func walrusStroke(pose skeleton.Pose, yaw, phase float64) skeleton.Pose {
	share := yaw / float64(len(walrusBendJoints))
	for _, joint := range walrusBendJoints {
		pose = pose.Compose(joint, skeleton.NewTurn(walrusUpAxis, share))
	}
	swing := walrusFlipperSwing * math.Sin(phase)
	for _, side := range []float64{-1, 1} {
		name := walrusSideName(side)
		pose = pose.Compose("astragalus "+name, skeleton.NewTurn(walrusUpAxis, swing)).
			Compose("humerus "+name, skeleton.NewTurn(walrusUpAxis, side*swing))
	}
	return pose
}

// walrusSkin binds the hide to the skeleton in the pose, bent by yaw and
// stroked by phase. It gives back the function that carries a point of the
// rest hide to its place, and the motion of the skull, which carries every
// tusk, bristle, eye and nostril rigidly with the head.
func walrusSkin(pose walrusPose, yaw, phase float64) (func(maths.Vec) maths.Vec,
	func(maths.Vec) maths.Vec, error) {
	body := walrusRest
	frames, err := walrusPosedFrames(body,
		walrusStroke(walrusPoseOf(body, pose), yaw, phase))
	if err != nil {
		return nil, nil, err
	}
	holders := walrusHolders(body)
	_, skull := body.bone("skull")
	hide := func(p maths.Vec) maths.Vec {
		return skeleton.Carry(p, frames, skeleton.Nearness(p, holders))
	}
	return hide, frames[skull].Apply, nil
}

package manufacture

import (
	"fmt"
	"math"
	"strings"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// A bird's poses, each made by turning its joints from the rest pose:
// standing, the wings folded against the flanks, the trunk raised, the
// neck in an S and the toes on the ground; or gliding, the wings
// spread, the neck as the species carries it and the legs drawn back.

// The stances a bird kind may be made in.
const (
	birdStanding = "standing" // defined: on the ground, wings folded
	birdGliding  = "gliding"  // defined: in the air, wings spread
	birdBones    = "bones"    // defined: the skeleton alone, at rest
	birdOutlined = "outline"  // defined: standing, no feather on
)

// birdStances are the stances, each a kind for every species.
var birdStances = []string{birdStanding, birdGliding, birdBones,
	birdOutlined}

// The fold of a wing at rest on the flank, in radians: the shoulder
// swings the humerus back, the elbow brings the forearm forward, the
// wrist swings the hand back, and the shoulder rolls the folded wing's
// upper face out. Each is picked; none is read from a folded wing.
const (
	birdFoldShoulder = -1.62
	birdFoldElbow    = 2.85
	birdFoldWrist    = -2.8
	birdFoldRoll     = 1.40
	birdFoldDrop     = .02  // a folded feather's droop below the trunk
	birdFoldOut      = -.06 // and its lean in toward the flank
	birdDihedral     = .06  // a gliding wing raised above level
	birdTailClosed   = .05  // a closed tail's spread
	birdGlideBill    = .08  // the bill below level in flight
)

// How the neck bends in each of its thirds, from the chest to the
// head, as multiples of the row's bend at each joint; picked.
var (
	birdStandingNeck  = [3]float64{1, -1.6, 1.2}
	birdRetractedNeck = [3]float64{1.8, -2.6, .8}
)

// The legs drawn back in flight: the ways the shank, the tarsus and
// the toes are aimed for legs held out behind and for feet tucked
// under the tail; picked.
var (
	birdTrailedWays = [3]maths.Vec{maths.V(-.85, -.5, 0),
		maths.V(-1, -.1, 0), maths.V(-1, -.15, 0)}
	birdTuckedWays = [3]maths.Vec{maths.V(-.7, -.7, 0),
		maths.V(-1, .15, 0), maths.V(-1, .1, 0)}
)

// pose is the bird's pose in stance.
func (body *birdBody) pose(stance string) skeleton.Pose {
	switch stance {
	case birdStanding, birdOutlined:
		return body.standing()
	case birdGliding:
		return body.gliding()
	}
	return body.grounded(skeleton.Pose{}, 0)
}

// standing is the bird on the ground: the wings folded, the tail
// closed, the trunk raised by the row's stance about the hips with the
// legs turned back to upright, the neck in an S, the bill level. A bird
// whose folded wing would hang below its toes carries it up; see
// wingRise.
func (body *birdBody) standing() skeleton.Pose {
	pose := body.standingRaised(0)
	if rise := body.wingRise(pose); rise > 0 {
		pose = body.standingRaised(rise)
	}
	return pose
}

// standingRaised is the bird standing with its folded wings turned up
// about the shoulders by the angle rise, in radians.
func (body *birdBody) standingRaised(rise float64) skeleton.Pose {
	m := body.measures
	pose := body.folded(skeleton.Pose{}, rise)
	pose = body.tailClosedPose(pose)
	pose = body.neckCurved(pose, birdStandingNeck)
	pose = body.headAimed(pose, -m.billTilt-m.stance)
	for _, side := range birdSides {
		pose = pose.With("femur-"+side.name,
			skeleton.NewTurn(birdAcross, -m.stance))
	}
	return body.grounded(body.jittered(pose), m.stance)
}

// gliding is the bird in the air: the wings spread and raised a
// little, the neck folded or held out, the legs drawn back.
func (body *birdBody) gliding() skeleton.Pose {
	pose := body.spread(skeleton.Pose{})
	if body.species.Neck == birdRetracted {
		pose = body.neckCurved(pose, birdRetractedNeck)
	} else {
		pose = body.neckStraight(pose)
	}
	pose = body.headAimed(pose, -birdGlideBill)
	ways := birdTrailedWays
	if body.species.Legs == birdTucked {
		ways = birdTuckedWays
	}
	return body.jittered(body.legsDrawn(pose, ways))
}

// The gliding wing's bend, in radians: the arm swept a little forward
// at the shoulder, the forearm back at the elbow, the hand swept back
// at the wrist and its tip drooped, so the leading edge bends at the
// wrist and the span curves down to the tips. Picked, from how a
// soaring bird is seen; no measured glide was read.
const (
	birdGlideShoulder = .25
	birdGlideElbow    = -.30
	birdGlideWrist    = -.45
	birdGlideDroop    = .14
)

// spread is pose with both wings spread for a glide.
func (body *birdBody) spread(pose skeleton.Pose) skeleton.Pose {
	for _, side := range birdSides {
		s := side.sign
		pose = pose.With("humerus-"+side.name,
			skeleton.NewTurn(birdAlong, -s*birdDihedral))
		pose = pose.Compose("humerus-"+side.name,
			skeleton.NewTurn(birdUp, s*birdGlideShoulder))
		for _, bone := range []string{"radius-", "ulna-"} {
			pose = pose.With(bone+side.name,
				skeleton.NewTurn(birdUp, s*birdGlideElbow))
		}
		pose = pose.With("carpometacarpus-"+side.name,
			skeleton.NewTurn(birdUp, s*birdGlideWrist))
		pose = pose.Compose("carpometacarpus-"+side.name,
			skeleton.NewTurn(birdAlong, s*birdGlideDroop))
	}
	return pose
}

// folded is pose with both wings folded and their feathers laid back
// along the flank. Each wing is then turned up as a whole about its
// shoulder by the angle rise, nought for a wing that lies level.
func (body *birdBody) folded(pose skeleton.Pose,
	rise float64) skeleton.Pose {
	for _, side := range birdSides {
		pose = body.armFolded(pose, side.name, side.sign, rise)
	}
	for _, feather := range body.feathers {
		if !birdOnWing(feather.group) {
			continue
		}
		way := maths.V(-1, math.Tan(rise)-birdFoldDrop,
			feather.sign*birdFoldOut)
		pose = body.bones.Aim(pose, feather.name, way)
	}
	return pose
}

// armFolded is pose with the arm and hand of one side folded, and then,
// when rise is above nought, the whole folded arm turned up about the
// shoulder by that angle.
func (body *birdBody) armFolded(pose skeleton.Pose, side string, sign,
	rise float64) skeleton.Pose {
	pose = pose.With("humerus-"+side,
		skeleton.NewTurn(birdUp, sign*birdFoldShoulder))
	pose = pose.Compose("humerus-"+side,
		skeleton.NewTurn(birdAlong, sign*birdFoldRoll))
	if rise > 0 {
		pose = pose.Compose("humerus-"+side,
			skeleton.NewTurn(birdAcross, -rise))
	}
	for _, bone := range []string{"radius-", "ulna-"} {
		pose = pose.With(bone+side,
			skeleton.NewTurn(birdUp, sign*birdFoldElbow))
	}
	return pose.With("carpometacarpus-"+side,
		skeleton.NewTurn(birdUp, sign*birdFoldWrist))
}

// birdOnWing is whether a feather of group grows on the wing.
func birdOnWing(group string) bool {
	switch group {
	case birdPrimary, birdSecondary, birdGreater, birdMedian, birdLesser,
		birdAlula:
		return true
	}
	return false
}

// tailClosedPose is pose with the tail's feathers drawn together.
func (body *birdBody) tailClosedPose(pose skeleton.Pose) skeleton.Pose {
	for _, feather := range body.feathers {
		switch feather.group {
		case birdRectrix, birdUpperCovert, birdUnderCovert:
			way := maths.V(-1, 0, feather.share*birdTailClosed)
			pose = body.bones.Aim(pose, feather.name, way)
		}
	}
	return pose
}

// neckCurved is pose with each cervical turned up or down by the row's
// bend times its third's multiple, the chest's third first.
func (body *birdBody) neckCurved(pose skeleton.Pose,
	thirds [3]float64) skeleton.Pose {
	m := body.measures
	for n := 1; n <= m.cervicals; n++ {
		fromChest := float64(m.cervicals-n) / float64(m.cervicals)
		third := min(int(fromChest*3), 2)
		pose = pose.With(fmt.Sprintf("C%d", n),
			skeleton.NewTurn(birdAcross, m.neckBend*thirds[third]))
	}
	return pose
}

// neckStraight is pose with the neck brought down to level, each
// cervical taking its share of the rise.
func (body *birdBody) neckStraight(pose skeleton.Pose) skeleton.Pose {
	m := body.measures
	for n := 1; n <= m.cervicals; n++ {
		pose = pose.With(fmt.Sprintf("C%d", n), skeleton.NewTurn(birdAcross,
			-birdNeckRise/float64(m.cervicals)))
	}
	return pose
}

// headAimed is pose with the skull turned so the bill points angle
// above level, in the trunk's own frame.
func (body *birdBody) headAimed(pose skeleton.Pose,
	angle float64) skeleton.Pose {
	way := maths.V(math.Cos(angle), math.Sin(angle), 0)
	return body.bones.Aim(pose, "cranium", way)
}

// legsDrawn is pose with each shank, tarsus and toe aimed along ways.
func (body *birdBody) legsDrawn(pose skeleton.Pose,
	ways [3]maths.Vec) skeleton.Pose {
	for _, side := range birdSides {
		pose = body.bones.Aim(pose, "tibiotarsus-"+side.name, ways[0])
		pose = body.bones.Aim(pose, "tarsometatarsus-"+side.name, ways[1])
		for n := range birdToes {
			name := fmt.Sprintf("toe%d-1-%s", n+1, side.name)
			pose = body.bones.Aim(pose, name, ways[2])
		}
	}
	return pose
}

// jittered is pose with every feather turned a little about its root,
// by an angle its seed draws, so no two birds lie alike.
func (body *birdBody) jittered(pose skeleton.Pose) skeleton.Pose {
	for i, feather := range body.feathers {
		angle := birdJitter * body.from.swing(growing, i, 0)
		pose = pose.Compose(feather.name, skeleton.NewTurn(birdUp, angle))
	}
	return pose
}

// grounded is pose with the whole bird turned nose up by pitch about
// its hips and lifted so its lowest toe rests on y nought.
func (body *birdBody) grounded(pose skeleton.Pose,
	pitch float64) skeleton.Pose {
	hip := body.hip(1)
	pivot := maths.V(hip.X, hip.Y, 0)
	turn := maths.Rotation(birdAcross, pitch)
	shift := pivot.Sub(turn.Out(pivot))
	frames := body.bones.Frames(pose.Carried(skeleton.NewFrame(turn, shift)))
	low := math.Inf(1)
	for i := 0; i < body.bones.Len(); i++ {
		bone := body.bones.Bone(i)
		if !strings.HasPrefix(bone.Name(), "toe") {
			continue
		}
		thick, _ := bone.Thickness()
		for _, end := range []maths.Vec{bone.Head(), bone.Tail()} {
			low = math.Min(low, frames[i].Apply(end).Y-birdLegSkin*thick)
		}
	}
	lift := birdUp.Scale(-low)
	return pose.Carried(skeleton.NewFrame(turn, shift.Add(lift)))
}

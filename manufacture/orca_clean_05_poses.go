package manufacture

import (
	"math"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// orcaAcross is the axis across the body, about which the column bends
// up and down.
var orcaAcross = maths.V(0, 0, 1)

// orcaAlong is the axis along the body, nose forward, about which a
// flipper turns out from the flank.
var orcaAlong = maths.V(1, 0, 0)

// The swim: a wave of bending that runs back along the lumbars and
// caudals, growing toward the flukes, as a cetacean's vertical stroke
// is concentrated in the rear of the body (S6 of
// doc/model/bioform/orca/SOURCES.md). The numbers are chosen; the
// stroke is not fitted to recorded motion.
const (
	orcaSwimTurn = .012  // largest turn of one joint, in radians
	orcaSwimLag  = 1.2   // phase a joint trails per metre behind, in radians
	orcaSwimFrom = -.40  // x where the column begins to bend
	orcaSwimFull = -2.30 // x behind which every joint bends fully
)

// orcaSwimPose is the skeleton at phase of the swim: each lumbar and
// caudal turned about the across axis at its head.
func orcaSwimPose(bones *skeleton.Skeleton, phase float64) skeleton.Pose {
	pose := skeleton.Pose{}
	for i := 0; i < bones.Len(); i++ {
		bone := bones.Bone(i)
		region, _ := orcaVertebraOf(bone.Name())
		x := bone.Head().X
		share := orcaStep(orcaSwimFrom, orcaSwimFull, x)
		if region != "L" && region != "Ca" || share == 0 {
			continue
		}
		angle := orcaSwimTurn * share *
			math.Sin(phase+orcaSwimLag*(orcaSwimFrom-x))
		pose = pose.With(bone.Name(), skeleton.NewTurn(orcaAcross, angle))
	}
	return pose
}

// The spy-hop: the animal upright in the water with its head raised
// clear, the flippers turned out from the flanks and the tail curled a
// little under the body as it holds itself there. Every number is
// chosen.
const (
	orcaSpyHopTilt   = 1.40 // how far the body turns head up, in radians
	orcaSpyHopSpread = .45  // how far each flipper turns out, in radians
	orcaSpyHopCurl   = .010 // turn of each caudal toward the belly, in radians
)

// orcaSpyHopPivot is the point the body is turned upright about, near
// the middle of its mass; chosen.
var orcaSpyHopPivot = maths.V(.9, 0, 0)

// orcaSpyHopPose is the skeleton spy-hopping: the whole turned head up
// about the pivot, the flippers turned out at the shoulder, each caudal
// turned toward the belly.
func orcaSpyHopPose(bones *skeleton.Skeleton) skeleton.Pose {
	pose := skeleton.Pose{}.Carried(skeleton.TurnAbout(orcaSpyHopPivot,
		orcaAcross, orcaSpyHopTilt))
	pose = pose.With("right-humerus",
		skeleton.NewTurn(orcaAlong, -orcaSpyHopSpread))
	pose = pose.With("left-humerus",
		skeleton.NewTurn(orcaAlong, orcaSpyHopSpread))
	for i := 0; i < bones.Len(); i++ {
		name := bones.Bone(i).Name()
		if region, _ := orcaVertebraOf(name); region == "Ca" {
			pose = pose.With(name, skeleton.NewTurn(orcaAcross,
				orcaSpyHopCurl))
		}
	}
	return pose
}

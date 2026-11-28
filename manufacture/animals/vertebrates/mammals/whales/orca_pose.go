package whales

import (
	"math"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/manufacture/animals/vertebrates/mammals"
	"github.com/janearc/slayer/maths"
)

// How the skin is bound to the bones. Each point of the trunk follows
// the skull or the vertebrae nearest it along the body, each by a share
// that falls from one at the vertebra's middle to zero a reach away; a
// point on a flipper follows that flipper's humerus, the forearm and
// hand being one stiff plate in a cetacean. The reach and the width of
// the flipper's hold are chosen: wide enough that no crease shows.
const (
	orcaBlendReach  = 1.5  // a vertebra's hold, in its own lengths each side
	orcaFlipperHold = .05  // width over which the flipper's hold fades, in metres
	orcaFlipperLean = .015 // how far the hold's edge lies toward the trunk
)

// orcaAxis is the skull and the column, front to back.
func orcaAxis(bones *skeleton.Skeleton) []mammals.Span {
	spans := []mammals.Span{}
	for i := 0; i < bones.Len(); i++ {
		bone := bones.Bone(i)
		region, _ := orcaVertebraOf(bone.Name())
		if region == "" && bone.Name() != "cranium" {
			continue
		}
		head, tail := bone.Head().X, bone.Tail().X
		spans = append(spans, mammals.Span{Bone: i, Middle: (head + tail) / 2,
			Run: math.Abs(head - tail)})
	}
	return spans
}

// orcaFlipperShare is how far rest point p belongs to a flipper rather
// than to the trunk: one on the paddle, zero on the body, eased across
// the fillet between.
func (model *orcaModel) orcaFlipperShare(p maths.Vec) float64 {
	lead := orcaFlipper(p) - model.orcaTrunk(p) + orcaFlipperLean
	return orcaStep(orcaFlipperHold/2, -orcaFlipperHold/2, lead)
}

// orcaBind is the bones that move rest point p and the share of each.
func (model *orcaModel) orcaBind(bones *skeleton.Skeleton, spans []mammals.Span,
	p maths.Vec,
) []skeleton.Influence {
	hold := mammals.AxialHold(spans, p.X, orcaBlendReach, []skeleton.Influence{})
	flipper := model.orcaFlipperShare(p)
	if flipper == 0 {
		return hold
	}
	side := "right-humerus"
	if p.Z < 0 {
		side = "left-humerus"
	}
	total := 0.
	for _, each := range hold {
		total += each.Weight()
	}
	scaled := []skeleton.Influence{}
	for _, each := range hold {
		scaled = append(scaled, skeleton.NewInfluence(each.Bone(),
			each.Weight()*(1-flipper)/total))
	}
	return append(scaled, skeleton.NewInfluence(orcaMust(bones, side),
		flipper))
}

// orcaBinding binds the relieved geometry using the unrelieved points
// to choose the weights, so scars and small ridges stay on the skin.
func (model *orcaModel) orcaBinding() (*skeleton.Skin, error) {
	model.bindingOnce.Do(func() {
		bones, err := model.orcaSkeleton()
		if err != nil {
			model.bindingErr = err
			return
		}
		spans := orcaAxis(bones)
		rest := model.orcaRest()
		model.binding, model.bindingErr = bones.BindSkin(skeleton.SkinInput{
			Points: rest.relieved,
			Weigh: func(index int, _ maths.Vec) []skeleton.Influence {
				return model.orcaBind(bones, spans, rest.points[index])
			},
		})
	})
	return model.binding, model.bindingErr
}

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

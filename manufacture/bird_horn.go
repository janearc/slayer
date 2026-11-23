package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// The horny parts of a bird's foot and leg. Every toe ends in a claw, a
// sheath of horn over the toe's last bone that reaches a little past it
// and curves down. The front of the tarsus is covered in scutes, broad
// scales one above another, each wrapping the front of the leg.
//
// That every toe has a claw is the anatomy of the foot and no count to
// cite; the claws are as many as the toes in birdToes. The scutes and the
// sizes below are picked, to look as a heron's, gull's or tern's leg is
// seen close; no count of scutes was read.

// The claw's sheath, as shares: how much of the last bone it covers, how
// far past the bone's end it reaches in that bone's lengths, its root's
// radius of the bone's skin, and how far its tip lies below the toe's
// axis in skins, so that the tip touches where the toe does. All picked.
const (
	birdClawCover = .5
	birdClawReach = .6
	birdClawGirth = 1.3
	birdClawDrop  = .8
)

// The claw's stations, root to tip, as shares of its girth: narrower
// across than deep, as a claw is flattened from the sides; picked.
var (
	birdClawWide = []float64{.8, .7, .5, .28, .06}
	birdClawDeep = []float64{1, .9, .65, .38, .08}
)

// The scutes on each tarsus: how many, how much of the room each has
// that it fills in height, how far round the front of the leg it wraps
// either side of the middle in radians (50 degrees), and its thickness
// outward of the leg's radius. All picked.
const (
	birdScutes         = 6
	birdScuteHeight    = .8
	birdScuteHalfAngle = .87
	birdScuteThick     = .15
)

// birdRodFlare is the exponent of the rod maker's flare toward the ends
// of a long bone: the radius there is the shaft's plus the rest of the
// end's radius times (2t - 1) to this power, t along the bone.
const birdRodFlare = 4

// clawPieces are the claws, one on the last bone of each toe.
func (body *birdBody) clawPieces() []birdPiece {
	pieces := []birdPiece{}
	for _, side := range birdSides {
		for n, toe := range birdToes {
			name := fmt.Sprintf("toe%d-%d-%s", n+1, toe.phalanges, side.name)
			at, _ := body.bones.Find(name)
			bone := body.bones.Bone(at)
			thick, _ := bone.Thickness()
			pieces = append(pieces, birdPiece{bone: name, region: "feet",
				form: birdClaw(bone.Head(), bone.Tail(), birdLegSkin*thick)})
		}
	}
	return pieces
}

// birdClaw is the claw over the toe's last bone from head to tail, whose
// skin is skin thick: from half way along the bone to past its end, the
// tip carried down.
func birdClaw(head, tail maths.Vec, skin float64) skeleton.Profile {
	step := tail.Sub(head)
	root := head.Add(step.Scale(1 - birdClawCover))
	tip := tail.Add(step.Scale(birdClawReach)).
		Add(birdUp.Scale(-birdClawDrop * skin))
	girth := birdClawGirth * skin
	return skeleton.NewProfile(root, tip, birdUp,
		birdScaled(birdClawWide, girth), birdScaled(birdClawDeep, girth))
}

// scutePieces are the scutes down the front of each tarsus, evenly
// spaced, each in the middle of its share of the bone's length.
func (body *birdBody) scutePieces() []birdPiece {
	m := body.measures
	skin := birdLegSkin * birdLimbBone * m.length
	pieces := []birdPiece{}
	for _, side := range birdSides {
		name := "tarsometatarsus-" + side.name
		at, _ := body.bones.Find(name)
		bone := body.bones.Bone(at)
		room := bone.Length() / birdScutes
		for i := 0; i < birdScutes; i++ {
			share := (float64(i) + .5) / birdScutes
			middle := bone.Head().Add(bone.Tail().Sub(bone.Head()).Scale(share))
			flare := math.Pow(2*share-1, birdRodFlare)
			radius := skin * (birdShaft + (1-birdShaft)*flare)
			pieces = append(pieces, birdPiece{bone: name, region: "legs",
				form: skeleton.NewArc(birdScuteBand(middle, radius),
					birdScuteThick*radius, birdScuteHeight*room/2, birdUp)})
		}
	}
	return pieces
}

// birdScuteBand are the four points of the curve that wraps the front of
// the leg, +x, at middle, radius out from its axis: the Bezier curve that
// runs along that circle from the half angle behind the middle to the half
// angle before it, with handles of 4/3 tan(angle / 2) of the radius.
func birdScuteBand(middle maths.Vec, radius float64) [4]maths.Vec {
	cos, sin := math.Cos(birdScuteHalfAngle), math.Sin(birdScuteHalfAngle)
	handle := 4.0 / 3 * math.Tan(birdScuteHalfAngle/2) * radius
	at := func(x, z float64) maths.Vec {
		return maths.V(middle.X+x, middle.Y, middle.Z+z)
	}
	return [4]maths.Vec{at(radius*cos, -radius*sin),
		at(radius*cos+handle*sin, -radius*sin+handle*cos),
		at(radius*cos+handle*sin, radius*sin-handle*cos),
		at(radius*cos, radius*sin)}
}

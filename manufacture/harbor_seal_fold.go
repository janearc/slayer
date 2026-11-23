package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The neck folds. When the head nods, the skin on the inside of the bend
// has nowhere to go and stands out in rolls across the neck. All of these
// numbers are chosen.
const (
	// sealNeckStart is the X, in metres, where the neck's rolls begin.
	sealNeckStart = .30
	// sealNeckEnd is the X where they end, at the back of the skull.
	sealNeckEnd = .60
	// sealNeckBlend is how far along X a roll's window fades in and out.
	sealNeckBlend = .05
	// sealNeckPeriod is the distance between two rolls.
	sealNeckPeriod = .075
	// sealNeckDepth is how far a roll stands out at the largest nod.
	sealNeckDepth = .016
	// sealNeckLimit is the largest head pitch, in radians, the rig allows.
	sealNeckLimit = .92
)

// The flipper-root folds. A turned flipper gathers the trunk skin around
// its root into rings.
const (
	// sealRootPeriod is the distance between two rings around a root.
	sealRootPeriod = .05
	// sealRootDepth is how far a ring stands out at a full turn.
	sealRootDepth = .008
	// sealRootTurn is the flipper turn, in radians, that gives full depth.
	sealRootTurn = .6
	// sealRootNear and sealRootFar bound where rings form, as the distance
	// from the root in metres, each with a blend.
	sealRootNear, sealRootFar, sealRootBlend = .03, .13, .03
)

// fold is how far a vertex stands out along its skin's normal where the
// pose bunches the skin, as an offset in the spine's local axes. At rest
// there is none.
func (rig *SealRig) fold(index int, pose SealRigPose) maths.Vec {
	vertex := rig.vertices[index]
	normal := rig.normals[index]
	amount := sealNeckFold(vertex.point, normal, pose.HeadPitch)
	if vertex.weight > 0 {
		limb := rig.limbs[vertex.limb]
		amount += sealRootFold(vertex.point.Sub(limb.root).Len(),
			pose.angle(vertex.limb))
	}
	if amount == 0 {
		return maths.Vec{}
	}
	lifted := pose.nod(vertex.point.Add(normal)).Sub(pose.nod(vertex.point))
	return lifted.Scale(amount)
}

// sealNeckFold is the height of the neck roll at a point whose skin faces
// along normal, for a head pitch. Only the inside of the bend rolls: the
// underside when the head nods down, the nape when it tips back.
func sealNeckFold(point, normal maths.Vec, pitch float64) float64 {
	inside := math.Copysign(1, pitch) * normal.Y
	window := maths.Smooth(sealNeckStart, sealNeckStart+sealNeckBlend, point.X) *
		(1 - maths.Smooth(sealNeckEnd-sealNeckBlend, sealNeckEnd, point.X))
	ripple := .5 * (1 - math.Cos(2*math.Pi*(point.X-sealNeckStart)/sealNeckPeriod))
	return sealNeckDepth * math.Abs(pitch) / sealNeckLimit * window * ripple *
		maths.Smooth(-.1, .6, inside)
}

// sealRootFold is the height of the ring a given distance from a flipper
// root, for the flipper's turn.
func sealRootFold(distance, angle float64) float64 {
	zone := maths.Smooth(sealRootNear, sealRootNear+sealRootBlend, distance) *
		(1 - maths.Smooth(sealRootFar-sealRootBlend, sealRootFar, distance))
	ripple := .5 * (1 - math.Cos(2*math.Pi*distance/sealRootPeriod))
	return sealRootDepth * math.Min(1, math.Abs(angle)/sealRootTurn) * zone * ripple
}

package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// pterosaurSides are the two wings' signs along the span: left, then right.
var pterosaurSides = [2]float64{-1, 1}

// pterosaurPose is every joint the skin and membrane bind to, in animal
// space: x runs tail to head, y up, z along the span. The wing chain is
// the earlier rig's (hatzSourceJoints) and is not rebuilt here.
type pterosaurPose struct {
	// arm is, per side, the shoulder, elbow, wrist, the end of the
	// metacarpal and the wing finger's tip.
	arm [2][5]maths.Vec
	// leg is, per side, the hip, knee, ankle and the ball of the foot.
	leg [2][4]maths.Vec
	// neck is the spine of the neck from the chest to the occiput.
	neck []maths.Vec
	// headPitch turns the skull nose down, in radians.
	headPitch float64
	// gape is how far the lower jaw is open, in radians.
	gape float64
	// lift is how high the trunk stands above the ground, in metres.
	lift float64
	// standing is true on all fours with the wings folded.
	standing bool
}

// The numbers the poses use.
const (
	// neckBaseX and neckBaseY are where the neck leaves the chest, in
	// the flight pose, in metres. Authored.
	neckBaseX = .36
	neckBaseY = .17
	// flightNeckStart and flightNeckEnd are the neck's pitch up from
	// the chest and at the skull in flight, in radians. Authored.
	flightNeckStart = .42
	flightNeckEnd   = .02
	// standNeckStart and standNeckEnd are the same on the ground.
	standNeckStart = 1.0
	standNeckEnd   = -.30
	// flightHeadDown and standHeadDown are the skull's pitch down from
	// level, in radians. Authored: azhdarchid skulls are held with the
	// beak down.
	flightHeadDown = .20
	standHeadDown  = .42
	// neckBob is how far the neck's pitch follows the wingbeat, in
	// radians. Authored.
	neckBob = .05
	// legSwing is how far the thigh follows the wingbeat, as a rise of
	// its direction. Authored.
	legSwing = .12
	// handLength and the standing wrist target are in metres.
	// Authored: the manus is planted a little ahead of the shoulder.
	standWristX      = .62
	standWristHeight = .17
	standWristSpread = .52
	// standAnkleX, standAnkleHeight and standAnkleSpread place the
	// planted ankle. Authored.
	standAnkleX      = -.95
	standAnkleHeight = .13
	standAnkleSpread = .34
)

// smoothShare is a rise from 0 to 1 with no slope at either end.
func smoothShare(share float64) float64 {
	share = clamp(share, 0, 1)
	return share * share * (3 - 2*share)
}

// neckSpine is the neck's vertebral points from base, the pitch turning
// smoothly from start to end along its length.
func neckSpine(base maths.Vec, start, end float64) []maths.Vec {
	spine := []maths.Vec{base}
	step := pterosaurNeckLength / float64(pterosaurNeckJoints-1)
	for i := 1; i < pterosaurNeckJoints; i++ {
		share := (float64(i) - .5) / float64(pterosaurNeckJoints-1)
		pitch := start + (end-start)*smoothShare(share)
		heading := maths.V(math.Cos(pitch), math.Sin(pitch), 0)
		spine = append(spine, spine[i-1].Add(heading.Scale(step)))
	}
	return spine
}

// twoBoneJoint is the middle joint of a chain of two bones, first and
// second long, from root toward target, bending toward pole. The second
// result is where the chain's end really lands: the target, or the
// nearest point the bones can reach along the line to it.
func twoBoneJoint(root, target maths.Vec, first, second float64,
	pole maths.Vec) (maths.Vec, maths.Vec) {
	const margin = 1e-3
	line := target.Sub(root)
	reach := clamp(line.Len(), math.Abs(first-second)+margin,
		first+second-margin)
	direction := line.Unit()
	along := (first*first - second*second + reach*reach) / (2 * reach)
	height := math.Sqrt(math.Max(0, first*first-along*along))
	bend := pole.Sub(direction.Scale(pole.Dot(direction))).Unit()
	joint := root.Add(direction.Scale(along)).Add(bend.Scale(height))
	return joint, root.Add(direction.Scale(reach))
}

// legChain is the hip, knee, ankle and ball of the foot for a hip and the
// directions of thigh, shank and foot.
func legChain(hip maths.Vec, directions [3]maths.Vec) [4]maths.Vec {
	lengths := [3]float64{pterosaurThigh, pterosaurShank, pterosaurFoot}
	chain := [4]maths.Vec{hip}
	for i, direction := range directions {
		chain[i+1] = chain[i].Add(direction.Unit().Scale(lengths[i]))
	}
	return chain
}

// flightLeg is the hind limb trailing behind the body, the thigh swinging
// a little with the wingbeat phase.
func flightLeg(phase, side float64) [4]maths.Vec {
	swing := legSwing * math.Sin(phase-.6)
	hip := maths.V(pterosaurHipAt+.10, -.10, side*.17)
	return legChain(hip, [3]maths.Vec{
		maths.V(-.55, -.60+swing, side*.32),
		maths.V(-.70, .45-swing, side*.30),
		maths.V(-.80, -.30, side*.10)})
}

// pterosaurFlight is the pose in flight at a wingbeat phase, in radians:
// the earlier rig's wing chain, the legs trailing, the neck level.
func pterosaurFlight(phase, gape float64) pterosaurPose {
	pose := pterosaurPose{gape: gape}
	for i, side := range pterosaurSides {
		copy(pose.arm[i][:], hatzSourceJoints(phase, side))
		pose.leg[i] = flightLeg(phase, side)
	}
	bob := neckBob * math.Sin(phase-.4)
	base := maths.V(neckBaseX, neckBaseY, 0)
	pose.neck = neckSpine(base, flightNeckStart+bob, flightNeckEnd-bob)
	pose.headPitch = flightHeadDown - flightNeckEnd + bob
	return pose
}

// standingArm is the wing chain folded with the wrist planted: the
// shoulder and elbow reach to a wrist on the ground, the metacarpal rises
// back and the wing finger folds along the flank. The four bone lengths
// are the rig's.
func standingArm(side float64, lift float64) [5]maths.Vec {
	shoulder := hatzSourceJoints(0, side)[0].Add(maths.V(0, lift, 0))
	target := maths.V(standWristX, standWristHeight, side*standWristSpread)
	elbow, wrist := twoBoneJoint(shoulder, target, hatzSourceBones[0],
		hatzSourceBones[1], maths.V(-.8, .1, side*.6))
	back := wrist.Add(maths.V(-.45, .85, side*.10).Unit().
		Scale(hatzSourceBones[2]))
	tip := back.Add(maths.V(-.97, .20, side*.05).Unit().
		Scale(hatzSourceBones[3]))
	return [5]maths.Vec{shoulder, elbow, wrist, back, tip}
}

// standingLeg is the hind limb planted: the hip over an ankle on the
// ground, the knee forward, the foot flat.
func standingLeg(side, lift float64) [4]maths.Vec {
	hip := maths.V(pterosaurHipAt, lift-.02, side*.17)
	ankleTarget := maths.V(standAnkleX, standAnkleHeight,
		side*standAnkleSpread)
	knee, ankle := twoBoneJoint(hip, ankleTarget, pterosaurThigh,
		pterosaurShank, maths.V(1, 0, side*.3))
	foot := maths.V(.94, -.31, side*.12).Unit().Scale(pterosaurFoot)
	ball := ankle.Add(foot)
	return [4]maths.Vec{hip, knee, ankle, ball}
}

// pterosaurStanding is the pose on all fours with the wings folded.
func pterosaurStanding(gape float64) pterosaurPose {
	lift := pterosaurStandLift
	pose := pterosaurPose{gape: gape, lift: lift, standing: true}
	for i, side := range pterosaurSides {
		pose.arm[i] = standingArm(side, lift)
		pose.leg[i] = standingLeg(side, lift)
	}
	base := maths.V(neckBaseX, neckBaseY+lift, 0)
	pose.neck = neckSpine(base, standNeckStart, standNeckEnd)
	pose.headPitch = standHeadDown - standNeckEnd
	return pose
}

// pterosaurGlideSpan is the wingspan of the rig at glide, in metres, tip
// to tip: the span the skull's share is taken of.
func pterosaurGlideSpan() float64 {
	tip := hatzSourceJoints(0, 1)[4]
	return 2 * tip.Z
}

// pterosaurSkullLength is the skull's length in metres.
func pterosaurSkullLength() float64 {
	return pterosaurSkullShareOfSpan * pterosaurGlideSpan()
}

// forwardAnchor is the point on the neck's base where the forward
// membrane's front edge is fixed, for a side.
func (pose pterosaurPose) forwardAnchor(sideIndex int) maths.Vec {
	base := pose.neck[0]
	return maths.V(base.X+.05, base.Y-.04, pterosaurSides[sideIndex]*.17)
}

// pteroidTip is the end of the pteroid bone, which springs from the wrist
// toward the neck's base at its fixed length.
func (pose pterosaurPose) pteroidTip(sideIndex int) maths.Vec {
	wrist := pose.arm[sideIndex][2]
	toward := pose.forwardAnchor(sideIndex).Sub(wrist).Unit()
	return wrist.Add(toward.Scale(pterosaurPteroidLength))
}

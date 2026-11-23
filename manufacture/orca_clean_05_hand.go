package manufacture

import (
	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// The flipper's bones, laid out inside the paddle of orcaFlipper. The
// phalanges per digit are those the specimen record S1 of
// doc/model/bioform/orca/SOURCES.md lists for its right manus: digit
// one one phalanx, digit two four, digit three three, digit four two,
// digit five one. That specimen is a juvenile male.
//
// Every length, spread and splay is chosen, so that the hand fills the
// paddle as a cetacean's does, inside its outline.
const (
	orcaShoulderAlong = .10  // the shoulder joint, along the paddle from its root
	orcaHumerusLong   = .14  // the humerus
	orcaForearmLong   = .15  // the radius and the ulna
	orcaWristLong     = .06  // the carpals
	orcaHumerusThick  = .045 // half thickness of the humerus at the shoulder
	orcaScapulaThick  = .06  // half thickness of the shoulder blade's broad end
)

// orcaScapulaTop is where the shoulder blade's broad end lies on the
// right flank, above and behind the shoulder, outside the ribs; chosen.
var orcaScapulaTop = maths.V(1.62, .08, .44)

// orcaHandFrame is the right paddle's shoulder joint, the direction
// along its middle (trailing as the paddle's outline does) and the
// direction across it toward the leading edge.
func orcaHandFrame() (shoulder, along, across maths.Vec) {
	root := maths.V(orcaFlipperRootX, orcaFlipperRootY, orcaFlipperRootZ)
	shoulder = root.Add(orcaFlipperSpan.Scale(orcaShoulderAlong))
	along = orcaFlipperSpan.Sub(orcaFlipperChordway.Scale(
		orcaFlipperSweep)).Unit()
	across = orcaFlipperChordway.Sub(along.Scale(
		orcaFlipperChordway.Dot(along))).Unit()
	return shoulder, along, across
}

// orcaArmAndDigits is hand with the arm and five digits, from the
// leading edge to the trailing: each its knuckle's place across the
// wrist, its splay in radians, its metacarpal and then its phalanges.
func orcaArmAndDigits(hand skeleton.Hand) skeleton.Hand {
	return hand.Arm(orcaHumerusLong, orcaForearmLong, orcaWristLong).
		Digit(.10, .25, .07, .06).
		Digit(.045, .06, .10, .10, .09, .08, .06).
		Digit(0, -.02, .10, .10, .09, .08).
		Digit(-.05, -.12, .09, .08, .07).
		Digit(-.10, -.28, .07, .06)
}

// orcaMirror is p reflected across the midline when side is minus one.
func orcaMirror(p maths.Vec, side float64) maths.Vec {
	return maths.V(p.X, p.Y, side*p.Z)
}

// orcaHand is one side's flipper bones; the left is the right reflected.
func orcaHand(side float64) skeleton.Hand {
	shoulder, along, across := orcaHandFrame()
	return orcaArmAndDigits(skeleton.NewHand(orcaMirror(shoulder, side),
		orcaMirror(along, side), orcaMirror(across, side),
		orcaHumerusThick))
}

// orcaFlippers adds each side's shoulder blade, jointed to the second
// thoracic, and the flipper's bones jointed to the shoulder blade.
func orcaFlippers(bones *skeleton.Skeleton) error {
	shoulder, _, _ := orcaHandFrame()
	for _, side := range orcaSides {
		top := orcaMirror(orcaScapulaTop, side.sign)
		joint := orcaMirror(shoulder, side.sign)
		blade := "scapula-" + side.name
		if err := bones.Add(blade, "T2", top, joint, orcaScapulaThick,
			orcaHumerusThick); err != nil {
			return err
		}
		if err := bones.Hand(side.name+"-", blade,
			orcaHand(side.sign)); err != nil {
			return err
		}
	}
	return nil
}

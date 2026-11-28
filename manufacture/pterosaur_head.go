package manufacture

import (
	"github.com/janearc/slayer/maths"
)

// The numbers the head's pieces use.
const (
	// beakRearShare is where the skin-covered rear of the skull ends and
	// the horn sheath, which begins at pterosaurBeakSheath, has long
	// since taken over, as a share of the skull's length.
	beakRearShare = .40
	// ballRings and ballSides are the rings of a small ellipsoid.
	ballRings = 12
	ballSides = 20
	// lidReach, lidRadius, irisRadius and pupilRadius size the eye's
	// parts, in metres. Authored.
	lidReach    = .04
	lidRadius   = .062
	irisRadius  = .028
	pupilRadius = .013
	// eyeSink is how far the eye sits inside the skull's surface, in
	// metres.
	eyeSink = .015
	// tongueRise is how far the tongue stands on the lower jaw's top.
	tongueRise = .002
)

// headFrame is the frame that places the pose's head on its neck.
func (pose pterosaurPose) headFrame() headFrame {
	return headFrame{origin: pose.neck[len(pose.neck)-1],
		pitch: pose.headPitch}
}

// headPieces are the skull's pieces in the animal's space: the rear of
// the skull, the horn beak, the crest, the lower jaw in two parts with the
// tongue, the palate, and both eyes.
func (pose pterosaurPose) headPieces(looks pterosaurLooks) []pterosaurPiece {
	length := pterosaurSkullLength()
	frame := pose.headFrame()
	fixed := frame.place
	open := func(point maths.Vec) maths.Vec {
		pivot := jawPivot(length)
		return frame.place(turnedDown(point, pivot, pose.gape))
	}
	up := maths.V(0, 1, 0)
	rear := tubeLoft(skullKeys(0, beakRearShare, 0, length), headSpacing,
		headSides, up, func(along, angle float64) float64 {
			return skullRelief(along, angle, length)
		})
	sheathKeys := skullKeys(pterosaurBeakSheath, 1, sheathGrow, length)
	sheath := tubeLoft(sheathKeys, headSpacing, headSides, up, sheathRelief)
	crest := tubeLoft(crestKeys(length), headSpacing, headSides, up, nil)
	jawRear := tubeLoft(jawKeys(jawBack, pterosaurBeakSheath, length),
		headSpacing, headSides, up, nil)
	jawFront := tubeLoft(jawKeys(pterosaurBeakSheath-.02, jawEnd, length),
		headSpacing, headSides, up, sheathRelief)
	pieces := []pterosaurPiece{
		solidPiece("skull", rear.mapped(fixed), looks.hide),
		solidPiece("beak", sheath.mapped(fixed), looks.beak),
		solidPiece("crest", crest.mapped(fixed), looks.beak),
		solidPiece("jaw-rear", jawRear.mapped(open), looks.hide),
		solidPiece("jaw-beak", jawFront.mapped(open), looks.beak),
		solidPiece("tongue", tongueSurface(length).mapped(open),
			looks.mouth),
		solidPiece("palate", palateSurface(length).mapped(fixed),
			looks.mouth),
	}
	for _, side := range pterosaurSides {
		pieces = append(pieces,
			eyePieces(side, length, fixed, looks)...)
	}
	return pieces
}

// tongueSurface is the tongue: a flat strip lying on the lower jaw's top,
// from just in front of the hinge to most of the way to the tip.
func tongueSurface(length float64) tubeSurface {
	keys := skullKeys(.12, .80, 0, length)
	for i, key := range keys {
		_, height := skullSection(key.centre.X / length)
		keys[i].centre.Y = -mouthLine*height + tongueRise
		keys[i].halfWidth = .45 * key.halfWidth
		keys[i].halfHeight = .012
		keys[i].squareness = 2.2
	}
	return tubeLoft(keys, headSpacing, headSides, maths.V(0, 1, 0), nil)
}

// palateSurface is the roof of the mouth: a flat strip on the underside
// of the upper beak, in view when the jaw is open.
func palateSurface(length float64) tubeSurface {
	keys := skullKeys(.12, .80, 0, length)
	for i, key := range keys {
		_, height := skullSection(key.centre.X / length)
		keys[i].centre.Y = -height
		keys[i].halfWidth = .5 * key.halfWidth
		keys[i].halfHeight = .006
		keys[i].squareness = 2.2
	}
	return tubeLoft(keys, headSpacing, headSides, maths.V(0, 1, 0), nil)
}

// eyePieces are one side's eye: the eyeball, the iris and the pupil on its
// face, the lid ringing it, set in the skull at the eye's station.
func eyePieces(side, length float64, place func(maths.Vec) maths.Vec,
	looks pterosaurLooks) []pterosaurPiece {
	width, _ := skullSection(eyeSocketAt)
	at := eyeSocketAt * length
	across := width - eyeSink
	axis := maths.V(0, 0, side)
	ellipsoid := func(depth, reach, radius float64) tubeSurface {
		centre := maths.V(at, eyeSocketHeight, side*(across+depth))
		return ellipsoidSurface(centre, axis, reach, radius, radius,
			ballRings, ballSides).mapped(place)
	}
	return []pterosaurPiece{
		solidPiece("eyelid", ellipsoid(.005, lidReach, lidRadius),
			looks.hide),
		solidPiece("eyeball", ellipsoid(.01, pterosaurEyeRadius,
			pterosaurEyeRadius), looks.eye),
		solidPiece("iris", ellipsoid(.048, .012, irisRadius),
			looks.iris),
		solidPiece("pupil", ellipsoid(.054, .008, pupilRadius),
			looks.pupil),
	}
}

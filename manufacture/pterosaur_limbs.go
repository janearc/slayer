package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// The limbs are made of the skeleton's own joints: every bone is a loft
// between two joints of the pose, and every joint carries a ball larger
// than the bones that meet in it, so no pose can open a gap or pinch the
// skin at the shoulder, the elbow, the wrist or the ankle. The wing's
// bones are thick near the body, where muscle is, and thin toward the
// finger's tip, where there is only bone and skin.

// boneRing is a control ring of a bone: its share of the way from the
// bone's first joint to its second, and its half width and half height.
type boneRing struct{ share, halfWidth, halfHeight float64 }

// The numbers the limbs use.
const (
	// limbSides and limbSpacing are the rings' size on a limb.
	limbSides   = 20
	limbSpacing = .03
	// digitDown is how far the hand's digits point down for each part
	// forward. Authored.
	digitDown = .55
	// claw sizes, in metres: the handClaw is on the wing's free digits
	// and the footClaw on the toes. Authored.
	handClaw = .075
	footClaw = .06
	// clawCurl is how far a claw's tip is carried down, as a share of its
	// length. Authored.
	clawCurl = .5
	// clawRings is how many rings a claw is drawn in.
	clawRings = 5
)

// boneLoft is a bone between two joints, thicker or thinner along it by
// its rings, as a closed surface.
func boneLoft(from, to maths.Vec, rings []boneRing, hint maths.Vec,
	relief func(along, angle float64) float64) tubeSurface {
	keys := make([]tubeKey, len(rings))
	for i, ring := range rings {
		keys[i] = tubeKey{from.Add(to.Sub(from).Scale(ring.share)),
			ring.halfWidth, ring.halfHeight, 2.2}
	}
	return tubeLoft(keys, limbSpacing, limbSides, hint, relief)
}

// jointBall is a ball at a joint, as a piece.
func jointBall(name string, centre, axis maths.Vec, radius float64,
	look *thing.Material) pterosaurPiece {
	return solidPiece(name, ellipsoidSurface(centre, axis, radius, radius,
		radius, ballRings, ballSides), look)
}

// clawPiece is a curved claw from base along direction, curling toward
// down, length long and radius thick at its root.
func clawPiece(name string, base, direction, down maths.Vec, length,
	radius float64, look *thing.Material) pterosaurPiece {
	keys := make([]tubeKey, clawRings)
	for i := range keys {
		share := float64(i) / float64(clawRings-1)
		curl := clawCurl * share * share * length
		offset := direction.Unit().Scale(share * length).
			Add(down.Unit().Scale(curl))
		thick := radius * math.Pow(1-share, 1.2)
		keys[i] = tubeKey{base.Add(offset), thick, thick, tubeEllipse}
	}
	surface := tubeLoft(keys, limbSpacing/3, limbSides/2, down, nil)
	return solidPiece(name, surface, look)
}

// wingBoneRings are the control rings of the wing's bones: the arm's
// muscle and its crest, the slimmer forearm, the long metacarpal, and
// the finger tapering to a point with a swelling at each of its three
// joints. The finger's joints are drawn on the one rigged bone; the rig
// keeps it a single straight bone.
var (
	humerusRings = []boneRing{{0, .075, .085}, {.18, .085, .115},
		{.5, .065, .07}, {1, .05, .055}}
	ulnaRings = []boneRing{{0, .05, .055}, {.5, .042, .045},
		{1, .035, .035}}
	metacarpalRings = []boneRing{{0, .04, .045}, {.5, .032, .034},
		{1, .03, .03}}
	fingerRings = []boneRing{{0, .03, .03}, {.22, .0225, .0225},
		{.27, .028, .028}, {.32, .021, .021}, {.5, .017, .017},
		{.55, .021, .021}, {.6, .015, .015}, {.75, .011, .011},
		{.79, .014, .014}, {.83, .009, .009}, {1, .004, .004}}
	pteroidRings = []boneRing{{0, .02, .02}, {.5, .015, .015},
		{1, .008, .008}}
)

// armPieces are one wing's bones and joints: the pectoral muscle, the
// shoulder, arm, elbow, forearm, wrist, metacarpal, finger and pteroid.
func (pose pterosaurPose) armPieces(sideIndex int,
	looks pterosaurLooks) []pterosaurPiece {
	joint := pose.arm[sideIndex]
	side := pterosaurSides[sideIndex]
	humerus := joint[1].Sub(joint[0])
	chest := joint[0].Add(maths.V(.05, -.12, -.04*side))
	skin := looks.hide
	return []pterosaurPiece{
		solidPiece("pectoral", ellipsoidSurface(chest, humerus,
			.24, .12, .14, ballRings, ballSides), skin),
		jointBall("shoulder", joint[0], humerus, .125, skin),
		bonePiece("humerus", joint[0], joint[1], humerusRings, skin),
		jointBall("elbow", joint[1], humerus, .07, skin),
		bonePiece("ulna", joint[1], joint[2], ulnaRings, skin),
		jointBall("wrist", joint[2], humerus, .062, skin),
		bonePiece("metacarpal", joint[2], joint[3], metacarpalRings,
			skin),
		jointBall("finger-root", joint[3], humerus, .036, skin),
		bonePiece("wing-finger", joint[3], joint[4], fingerRings, skin),
		bonePiece("pteroid", joint[2], pose.pteroidTip(sideIndex),
			pteroidRings, looks.beak),
	}
}

// bonePiece is a bone between two joints as a piece.
func bonePiece(name string, from, to maths.Vec, rings []boneRing,
	look *thing.Material) pterosaurPiece {
	surface := boneLoft(from, to, rings, maths.V(0, 1, 0), nil)
	return solidPiece(name, surface, look)
}

// rotatedAboutUp is the vector turned by yaw about the vertical axis.
func rotatedAboutUp(vector maths.Vec, yaw float64) maths.Vec {
	cosine, sine := math.Cos(yaw), math.Sin(yaw)
	return maths.V(vector.X*cosine+vector.Z*sine, vector.Y,
		-vector.X*sine+vector.Z*cosine)
}

// digitRings are the control rings of a finger or toe, tapering.
var digitRings = []boneRing{{0, .024, .022}, {.5, .018, .016},
	{1, .013, .012}}

// digitPieces are a set of digits springing from a point: a tube and a
// claw each, with the given lengths and yaws from the forward direction.
// The yaws turn about the vertical, so a positive yaw swings a digit
// toward +z.
func digitPieces(name string, origin, forward maths.Vec, lengths,
	yaws []float64, claw float64, looks pterosaurLooks) []pterosaurPiece {
	var pieces []pterosaurPiece
	down := maths.V(0, -1, 0)
	for i, length := range lengths {
		heading := rotatedAboutUp(forward, yaws[i])
		tip := origin.Add(heading.Unit().Scale(length))
		pieces = append(pieces,
			bonePiece(name, origin, tip, digitRings, looks.hide),
			clawPiece(name+"-claw", tip, heading, down, claw, .013,
				looks.claw))
	}
	return pieces
}

// handPieces are the wing's three free digits and their claws, springing
// from the wrist forward and down.
func (pose pterosaurPose) handPieces(sideIndex int,
	looks pterosaurLooks) []pterosaurPiece {
	wrist := pose.arm[sideIndex][2]
	side := pterosaurSides[sideIndex]
	forward := maths.V(1, -digitDown, 0)
	lengths := []float64{.30, .36, .30}
	yaws := []float64{-side * .35, 0, side * .35}
	return digitPieces("hand-digit", wrist, forward, lengths, yaws,
		handClaw, looks)
}

// legPieces are one hind limb's bones and joints: the thigh, knee, shank,
// ankle and the foot with its four toes and their claws.
func (pose pterosaurPose) legPieces(sideIndex int,
	looks pterosaurLooks) []pterosaurPiece {
	joint := pose.leg[sideIndex]
	thigh := joint[1].Sub(joint[0])
	skin := looks.hide
	pieces := []pterosaurPiece{
		jointBall("hip", joint[0], thigh, .115, skin),
		bonePiece("thigh", joint[0], joint[1], thighRings, skin),
		jointBall("knee", joint[1], thigh, .065, skin),
		bonePiece("shank", joint[1], joint[2], shankRings, skin),
		jointBall("ankle", joint[2], thigh, .05, skin),
		bonePiece("foot", joint[2], joint[3], footRings, skin),
	}
	return append(pieces, pose.toePieces(sideIndex, looks)...)
}

// The hind limb's control rings: the thigh's muscle, the slim shank, and
// the broad flat foot. Authored.
var (
	thighRings = []boneRing{{0, .11, .12}, {.3, .12, .13},
		{.7, .075, .08}, {1, .06, .06}}
	shankRings = []boneRing{{0, .06, .06}, {.4, .048, .05},
		{1, .036, .036}}
	footRings = []boneRing{{0, .05, .035}, {1, .055, .03}}
)

// toePieces are a foot's four toes and claws, fanned from the ball of the
// foot along the foot's own direction, pressed a little down.
func (pose pterosaurPose) toePieces(sideIndex int,
	looks pterosaurLooks) []pterosaurPiece {
	joint := pose.leg[sideIndex]
	forward := joint[3].Sub(joint[2])
	level := maths.V(forward.X, -.12*forward.Len(), forward.Z)
	lengths := []float64{.16, .22, .22, .16}
	yaws := []float64{-.30, -.10, .10, .30}
	return digitPieces("toe", joint[3], level, lengths, yaws, footClaw,
		looks)
}

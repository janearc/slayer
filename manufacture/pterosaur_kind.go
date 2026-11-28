package manufacture

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// pterosaurKind is a pterosaur in the manner of Hatzegopteryx: an
// azhdarchid with a long toothless beak, a heavy neck and long, narrow
// wings. It is an original model made of parts, a skull and a hinged jaw,
// a lofted trunk and neck, bones and joints along the earlier study's
// wing chain, hind limbs, and two panels of membrane per wing that pleat
// and fold with the skeleton. Its proportions are authored hypotheses
// and are marked in pterosaur_proportions.go; it is not an approved
// reconstruction. The design is in doc/bioforms/hatz-next/02-plan.md.
//
// The wingbeat has the three second period of the earlier study. A
// standing kind is on all fours with its wings folded; a gape opens the
// jaw.
type pterosaurKind struct {
	// standing puts the animal on the ground on all fours, wings folded.
	standing bool
	// gape is how far the lower jaw is open, in radians.
	gape float64
	// plain selects an untextured membrane where rest-space paint is unavailable.
	plain bool
	// silver is an invented cool, pale reflected appearance, not emission.
	silver bool
}

// pterosaurPeriod is the wingbeat's period in seconds, as in the earlier
// study.
const pterosaurPeriod = 3.0

// pterosaurHeadGape is how far the head study's lower jaw is open, in
// radians: about twelve degrees. Authored.
const pterosaurHeadGape = .21

// pterosaurGroundHalf is half the side of the ground drawn under a
// standing animal, in metres.
const pterosaurGroundHalf = 40.0

// made is the animal at rest in the glide pose.
func (kind pterosaurKind) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	return kind.build(0)
}

// madeAtTime is the animal at a time in its wingbeat.
func (kind pterosaurKind) madeAtTime(from seed, detail string,
	at time.Duration) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	if detail != "quick" {
		return good{}, fmt.Errorf("pterosaur: unsupported detail %q",
			detail)
	}
	return kind.build(2 * math.Pi * at.Seconds() / pterosaurPeriod)
}

// build is the animal at a wingbeat phase, in radians.
func (kind pterosaurKind) build(phase float64) (good, error) {
	if math.IsNaN(phase) || math.IsInf(phase, 0) {
		return good{}, fmt.Errorf("pterosaur: invalid phase")
	}
	pose := pterosaurFlight(phase, kind.gape)
	if kind.standing {
		pose = pterosaurStanding(kind.gape)
	}
	looks := newPterosaurLooks()
	if kind.silver {
		looks = silverPterosaurLooks()
	}
	if kind.plain {
		looks.membrane.Texture = nil
	}
	pieces := pose.pieces(looks)
	if kind.standing {
		pieces = append(pieces, groundPiece())
	}
	things, err := thingsOf(pieces)
	if err != nil {
		return good{}, err
	}
	return good{native: things}, nil
}

// groundPiece is a flat square of ground under a standing animal.
func groundPiece() pterosaurPiece {
	half := pterosaurGroundHalf
	corners := []maths.Vec{
		maths.V(-half, 0, -half), maths.V(-half, 0, half),
		maths.V(half, 0, half), maths.V(half, 0, -half)}
	light, dark := maths.RGB(.30, .27, .22), maths.RGB(.26, .23, .19)
	look := &thing.Material{Name: "pterosaur-ground", Colour: light,
		Texture: texture.Checker(1.5, light, dark)}
	return pterosaurPiece{name: "ground", points: corners,
		faces: [][3]int{{0, 1, 2}, {0, 2, 3}}, look: look}
}

// pieces is every piece of the animal in a pose: the torso and its
// fibres, the head, and for each side the wing, the hand, the leg and the
// membrane.
func (pose pterosaurPose) pieces(looks pterosaurLooks) []pterosaurPiece {
	torso := pose.torsoSurface()
	pieces := []pterosaurPiece{solidPiece("torso", torso, looks.hide)}
	points, faces := pycnofibres(torso)
	pieces = append(pieces, pterosaurPiece{name: "pycnofibres",
		points: points, faces: faces, look: looks.fibre})
	pieces = append(pieces, pose.headPieces(looks)...)
	balls := pose.bodyBalls()
	for sideIndex := range pterosaurSides {
		pieces = append(pieces, pose.armPieces(sideIndex, looks)...)
		pieces = append(pieces, pose.handPieces(sideIndex, looks)...)
		pieces = append(pieces, pose.legPieces(sideIndex, looks)...)
		pieces = append(pieces,
			pose.membranePieces(sideIndex, looks, balls)...)
	}
	return pieces
}

// membranePieces are one side's two panels of skin and the stiffening
// fibres drawn on each.
func (pose pterosaurPose) membranePieces(sideIndex int, looks pterosaurLooks,
	balls []pterosaurBall) []pterosaurPiece {
	const (
		fibreRidgeWidth  = .012
		fibreRidgeHeight = .003
	)
	var pieces []pterosaurPiece
	panels := []struct {
		name string
		spec membraneSpec
	}{{"membrane", pose.mainPanel(sideIndex)},
		{"propatagium", pose.forwardPanel(sideIndex)}}
	for _, panel := range panels {
		sheet := panel.spec.surface(balls)
		pieces = append(pieces, pterosaurPiece{name: panel.name,
			points: sheet.points, rest: sheet.rest,
			faces: sheet.faces, look: looks.membrane})
		points, faces := sheet.fibreRidges(pterosaurFibreEvery,
			fibreRidgeWidth, fibreRidgeHeight)
		pieces = append(pieces, pterosaurPiece{
			name:   panel.name + "-fibres",
			points: points, faces: faces, look: looks.ridge,
			flat: true})
	}
	return pieces
}

// bodyBalls are spheres standing for the body's volume, which the
// membrane may not pass through: along the trunk and neck, and the thighs
// and shanks.
func (pose pterosaurPose) bodyBalls() []pterosaurBall {
	const ballShare = .95
	var balls []pterosaurBall
	for _, key := range pose.torsoKeys() {
		balls = append(balls, pterosaurBall{key.centre,
			ballShare * math.Min(key.halfWidth, key.halfHeight)})
	}
	for _, leg := range pose.leg {
		balls = append(balls, legBalls(leg)...)
	}
	return balls
}

// legBalls are balls along a hind limb's thigh and shank.
func legBalls(leg [4]maths.Vec) []pterosaurBall {
	const (
		thighRadius = .09
		shankRadius = .06
		perBone     = 3
	)
	radii := []float64{thighRadius, shankRadius}
	var balls []pterosaurBall
	for bone, radius := range radii {
		span := leg[bone+1].Sub(leg[bone])
		for i := 0; i <= perBone; i++ {
			share := float64(i) / perBone
			centre := leg[bone].Add(span.Scale(share))
			balls = append(balls, pterosaurBall{centre, radius})
		}
	}
	return balls
}

package manufacture

import (
	"math"
	"strings"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// A wing folded on a standing bird lies back along the flank, its feathers
// pointing back and a little down, and the trunk is pitched nose up about
// the hips, which lowers everything behind them. For most birds the tips
// stay well above the toes.
//
// A frigatebird's legs are a few centimetres long and its folded wings
// reach most of a metre past the hips, so the pitch would put the hand
// and the tips below its toes, through the ground it stands on. A perched
// frigatebird carries its folded wings up and crossed over its tail,
// the tips well above its feet.
//
// So a standing bird whose folded wing would hang below its toes has the
// whole wing turned up about its shoulder, arm, hand and feathers
// together, until none of it does. A bird whose wing clears its toes is
// not changed, and no bird is lifted off its toes.

// birdRiseMargin is the share more than the least angle a wing is turned
// up by. The angle is worked as if a point's lift were its distance
// behind the shoulder times the angle, which is a little more than it
// is; picked.
const birdRiseMargin = 1.25

// wingRise is the angle, in radians, by which the folded wings in pose
// must turn up about the shoulders so that no point of them hangs lower
// than the toes. It is nought when no point hangs below the toes.
//
// A point hanging hang below the toes and behind the shoulder by behind
// is lifted by turning the wing up by about hang over behind, so the
// angle is the greatest of those over every corner of the wing, times
// birdRiseMargin.
func (body *birdBody) wingRise(pose skeleton.Pose) float64 {
	frames := body.bones.Frames(pose)
	ground := body.toeLow(frames)
	at, _ := body.bones.Find("humerus-right")
	shoulder := frames[at].Apply(body.bones.Bone(at).Head())
	rise := 0.0
	for _, piece := range body.foldedWing() {
		for _, corner := range body.cornersOf(piece, frames) {
			behind := shoulder.X - corner.X
			if hang := ground - corner.Y; hang > 0 && behind > 0 {
				rise = math.Max(rise, hang/behind)
			}
		}
	}
	return rise * birdRiseMargin
}

// foldedWing are the pieces of the wing: the small coverts that ride on
// its bones and every feather that grows on it.
func (body *birdBody) foldedWing() []birdPiece {
	pieces := body.wingPieces()
	for _, feather := range body.feathers {
		if birdOnWing(feather.group) {
			pieces = append(pieces, birdPiece{bone: feather.name,
				form: body.forms[feather.name]})
		}
	}
	return pieces
}

// toeLow is the lowest height of the toes carried by frames.
func (body *birdBody) toeLow(frames []skeleton.Frame) float64 {
	low := math.Inf(1)
	for _, piece := range body.legPieces() {
		if !strings.HasPrefix(piece.bone, "toe") {
			continue
		}
		for _, corner := range body.cornersOf(piece, frames) {
			low = math.Min(low, corner.Y)
		}
	}
	return low
}

// cornersOf are the corners of piece carried by frames.
func (body *birdBody) cornersOf(piece birdPiece,
	frames []skeleton.Frame) []maths.Vec {
	at, _ := body.bones.Find(piece.bone)
	corners, _ := piece.form.Build([]maths.Vec{}, [][3]int{},
		frames[at].Apply)
	return corners
}

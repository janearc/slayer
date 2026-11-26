package pinnipeds

import (
	"fmt"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// walrusRestOnGround turns the bone named about the side axis through its
// joint until the joint at the base of the ray named lies on the ground,
// found by halving the angle between a lift and a drop. It refuses a bone
// or a ray the skeleton does not have, and a swing whose two ends lie on
// the same side of the ground, which has no such angle between them.
func walrusRestOnGround(body *walrusBones, pose skeleton.Pose, bone,
	ray string) (skeleton.Pose, error) {
	const (
		steps       = 30
		clear       = .05
		widest      = .9
		groundLevel = walrusFloor + clear
	)
	at, there := body.tree.Find(ray)
	if _, turned := body.tree.Find(bone); !there || !turned {
		return skeleton.Pose{}, fmt.Errorf("%w: the skeleton lacks %q or %q",
			errWalrusGround, bone, ray)
	}
	base := body.tree.Bone(at)
	height := func(angle float64) float64 {
		tried := pose.Compose(bone, skeleton.NewTurn(walrusSideAxis, angle))
		return body.tree.Frames(tried)[at].Apply(base.Head()).Y - groundLevel
	}
	low, high := -widest, widest
	lowAbove := height(low)
	if highAbove := height(high); (lowAbove > 0) == (highAbove > 0) {
		return skeleton.Pose{}, fmt.Errorf("%w: turning %q leaves %q %.3f "+
			"and %.3f m off it at the ends of its swing", errWalrusGround, bone,
			ray, lowAbove, highAbove)
	}
	for i := 0; i < steps; i++ {
		middle := (low + high) / 2
		above := height(middle)
		if (above > 0) == (lowAbove > 0) {
			low, lowAbove = middle, above
		} else {
			high = middle
		}
	}
	return pose.Compose(bone, skeleton.NewTurn(walrusSideAxis, (low+high)/2)),
		nil
}

// walrusTuskPath is the middle line of a tusk at rest, from its root deep
// in the maxilla at t equal to nought to its worn end at one: the upper
// canine of the skeleton.
func walrusTuskPath(side, t float64) maths.Vec {
	return walrusCanineOf(side).at(t)
}

// walrusCanineOf is the tusk on one side of the skeleton at rest. A
// skeleton with no such tusk is given a fault, which refuses every pose.
func walrusCanineOf(side float64) walrusCanine {
	name := "tusk " + walrusSideName(side)
	canine, ok := walrusRest.forms[name].(walrusCanine)
	if !ok && walrusRest.fault == nil {
		walrusRest.fault = fmt.Errorf("%w: the %s", errWalrusForm, name)
	}
	return canine
}

// walrusPartIndex is the place in the parts of one head part on one side,
// nought for the left and one for the right.
func walrusPartIndex(sideIndex, part int) int {
	return walrusHidePart + 1 + sideIndex*walrusHeadPartCount + part
}

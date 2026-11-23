package manufacture

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// walrusDigitCount is the number of digits on every flipper.
const walrusDigitCount = 5

// walrusRest is the walrus skeleton at rest, laid out once. The hide is
// built over it, and the face's landmarks are placed in its skull's frame.
var walrusRest = walrusSkeleton()

// walrusFrame is the skull's frame at rest.
var walrusFrame = walrusSkullOf(walrusRest)

// errWalrusForm is a skeleton that lacks a bone the hide is built on, or
// has it in a form the hide cannot read.
var errWalrusForm = errors.New("walrus: the skeleton lacks a bone the hide is built on")

// walrusSkullOf is the frame of the skull of the skeleton given. A
// skeleton with no skull is given a fault, which refuses every pose.
func walrusSkullOf(body *walrusBones) walrusSkullFrame {
	skull, ok := body.forms["skull"].(walrusCranium)
	if !ok && body.fault == nil {
		body.fault = fmt.Errorf("%w: the skull", errWalrusForm)
	}
	return skull.frame
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

// walrusMasses is the flesh of the four flippers on their bones and the
// lids of the eyes on the outline given. The body, neck and head are that
// one outline, from the stations in walrus_outline.go, and these are
// joined to it.
func walrusMasses(outline *walrusOutline) ([]walrusMass, error) {
	lids, err := walrusLids(outline)
	if err != nil {
		return nil, err
	}
	return append(walrusLimbMasses(walrusRest), lids...), nil
}

// walrusRayLines is the middle line of every ray of every flipper at
// rest, from the base of its palm or sole bone to the web's rim: four
// flippers, fore then hind on the left, then the right.
var walrusRayLines = walrusFlipperRays(walrusRest)

// walrusFlipperRays is the rays of the four flippers of the skeleton.
func walrusFlipperRays(body *walrusBones) [][walrusDigitCount][]maths.Vec {
	var flippers [][walrusDigitCount][]maths.Vec
	for _, side := range []float64{-1, 1} {
		name := walrusSideName(side)
		flippers = append(flippers,
			walrusRayLinesOf(body, "metacarpal", "fore", name, walrusForeDigits),
			walrusRayLinesOf(body, "metatarsal", "hind", name, walrusHindDigits))
	}
	return flippers
}

// walrusRayLinesOf is the five rays of one flipper as lines of points.
func walrusRayLinesOf(body *walrusBones, palm, kind, side string,
	digits [walrusDigitCount][]float64) [walrusDigitCount][]maths.Vec {
	var rays [walrusDigitCount][]maths.Vec
	for ray := 1; ray <= walrusDigitCount; ray++ {
		for _, name := range walrusRayNames(palm, kind, side, ray, digits) {
			from, to := walrusBoneLine(body, name)
			if len(rays[ray-1]) == 0 {
				rays[ray-1] = append(rays[ray-1], from)
			}
			rays[ray-1] = append(rays[ray-1], to)
		}
		line := rays[ray-1]
		end, before := line[len(line)-1], line[len(line)-2]
		rays[ray-1] = append(line, end.Add(end.Sub(before).Unit().Scale(walrusWebRim)))
	}
	return rays
}

// walrusLineDistance is how far p lies from a line of points.
func walrusLineDistance(p maths.Vec, line []maths.Vec) float64 {
	nearest := math.Inf(1)
	for i := 1; i < len(line); i++ {
		nearest = math.Min(nearest, walrusSegmentDistance(p, line[i-1], line[i]))
	}
	return nearest
}

// walrusLimbShare is how much of a flipper's own skin there is at p: one
// on and between its rays, nought on the body away from them. The hide's
// colour uses it to darken the flippers.
func walrusLimbShare(p maths.Vec) float64 {
	const near, far = .05, .11
	nearest := math.Inf(1)
	for _, flipper := range walrusRayLines {
		for _, ray := range flipper {
			nearest = math.Min(nearest, walrusLineDistance(p, ray))
		}
	}
	return 1 - maths.Smooth(near, far, nearest)
}

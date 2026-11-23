package manufacture

import (
	"errors"
	"math"
	"strings"

	"github.com/janearc/slayer/maths"
)

// sealMass is one mass of the body at rest, held on a bone: an
// ellipsoid whose centre is offset from the bone's head, with half axes
// along the body, up and across, all in metres. A mirrored mass is on
// both sides, written for the right.
type sealMass struct {
	bone         string
	offset, half maths.Vec
	mirrored     bool
}

// sealMasses are the masses the trunk, neck and head are made of, from
// the tail forward. Each is placed on the bone it lies against; offsets
// and half axes are chosen, as no published girths were reached.
var sealMasses = []sealMass{
	// The root of the tail between the hind flippers.
	{"caudal3", maths.V(-.123, -.054, 0), maths.V(.08, .05, .045), false},
	// The hip and the muscles of the pelvic stroke.
	{"sacral1", maths.V(-.11, -.074, .04), maths.V(.18, .10, .08), true},
	// The gut and the belly wall under the lumbar column.
	{"lumbar2", maths.V(-.03, -.099, 0), maths.V(.30, .17, .19), false},
	// The long back muscles on each side of the column.
	{"thoracic10", maths.V(-.17, -.024, .05), maths.V(.55, .07, .06), true},
	// The heart, the lungs and the rib cage with its muscle.
	{"thoracic8", maths.V(-.08, -.109, 0), maths.V(.36, .18, .20), false},
	// The shoulder: the muscles over the scapula and the arm.
	{"fore-right-scapula", maths.V(-.03, .04, -.05), maths.V(.12, .14, .10),
		true},
	// The neck's muscles round the cervical column.
	{"neck3", maths.V(.007, -.04, 0), maths.V(.16, .11, .11), false},
	// The throat and the floor of the mouth.
	{"skull", maths.V(.02, -.101, 0), maths.V(.10, .06, .07), false},
	// The braincase and its jaw muscles.
	{"skull", maths.V(.06, -.016, 0), maths.V(.10, .085, .09), false},
	// The muzzle over the rostrum, with the whisker pads.
	{"skull", maths.V(.17, -.051, 0), maths.V(.06, .045, .055), false},
}

// Blubber over the masses, in metres: thick on the trunk, thin on the
// head and gone at the nose's tip. Chosen, under the 25 mm a search
// summary gave for one resting seal; that page was not opened.
const (
	sealTrunkBlubber = .025
	sealHeadBlubber  = .012
	sealBlubberFall  = .55 // the X where the head's thin blubber begins
	sealBlubberBlend = .08 // how far along X the two blend
	sealNoseStart    = .80 // the X where the blubber thins to the tip
	sealNoseTip      = .835
)

// sealSoftness is how far, in metres, the join of two masses is rounded
// rather than creased: the width of their soft maximum. Chosen.
const sealSoftness = .012

// sealLeast is the smallest reach of the skin from its centre, in
// metres, where no mass lies, as at the tip of the tail. Chosen.
const sealLeast = .004

var errSealMassBone = errors.New("harbor seal: a mass's bone is not here")

// sealBody is the masses placed in the rest space of the skeleton.
type sealBody struct{ centres, halves []maths.Vec }

// newSealBody places every mass on its bone, a mirrored one on both
// sides.
func newSealBody(bones *sealBones) (sealBody, error) {
	var body sealBody
	for _, mass := range sealMasses {
		sides := []float64{1}
		if mass.mirrored {
			sides = []float64{-1, 1}
		}
		for _, side := range sides {
			head, _, found := bones.ends(sealSideBone(mass.bone, side))
			if !found {
				return sealBody{}, errSealMassBone
			}
			body.centres = append(body.centres,
				head.Add(sealMirror(mass.offset, side)))
			body.halves = append(body.halves, mass.half)
		}
	}
	return body, nil
}

// sealSideBone is a right-side bone's name for the side; a name with no
// side is itself.
func sealSideBone(name string, side float64) string {
	return strings.Replace(name, "-right-", "-"+sealSideName(side)+"-", 1)
}

// reach is how far the body's surface lies from centre along way, a
// unit direction: the soft maximum of where way leaves each mass it
// passes through, and the blubber over it.
func (body sealBody) reach(centre, way maths.Vec) float64 {
	total, found := 0.0, false
	for index, middle := range body.centres {
		out, through := sealExit(centre, way, middle, body.halves[index])
		if through {
			total += math.Exp(out / sealSoftness)
			found = true
		}
	}
	reach := sealLeast
	if found {
		reach = math.Max(reach, sealSoftness*math.Log(total))
	}
	return reach + sealBlubber(centre.X)
}

// sealBlubber is the blubber's depth at x.
func sealBlubber(x float64) float64 {
	head := maths.Smooth(sealBlubberFall-sealBlubberBlend,
		sealBlubberFall+sealBlubberBlend, x)
	depth := sealTrunkBlubber + (sealHeadBlubber-sealTrunkBlubber)*head
	return depth * (1 - maths.Smooth(sealNoseStart, sealNoseTip, x))
}

// sealExit is how far along way from start a ray leaves the ellipsoid
// of centre middle and half axes half, and whether it leaves it ahead
// of start at all.
func sealExit(start, way, middle, half maths.Vec) (float64, bool) {
	scale := func(point maths.Vec) maths.Vec {
		return maths.V(point.X/half.X, point.Y/half.Y, point.Z/half.Z)
	}
	from, toward := scale(start.Sub(middle)), scale(way)
	a := toward.Dot(toward)
	b := 2 * from.Dot(toward)
	c := from.Dot(from) - 1
	discriminant := b*b - 4*a*c
	if discriminant < 0 {
		return 0, false
	}
	far := (-b + math.Sqrt(discriminant)) / (2 * a)
	return far, far > 0
}

// shape is each rest ring moved out or in along its own spokes to the
// body's surface. The rings are changed in place and returned.
func (body sealBody) shape(rings [][]maths.Vec) [][]maths.Vec {
	for _, ring := range rings {
		centre := alfaMiddle(ring)
		for index, point := range ring {
			way := point.Sub(centre).Unit()
			ring[index] = centre.Add(way.Scale(body.reach(centre, way)))
		}
	}
	return rings
}

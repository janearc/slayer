package manufacture

import (
	"errors"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
)

// beamFalloff is how far down the beam's Gaussian profile its box reaches:
// the light at the box's edge is a thousandth of the light at the centre.
const beamFalloff = 1000.0 // picked

// beamEdge is how many widths out from the axis the profile has fallen to
// a thousandth, from its intensity of exp(-2 r^2 / w^2).
var beamEdge = math.Sqrt(math.Log(beamFalloff) / 2)

// errLaser is the refusal of a beam with no reach, no waist or no finite
// spread, whose box cannot be said.
var errLaser = errors.New("manufacture: a laser needs a reach and a " +
	"waist above nothing, and a finite spread")

// Laser is a launch: a beam from a place, toward a direction, with its
// waist, reach and spread. It is an object of the manufactory that makes no
// mesh: a description of a box of light. The manufactory never calls emit
// (decided 2026-10-06; a test holds it), so what the beam carries and the
// lamp it becomes are for whoever stages it, with emit.NewBeam.
type Laser struct {
	// At is where the beam starts and Toward which way it goes.
	At, Toward maths.Vec
	// Radius is the beam's waist, Reach how far it goes and Divergence its
	// half-angle.
	Radius, Reach measure.Length
	Divergence    measure.Angle
}

// extent is the box of the beam out to where its profile has fallen to a
// thousandth: its axis for its whole reach, widened by that many widths of
// the beam at its far end, which is the widest it is.
func (laser Laser) extent(from seed) (maths.Box, error) {
	return laser.Extent()
}

// Extent is the beam's box, in the frame its place is given in.
func (laser Laser) Extent() (maths.Box, error) {
	reach := laser.Reach.InMetres()
	width := laser.Radius.InMetres() +
		reach*math.Tan(laser.Divergence.InRadians())
	if !(reach > 0 && laser.Radius.InMetres() > 0 && width > 0) ||
		math.IsInf(width, 0) {
		return maths.Box{}, errLaser
	}
	end := laser.At.Add(laser.Toward.Unit().Scale(reach))
	return maths.SegmentBox(laser.At, end, beamEdge*width), nil
}

// seen is how an eye sees the beam's box.
func (laser Laser) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenObject(laser, at, by)
}

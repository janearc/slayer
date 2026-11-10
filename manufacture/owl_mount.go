package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// base is a picked observing platform, azimuth rail and altitude cradle.
// It suggests the open modular concept, without modelling its friction drives.
func (owlObservatory) base(assembly *instrumentAssembly,
	pose maths.Subspace,
) {
	height := owlValue("platform-height")
	assembly.add(groundDrum(owlValue("platform-radius"), spaceZero, height),
		maths.Local(), body)
	rail := groundDrum(owlValue("azimuth-rail-radius"),
		owlValue("azimuth-rail-radius")-owlValue("rail-width"),
		owlValue("rail-height"))
	at := maths.NewSubspace(maths.YAxis.Scale(height),
		maths.Identity, spaceOne)
	assembly.add(rail, at, spaceStructure)
	for _, sign := range []float64{-spaceOne, spaceOne} {
		owlBearing(assembly, sign, pose)
	}
}

// owlBearing closes a wheel, hub, base and two raked support members.
func owlBearing(assembly *instrumentAssembly, sign float64,
	pose maths.Subspace,
) {
	radius := owlValue("altitude-wheel-radius")
	thick := owlValue("altitude-wheel-thickness")
	x := sign * owlValue("altitude-wheel-offset")
	centre := maths.V(x, pose.Origin.Y, spaceZero)
	orientation := maths.Rotation(maths.ZAxis, -math.Pi*spaceHalf)
	seat := centre.Sub(maths.XAxis.Scale(thick * spaceHalf))
	wheel := maths.NewSubspace(seat, orientation, spaceOne)
	assembly.add(groundDrum(radius, radius-owlValue("wheel-rim"),
		thick), wheel, spaceStructure)
	// The primary-plane frame joint is the axle's attachment, not an
	// independently authored reach that can stop short of the mirror cell.
	other := maths.V(sign*owlValue("primary-frame-radius"), pose.Origin.Y, spaceZero)
	assembly.rod(centre, other, owlValue("bearing-radius"), spaceStructure)
	width := owlValue("bearing-base-width")
	length := owlValue("bearing-base-length")
	corner := maths.V(x-width*spaceHalf, owlValue("platform-height"),
		-length*spaceHalf)
	assembly.box(width, owlValue("bearing-base-height"), length,
		corner, spaceStructure)
	owlBearingLegs(assembly, centre)
}

// owlBearingLegs leave the centre of the mirror cell open.
func owlBearingLegs(assembly *instrumentAssembly, centre maths.Vec) {
	for _, sign := range []float64{-spaceOne, spaceOne} {
		foot := maths.V(centre.X, owlValue("bearing-foot-height"),
			sign*owlValue("bearing-foot-reach"))
		assembly.rod(foot, centre, owlValue("bearing-leg-radius"), spaceStructure)
	}
}

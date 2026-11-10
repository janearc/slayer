package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// forkMount supports the primary on an altitude bearing above a pedestal.
func (row groundObservatory) forkMount(assembly *instrumentAssembly) {
	primary := row.opticalFrame()
	aperture := row.aperture
	pier := primary.Origin.Y * groundValue("pier-height")
	width := aperture * groundValue("pier-width")
	corner := maths.V(-width*spaceHalf, spaceZero, -width*spaceHalf)
	assembly.box(width, pier, width, corner, spaceStructure)
	back := -aperture * groundValue("rim-thickness")
	bearing := primary.Out(maths.YAxis.Scale(back))
	offset := aperture * groundValue("fork-offset")
	arm := aperture * groundValue("fork-width")
	depth := aperture * groundValue("fork-depth")
	span := offset/spaceHalf + arm
	fork := forkFrame{width: span, height: bearing.Y - pier + arm,
		arm: arm, depth: depth}
	at := maths.NewSubspace(maths.V(spaceZero, pier-arm, bearing.Z),
		maths.Identity, spaceOne)
	assembly.add(fork, at, spaceStructure)
	left := bearing.Add(maths.XAxis.Scale(-offset))
	right := bearing.Add(maths.XAxis.Scale(offset))
	assembly.rod(left, right,
		aperture*groundValue("bearing-radius"), spaceStructure)
}

// polarMount suggests the Hale's tilted polar axis and horseshoe support.
// Its latitude and member sizes are visual choices, not surveyed geometry.
func (row groundObservatory) polarMount(assembly *instrumentAssembly) {
	aperture := row.aperture
	altitude := groundValue("polar-altitude-deg") * maths.Degree
	angle := math.Pi*spaceHalf - altitude
	primary := row.opticalFrame()
	back := -aperture * groundValue("horseshoe-height")
	at := primary.Out(maths.YAxis.Scale(back))
	polar := tiltedFrame(at, angle)
	outer := aperture * groundValue("horseshoe-outer")
	assembly.add(slitCylinder{outer: outer,
		inner:   aperture * groundValue("horseshoe-inner"),
		height:  aperture * groundValue("horseshoe-height"),
		opening: groundValue("horseshoe-opening-deg") * maths.Degree},
		polar, spaceStructure)
	radius := aperture * groundValue("polar-support")
	footprint := row.diameter * spaceHalf * groundValue("polar-footprint")
	for _, sign := range []float64{-spaceOne, spaceOne} {
		foot := maths.V(sign*outer, radius, footprint)
		diameter := radius / spaceHalf
		corner := maths.V(foot.X-radius, spaceZero, foot.Z-radius)
		assembly.box(diameter, radius, diameter, corner, spaceStructure)
		joint := polar.Out(maths.XAxis.Scale(sign * outer))
		assembly.rod(foot, joint, radius, spaceStructure)
	}
	end := at.Add(polar.Orientation.Y.Scale(outer))
	assembly.rod(at, end,
		aperture*groundValue("bearing-radius"), spaceStructure)
}

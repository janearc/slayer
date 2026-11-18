package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// keckExterior places substantial housings along the parallel shutter edges.
// The housings follow the spherical front; they are not radial dome seams.
func (row groundObservatory) keckExterior(assembly *instrumentAssembly) {
	assembly.object.dressed[spaceFoil] = groundPaint("observatory enclosure", "keck-trim-reflectance")
	defer assembly.detailSince(len(assembly.object.parts))
	radius := row.diameter * spaceHalf
	centre := row.height - radius
	x := groundValue("keck-slit-width")*spaceHalf + groundValue("keck-housing-offset")
	start := groundValue("keck-plinth-height")
	stop := groundValue("keck-housing-top")
	count := int(groundValue("keck-track-steps"))
	depth := groundValue("keck-housing-radius")
	for _, sign := range []float64{-spaceOne, spaceOne} {
		point := func(index int) maths.Vec {
			y := start + (stop-start)*float64(index)/float64(count)
			z := -math.Sqrt(radius*radius - x*x - (y-centre)*(y-centre))
			return maths.V(sign*x, y, z)
		}
		for index := range count {
			from, to := point(index), point(index+1)
			gap := to.Sub(from).Unit().Scale(groundValue("rib-joint-metres"))
			assembly.rod(from.Add(gap), to.Sub(gap), depth, spaceFoil)
		}
	}
	// Horizontal battens make the broad lower shutter legible.
	for index := range int(groundValue("keck-shutter-battens")) {
		y := start + groundValue("keck-batten-step")*float64(index+1)
		z := -math.Sqrt(radius*radius - (y-centre)*(y-centre))
		assembly.box(groundValue("keck-slit-width")-2*depth,
			groundValue("keck-batten-height"), groundValue("keck-batten-depth"),
			maths.V(-groundValue("keck-slit-width")*spaceHalf+depth, y, z),
			spaceFoil)
	}
}

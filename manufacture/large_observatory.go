package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/research"
)

// These relay roles are defined after the existing survey instrument slots.
const (
	largeTertiary slot = surveyCamera + spaceOne
	largeFourth        = largeTertiary + spaceOne
	largeFifth         = largeFourth + spaceOne
)

type largeOptics struct {
	hole              int
	length, secondary float64
}

// largeGround joins source aperture and inventory to authored scenery.
func largeGround() groundObservatory {
	optics := &largeOptics{
		hole:      int(groundValue("elt-primary-hole-rings")),
		length:    groundValue("elt-truss-length"),
		secondary: groundValue("elt-secondary-diameter")}
	return groundObservatory{
		diameter: groundValue("elt-enclosure-diameter"),
		height:   groundValue("elt-enclosure-height"),
		aperture: research.Value("elt-primary-metres"),
		segments: int(research.Value("elt-primary-segments")),
		large:    optics}
}

// assemble suggests ESO's five-mirror design with a source-sized primary.
// Relay figures, sizes and positions are picked scenery, not prescriptions.
func (large largeOptics) assemble(row groundObservatory,
	assembly *instrumentAssembly, pose maths.Subspace,
) {
	length := row.aperture * large.length
	rim := row.aperture * groundValue("truss-rim")
	row.truss(assembly, pose, rim, length)
	primary := annularPrimary{count: row.segments, hole: large.hole,
		diameter: row.aperture, fill: groundValue("tile-fill"),
		depth:     row.aperture * groundValue("mirror-sag"),
		thickness: row.aperture * groundValue("mirror-thickness")}
	assembly.add(primary, pose, spaceMirror)
	radius := row.aperture * large.secondary * spaceHalf
	front := tiltedFrame(maths.YAxis.Scale(length), math.Pi).Within(pose)
	assembly.add(mirrorDish{radius: radius,
		depth: row.aperture * groundValue("mirror-sag") * spaceHalf},
		front, spaceSecondary)
	row.secondaryRods(assembly, pose, rim, length, radius)
	largeRelays(row.aperture, length, assembly, pose)
}

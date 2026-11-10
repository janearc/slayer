package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// owlRelay records picked poses and sizes for the four corrector surfaces.
type owlRelay struct {
	name                           string
	slot                           slot
	diameter, height, offset, tilt float64
	curved                         bool
}

// corrector keeps the six reflective roles distinct, without tracing their
// train.
func (owlObservatory) corrector(assembly *instrumentAssembly,
	pose maths.Subspace,
) {
	mirrors := []owlRelay{
		{name: "tertiary mirror", slot: largeTertiary, curved: true,
			diameter: owlValue("third-diameter"), height: owlValue("third-height")},
		{name: "fourth mirror", slot: largeFourth, curved: true,
			diameter: owlValue("fourth-diameter"), height: owlValue("fourth-height"),
			tilt: math.Pi},
		{name: "fifth mirror", slot: largeFifth,
			diameter: owlValue("fifth-diameter"), height: owlValue("fifth-height"),
			offset: owlValue("fifth-offset"), tilt: owlValue("fold-deg") * maths.Degree},
		{name: "sixth mirror", slot: owlSixth,
			diameter: owlValue("sixth-diameter"), height: owlValue("sixth-height"),
			offset: owlValue("sixth-offset"), tilt: -owlValue("fold-deg") * maths.Degree},
	}
	for _, mirror := range mirrors {
		mirror.add(assembly, pose)
	}
}

// owlSixth follows the five existing large-instrument mirror slots.
const owlSixth slot = largeFifth + spaceOne

// add connects the mirror's backing to an actual upper-frame joint.
func (mirror owlRelay) add(assembly *instrumentAssembly, pose maths.Subspace) {
	assembly.object.dressed[mirror.slot] = groundMirror(mirror.name)
	radius := mirror.diameter * spaceHalf
	thick := owlValue("corrector-thickness")
	var surface form = groundDrum(radius, spaceZero, thick)
	seat := thick * spaceHalf
	if mirror.curved {
		depth := owlValue("corrector-sag")
		dish := mirrorDish{radius: radius, depth: depth}
		surface, seat = dish, dish.backPlane()
	}
	centre := maths.V(mirror.offset, mirror.height, spaceZero)
	orientation := maths.Rotation(maths.ZAxis, mirror.tilt)
	local := maths.NewSubspace(centre, orientation, spaceOne)
	at := local.Within(pose)
	assembly.add(surface, at, mirror.slot)
	joint := at.Out(maths.YAxis.Scale(seat))
	foot := pose.Out(owlUpperFrame().nearestJoint(local.Out(maths.YAxis.Scale(seat))))
	assembly.rod(foot, joint, owlValue("corrector-support-radius"), spaceStructure)
}

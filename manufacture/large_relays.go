package manufacture

import "github.com/janearc/slayer/maths"

type largeRelay struct {
	name                           string
	material                       slot
	diameter, height, offset, tilt float64
	curved                         bool
}

// largeRelays adds the tertiary and two folded mirrors in the optical frame.
func largeRelays(aperture, length float64,
	assembly *instrumentAssembly, pose maths.Subspace,
) {
	height := length * groundValue("elt-fold-height")
	tilt := groundValue("elt-fold-tilt") * maths.Degree
	third := largeRelay{
		name: "tertiary mirror", material: largeTertiary,
		diameter: aperture * groundValue("elt-tertiary-diameter"),
		height:   aperture * groundValue("elt-tertiary-height"),
		curved:   true}
	fourth := largeRelay{name: "fourth mirror", material: largeFourth,
		diameter: aperture * groundValue("elt-fourth-diameter"),
		height:   height, tilt: tilt}
	fifth := largeRelay{name: "fifth mirror", material: largeFifth,
		diameter: aperture * groundValue("elt-fifth-diameter"),
		height:   height, tilt: tilt,
		offset: aperture * groundValue("elt-fifth-offset")}
	for _, relay := range []largeRelay{third, fourth, fifth} {
		relay.place(aperture, assembly, pose)
	}
}

// place gives each mirror its own role and seats its supporting rod.
func (relay largeRelay) place(aperture float64,
	assembly *instrumentAssembly, pose maths.Subspace,
) {
	assembly.object.dressed[relay.material] = groundMirror(relay.name)
	thick := aperture * groundValue("mirror-thickness")
	seat := thick * groundValue("elt-flat-support-seat")
	var mirror form = groundDrum(relay.diameter*spaceHalf, spaceZero, thick)
	if relay.curved {
		depth := aperture * groundValue("elt-relay-sag")
		mirror = mirrorDish{radius: relay.diameter * spaceHalf,
			depth: depth}
		seat = -depth * groundValue("elt-curved-support-seat")
	}
	centre := maths.V(relay.offset, relay.height, spaceZero)
	orientation := maths.Rotation(maths.ZAxis, relay.tilt)
	local := maths.NewSubspace(centre, orientation, spaceOne)
	at := local.Within(pose)
	assembly.add(mirror, at, relay.material)
	foot := maths.V(relay.offset+relay.diameter, spaceZero, spaceZero)
	joint := at.Out(maths.YAxis.Scale(seat))
	assembly.rod(pose.Out(foot), joint,
		aperture*groundValue("elt-relay-support"), spaceStructure)
}

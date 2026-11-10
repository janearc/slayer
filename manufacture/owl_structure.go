package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// owlFrame is one tapered modular space frame in the optical coordinate system.
type owlFrame struct {
	start, end, near, far float64
	bays, members         int
	radius                float64
	sharedTop             bool
}

// structure gives OWL a tapering upper frame and a deep triangulated mirror
// cell.
func (owlObservatory) structure(assembly *instrumentAssembly,
	pose maths.Subspace,
) {
	upper := owlUpperFrame()
	lower := owlFrame{start: -owlValue("cell-depth"),
		near: owlValue("cell-bottom-radius"), far: owlValue("primary-frame-radius"),
		bays: int(owlValue("lower-bays")), members: upper.members,
		radius: owlValue("frame-rod-radius"), sharedTop: true}
	upper.add(assembly, pose)
	lower.add(assembly, pose)
}

// owlUpperFrame is shared by the structure and its corrector attachments.
func owlUpperFrame() owlFrame {
	return owlFrame{end: owlValue("structure-length"),
		near: owlValue("primary-frame-radius"), far: owlValue("secondary-frame-radius"),
		bays: int(owlValue("upper-bays")), members: int(owlValue("azimuth-modules")),
		radius: owlValue("frame-rod-radius")}
}

// nearestJoint attaches equipment to a built station, never an empty interior.
func (frame owlFrame) nearestJoint(point maths.Vec) maths.Vec {
	nearest := frame.point(spaceZero, spaceZero)
	distance := nearest.Sub(point).Len()
	for bay := range frame.bays + spaceOne {
		for member := range frame.members {
			station := frame.point(bay, member)
			if next := station.Sub(point).Len(); next < distance {
				nearest, distance = station, next
			}
		}
	}
	return nearest
}

// point is a station corner on one of the picked twelve radial modules.
func (frame owlFrame) point(bay, member int) maths.Vec {
	share := float64(bay) / float64(frame.bays)
	height := frame.start + (frame.end-frame.start)*share
	radius := frame.near + (frame.far-frame.near)*share
	angle := fullTurn * float64(member%frame.members) / float64(frame.members)
	sine, cosine := math.Sincos(angle)
	return maths.V(radius*cosine, height, -radius*sine)
}

// add keeps each module's geometry explicit, without recursive mesh expansion.
func (frame owlFrame) add(assembly *instrumentAssembly,
	pose maths.Subspace,
) {
	for bay := range frame.bays + spaceOne {
		if frame.sharedTop && bay == frame.bays {
			continue
		}
		for member := range frame.members {
			here := frame.point(bay, member)
			next := frame.point(bay, member+spaceOne)
			frame.joint(assembly, pose, here)
			frame.member(assembly, pose, here, next)
			if bay == frame.bays {
				continue
			}
			top := frame.point(bay+spaceOne, member)
			across := frame.point(bay+spaceOne, member+spaceOne)
			frame.member(assembly, pose, here, top)
			frame.member(assembly, pose, here, across)
			frame.member(assembly, pose, next, top)
		}
	}
}

// member insets caps into the joint blocks instead of sharing their end plane.
func (frame owlFrame) member(assembly *instrumentAssembly,
	pose maths.Subspace, from, to maths.Vec,
) {
	direction := to.Sub(from).Unit()
	inset := frame.radius * owlValue("joint-inset")
	start := from.Add(direction.Scale(inset))
	end := to.Sub(direction.Scale(inset))
	assembly.rod(pose.Out(start), pose.Out(end), frame.radius, spaceStructure)
}

// joint supplies a closed block around a module's meeting points.
func (frame owlFrame) joint(assembly *instrumentAssembly,
	pose maths.Subspace, centre maths.Vec,
) {
	side := frame.radius * owlValue("joint-width")
	half := side * spaceHalf
	corner := centre.Sub(maths.V(half, half, half))
	localBox(assembly, pose, side, side, side, corner, spaceStructure)
}

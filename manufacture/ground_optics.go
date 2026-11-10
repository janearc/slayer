package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// optics builds a long closed tube or an open truss around the primary.
func (row groundObservatory) optics(assembly *instrumentAssembly,
	pose maths.Subspace,
) {
	if row.large != nil {
		row.large.assemble(row, assembly, pose)
		return
	}
	if row.survey != nil {
		row.survey.assemble(row, assembly, pose)
		return
	}
	aperture := row.aperture
	length := aperture * groundValue("open-length")
	rim := aperture * groundValue("truss-rim")
	if row.equatorial {
		length = aperture * groundValue("palomar-length")
		rim = aperture * groundValue("tube-radius")
		wall := aperture * groundValue("tube-wall")
		assembly.add(groundDrum(rim, rim-wall, length), pose, leaf)
	} else {
		row.truss(assembly, pose, rim, length)
	}
	var primary form = mirrorDish{radius: aperture * spaceHalf,
		depth: aperture * groundValue("mirror-sag")}
	if row.segments != spaceZero {
		primary = hexagonalPrimary{rings: hexRings(row.segments),
			diameter: aperture, fill: groundValue("tile-fill"),
			depth:     aperture * groundValue("mirror-sag"),
			thickness: aperture * groundValue("mirror-thickness")}
	}
	assembly.add(primary, pose, spaceMirror)
	row.secondaryOn(assembly, pose, rim, length)
}

// truss joins its two closed rims with longitudinal members.
func (row groundObservatory) truss(assembly *instrumentAssembly,
	pose maths.Subspace, radius, length float64,
) {
	thickness := row.aperture * groundValue("rim-thickness")
	for _, position := range []float64{spaceZero, length} {
		at := maths.NewSubspace(maths.YAxis.Scale(position),
			maths.Identity, spaceOne).Within(pose)
		assembly.add(groundDrum(radius, radius-thickness, thickness),
			at, spaceStructure)
	}
	members := int(groundValue("truss-members"))
	centre := radius - thickness*spaceHalf
	for index := range members {
		angle := fullTurn * float64(index) / float64(members)
		x, z := math.Cos(angle)*centre, -math.Sin(angle)*centre
		// End caps sit inside the rims, clear of radial support joints.
		back := pose.Out(maths.V(x, thickness*spaceHalf, z))
		front := pose.Out(maths.V(x, length+thickness*spaceHalf, z))
		radius := row.aperture * groundValue("truss-radius")
		assembly.rod(back, front, radius, spaceStructure)
	}
}

// secondaryOn keeps three rim attachments separate, without shared rod caps.
func (row groundObservatory) secondaryOn(assembly *instrumentAssembly,
	pose maths.Subspace, rim, length float64,
) {
	radius := row.aperture * groundValue("secondary-share") * spaceHalf
	var mirror form = mirrorDish{radius: radius,
		depth: row.aperture * groundValue("mirror-sag") * spaceHalf}
	if row.survey != nil {
		radius = row.survey.secondary * spaceHalf
		camera := row.aperture * groundValue("survey-camera-radius")
		clearance := row.aperture *
			groundValue("survey-camera-clearance")
		depth := row.aperture * groundValue("mirror-sag") * spaceHalf
		mirror = surveyMirror{outer: radius, inner: camera + clearance,
			depth:   depth,
			backing: row.aperture * groundValue("mirror-thickness")}
	}
	flipped := tiltedFrame(maths.YAxis.Scale(length), math.Pi)
	secondary := flipped.Within(pose)
	assembly.add(mirror, secondary, spaceSecondary)
	row.secondaryRods(assembly, pose, rim, length, radius)
}

// secondaryRods shares the three-point support between telescope families.
func (row groundObservatory) secondaryRods(assembly *instrumentAssembly,
	pose maths.Subspace, rim, length, radius float64,
) {
	for corner := range triangleCorners {
		angle := fullTurn * float64(corner) / triangleCorners
		x, z := math.Cos(angle), -math.Sin(angle)
		direction := maths.V(x, spaceZero, z)
		front := maths.YAxis.Scale(length)
		outer := front.Add(direction.Scale(rim))
		inner := front.Add(direction.Scale(radius))
		girth := row.aperture * groundValue("truss-radius")
		assembly.rod(pose.Out(outer), pose.Out(inner),
			girth, spaceStructure)
	}
}

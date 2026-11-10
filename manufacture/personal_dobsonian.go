package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// dobsonian carries an open Newtonian on a rocker and large altitude bearings.
func (row personalTelescope) dobsonian(assembly *instrumentAssembly,
	aperture float64, pose maths.Subspace,
) {
	base := aperture * row.value("base-height")
	board := groundDrum(aperture*row.value("base-radius"), spaceZero, base)
	assembly.add(board, maths.Local(), body)
	rocker := forkFrame{width: aperture * row.value("rocker-width"),
		height: aperture * row.value("rocker-height"),
		arm:    aperture * personalValue("mount-arm"),
		depth:  aperture * row.value("rocker-depth")}
	at := maths.NewSubspace(maths.YAxis.Scale(base),
		maths.Identity, spaceOne)
	assembly.add(rocker, at, body)
	row.altitudeBearings(assembly, aperture, pose.Origin)
	row.mirrorBox(assembly, aperture, pose)
	length := aperture * row.value("length")
	rim := aperture * personalValue("tube-radius")
	personalCage(assembly, aperture, length, pose)
	assembly.add(mirrorDish{radius: aperture * spaceHalf,
		depth: aperture * groundValue("mirror-sag")}, pose, spaceMirror)
	personalDiagonal(assembly, aperture, length, rim, pose)
	row.sideFocuser(assembly, aperture, length, pose)
}

// altitudeBearings put two circular bearing faces beside the mirror box.
func (row personalTelescope) altitudeBearings(assembly *instrumentAssembly,
	aperture float64, centre maths.Vec,
) {
	radius := aperture * row.value("bearing-radius")
	thick := aperture * personalValue("bearing-thickness")
	orientation := maths.Rotation(maths.ZAxis, -math.Pi*spaceHalf)
	for _, sign := range []float64{-spaceOne, spaceOne} {
		offset := sign * aperture * row.value("bearing-offset")
		at := centre.Add(maths.XAxis.Scale(offset))
		space := maths.NewSubspace(at, orientation, spaceOne)
		assembly.add(groundDrum(radius, spaceZero, thick), space, body)
	}
}

// mirrorBox uses separately closed panels with small authored joint clearances.
func (row personalTelescope) mirrorBox(assembly *instrumentAssembly,
	aperture float64, pose maths.Subspace,
) {
	width := aperture * row.value("box-width")
	height := aperture * row.value("box-height")
	wall := aperture * personalValue("box-wall")
	gap := aperture * personalValue("box-joint")
	half := width * spaceHalf
	corner := maths.V(-half, -wall-gap, -half)
	localBox(assembly, pose, width, wall, width, corner, body)
	for _, sign := range []float64{-spaceOne, spaceOne} {
		x := sign*half - (sign+spaceOne)*wall*spaceHalf
		localBox(assembly, pose, wall, height, width,
			maths.V(x, spaceZero, -half), body)
		z := sign*half - (sign+spaceOne)*wall*spaceHalf
		corner := maths.V(-half+wall+gap, spaceZero, z)
		span := width - (wall+gap)/spaceHalf
		localBox(assembly, pose, span, height, wall, corner, body)
	}
}

// sideFocuser suggests a Newtonian eyepiece beside the upper cage.
func (row personalTelescope) sideFocuser(assembly *instrumentAssembly,
	aperture, length float64, pose maths.Subspace,
) {
	rim := aperture * personalValue("tube-radius")
	from := pose.Out(maths.V(-rim, length, spaceZero))
	reach := aperture * personalValue("focuser-length")
	to := pose.Out(maths.V(-rim-reach,
		length, spaceZero))
	assembly.rod(from, to, aperture*personalValue("focuser-radius"),
		spaceStructure)
}

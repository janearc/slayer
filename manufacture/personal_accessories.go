package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// personalTubeRings close around the refractor without coincident wall faces.
func personalTubeRings(assembly *instrumentAssembly, pose maths.Subspace,
	aperture float64,
) {
	for _, name := range []string{"ring-lower", "ring-upper"} {
		height := aperture * personalValue(name)
		at := maths.NewSubspace(maths.YAxis.Scale(height),
			maths.Identity, spaceOne).Within(pose)
		ring := groundDrum(aperture*personalValue("ring-outer"),
			aperture*personalValue("ring-inner"),
			aperture*personalValue("ring-depth"))
		assembly.add(ring, at, spaceStructure)
	}
}

// personalFinder adds a small parallel tube on a short bracket.
func personalFinder(assembly *instrumentAssembly, pose maths.Subspace,
	aperture, length float64,
) {
	along := length * personalValue("finder-seat")
	offset := aperture * personalValue("finder-offset")
	start := pose.Out(maths.V(offset, along, spaceZero))
	reach := aperture * personalValue("finder-length")
	end := start.Add(pose.Orientation.Y.Scale(reach))
	assembly.rod(start, end, aperture*personalValue("finder-radius"),
		spaceStructure)
	radius := aperture * personalValue("tube-radius")
	foot := pose.Out(maths.V(radius, along, spaceZero))
	assembly.rod(foot, start, aperture*personalValue("finder-bracket"),
		spaceStructure)
}

// personalCage uses paired diagonal members rather than parallel cage posts.
func personalCage(assembly *instrumentAssembly, aperture, length float64,
	pose maths.Subspace,
) {
	lower := aperture * personalValue("cage-bottom")
	radius := aperture * personalValue("cage-radius")
	upper := aperture * personalValue("tube-radius")
	thickness := aperture * personalValue("cage-thickness")
	for _, station := range []struct{ height, radius float64 }{
		{lower, radius}, {length, upper},
	} {
		at := maths.NewSubspace(maths.YAxis.Scale(station.height),
			maths.Identity, spaceOne).Within(pose)
		assembly.add(groundDrum(station.radius,
			station.radius-thickness, thickness),
			at, spaceStructure)
	}
	pairs := int(personalValue("truss-pairs"))
	spread := personalValue("truss-spread-deg") * maths.Degree
	for index := range pairs {
		angle := fullTurn * float64(index) / float64(pairs)
		start := maths.V(radius*math.Cos(angle),
			lower+thickness*spaceHalf, -radius*math.Sin(angle))
		from := pose.Out(start)
		for _, sign := range []float64{-spaceOne, spaceOne} {
			next := angle + sign*spread
			end := maths.V(upper*math.Cos(next),
				length+thickness*spaceHalf,
				-upper*math.Sin(next))
			to := pose.Out(end)
			girth := aperture * groundValue("truss-radius")
			assembly.rod(from, to, girth,
				spaceStructure)
		}
	}
}

// personalDiagonal turns incoming light toward the Newtonian's side focuser.
// Its flat face is a visual fold, not a calibrated ray-traced prescription.
func personalDiagonal(assembly *instrumentAssembly,
	aperture, length, rim float64, pose maths.Subspace,
) {
	orientation := maths.Rotation(maths.ZAxis,
		personalValue("diagonal-tilt-deg")*maths.Degree)
	local := maths.NewSubspace(maths.YAxis.Scale(length),
		orientation, spaceOne).Within(pose)
	radius := aperture * personalValue("diagonal-radius")
	thick := aperture * personalValue("objective-thickness")
	mirror := groundDrum(radius, spaceZero, thick)
	assembly.add(mirror, local, spaceSecondary)
	groundObservatory{aperture: aperture}.secondaryRods(assembly,
		pose, rim, length, radius)
}

// personalFocuser houses the diagonal and its joints.
func personalFocuser(assembly *instrumentAssembly,
	pose maths.Subspace, aperture float64,
) {
	rear := -aperture * personalValue("rear-focuser")
	start := pose.Out(maths.YAxis.Scale(rear))
	joint := aperture * personalValue("focuser-joint")
	radius := aperture * personalValue("focuser-radius")
	tube := start.Add(pose.Orientation.Y.Scale(joint))
	assembly.rod(tube, pose.Origin, radius, spaceStructure)
	width := aperture * personalValue("diagonal-housing")
	half := width * spaceHalf
	localBox(assembly, pose, width, width, width,
		maths.V(-half, rear-half, -half), spaceStructure)
	reach := aperture * personalValue("diagonal-length")
	end := start.Add(pose.Orientation.Z.Scale(reach))
	bend := start.Add(pose.Orientation.Z.Scale(joint))
	assembly.rod(bend, end, radius, spaceStructure)
	reach = aperture * personalValue("eyepiece-length")
	eyepiece := end.Add(pose.Orientation.Y.Scale(reach))
	assembly.rod(end, eyepiece, radius, personalEyepiece)
}

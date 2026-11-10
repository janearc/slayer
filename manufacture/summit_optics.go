package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// summitRoof and summitLens are private roles beyond the personal eyepiece.
const (
	summitRoof slot = personalEyepiece + spaceOne
	summitLens      = summitRoof + spaceOne
)

// looks keeps structural materials separate from the opaque lens proxies.
func (row summitInstrument) looks() looks {
	paint := groundLooks()
	paint[summitRoof] = groundPaint("observatory enclosure",
		"summit-roof-reflectance")
	shade := maths.RGB(personalValue("glass-red"),
		personalValue("glass-green"), personalValue("glass-blue"))
	paint[summitLens] = personalPolished("corrector or objective proxy", shade)
	return paint
}

// reflector gives open trusses and closed tubes distinct structures.
func (row summitInstrument) reflector(assembly *instrumentAssembly) {
	aperture := row.aperture()
	pose, base := row.frame(), row.ground()
	rim := aperture * groundValue("truss-rim")
	if row.Family == "open" {
		base.truss(assembly, pose, rim, row.Length)
	} else {
		wall := aperture * groundValue("tube-wall")
		assembly.add(groundDrum(rim, rim-wall, row.Length), pose, leaf)
	}
	primary := mirrorDish{radius: aperture * spaceHalf,
		depth: aperture * groundValue("mirror-sag")}
	assembly.add(primary, pose, spaceMirror)
	if row.Family != "camera" {
		base.secondaryOn(assembly, pose, rim, row.Length)
	}
	if row.Family == "camera" || row.Family == "ranging" {
		row.corrector(assembly, pose)
	}
}

// corrector is an opaque display surface, not a refractive prescription.
func (row summitInstrument) corrector(assembly *instrumentAssembly,
	pose maths.Subspace,
) {
	aperture := row.aperture()
	gap := aperture * summitValue("lens-gap")
	local := maths.NewSubspace(maths.YAxis.Scale(row.Length+gap),
		maths.Identity, spaceOne).Within(pose)
	thickness := aperture * personalValue("objective-thickness")
	assembly.add(groundDrum(aperture*spaceHalf, spaceZero, thickness),
		local, summitLens)
}

// offAxis leaves the clear pupil free of a central secondary or spider.
// The shallow figure is scenery, not a cut from DKIST's parent paraboloid.
func (row summitInstrument) offAxis(assembly *instrumentAssembly) {
	aperture, pose := row.aperture(), row.frame()
	primary := mirrorDish{radius: aperture * spaceHalf,
		depth: aperture * groundValue("mirror-sag")}
	assembly.add(primary, pose, spaceMirror)
	offset := aperture * summitValue("off-axis-offset")
	centre := maths.V(offset, row.Length, spaceZero)
	secondary := tiltedFrame(centre, math.Pi).Within(pose)
	mirror := mirrorDish{radius: aperture * summitValue("off-axis-secondary"),
		depth: aperture * summitValue("off-axis-sag")}
	assembly.add(mirror, secondary, spaceSecondary)
	row.offAxisArm(assembly, pose, centre)
}

// offAxisArm runs outside the pupil before reaching the folded beam.
func (row summitInstrument) offAxisArm(assembly *instrumentAssembly,
	pose maths.Subspace, centre maths.Vec,
) {
	aperture := row.aperture()
	start := maths.V(centre.X, -aperture*groundValue("rim-thickness"),
		spaceZero)
	foot := aperture * summitValue("off-axis-foot")
	back := pose.Out(maths.XAxis.Scale(foot))
	base, top := pose.Out(start), pose.Out(centre)
	girth := aperture * summitValue("off-axis-arm")
	backward := back.Sub(base).Unit()
	landing := base.Add(backward.Scale(girth))
	lifted := base.Add(pose.Orientation.Y.Scale(girth))
	assembly.rod(back, landing, girth, spaceStructure)
	assembly.rod(lifted, top, girth, spaceStructure)
	width := aperture * summitValue("off-axis-joint")
	half := width * spaceHalf
	corner := start.Sub(maths.V(half, half, half))
	localBox(assembly, pose, width, width, width, corner, spaceStructure)
}

// twin keeps two parallel optical axes on a common support.
func (row summitInstrument) twin(assembly *instrumentAssembly) {
	offset := row.aperture() * summitValue("twin-axis-offset")
	for _, sign := range []float64{-spaceOne, spaceOne} {
		copy := row
		copy.Family = "closed"
		pose := copy.frame()
		pose.Origin.X += sign * offset
		copy.reflectorAt(assembly, pose)
	}
}

// reflectorAt translates a complete tube assembly, preserving its materials.
func (row summitInstrument) reflectorAt(assembly *instrumentAssembly,
	pose maths.Subspace,
) {
	local := &instrumentAssembly{fine: assembly.fine,
		object: good{dressed: assembly.object.dressed}}
	row.reflector(local)
	anchor := row.frame()
	delta := pose.Origin.Sub(anchor.Origin)
	shift := maths.NewSubspace(delta,
		maths.Identity, spaceOne)
	for _, piece := range local.object.parts {
		piece.space = piece.space.Within(shift)
		assembly.object.parts = append(assembly.object.parts, piece)
	}
	if assembly.trouble == nil {
		assembly.trouble = local.trouble
	}
}

// solarTube supplies a long visual refractor inside the Mees facility proxy.
// Its aperture is picked; the references give no instrument size.
func (row summitInstrument) solarTube(assembly *instrumentAssembly) {
	aperture, pose := row.aperture(), row.frame()
	radius := aperture * personalValue("tube-radius")
	wall := aperture * personalValue("tube-wall")
	assembly.add(groundDrum(radius, radius-wall, row.Length), pose, leaf)
	row.corrector(assembly, pose)
	rear := pose.Out(maths.YAxis.Scale(-aperture))
	assembly.rod(rear, pose.Origin,
		aperture*personalValue("focuser-radius"), spaceStructure)
}

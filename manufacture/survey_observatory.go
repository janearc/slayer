package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/research"
)

// These material roles are defined after the existing instrument slots.
const (
	surveyTertiary slot = spaceSecondary + spaceOne
	surveyBacking       = surveyTertiary + spaceOne
	surveyCamera        = surveyBacking + spaceOne
)

type surveyOptics struct{ secondary, tertiary float64 }

// surveyGround keeps physical mirror sizes separate from the picked building.
func surveyGround() groundObservatory {
	optics := &surveyOptics{
		secondary: research.Value("rubin-secondary-metres"),
		tertiary:  research.Value("rubin-tertiary-metres")}
	return groundObservatory{
		diameter: groundValue("survey-enclosure-diameter"),
		height:   groundValue("survey-enclosure-height"),
		aperture: research.Value("rubin-primary-metres"),
		survey:   optics}
}

// assemble puts two figures on a common backing under a short truss.
func (survey surveyOptics) assemble(row groundObservatory,
	assembly *instrumentAssembly, pose maths.Subspace,
) {
	tertiary := groundMirror("tertiary mirror")
	assembly.object.dressed[surveyTertiary] = tertiary
	assembly.object.dressed[surveyBacking] = groundPaint("mirror backing",
		"survey-blank-reflectance")
	assembly.object.dressed[surveyCamera] = groundPaint("camera housing",
		"survey-camera-reflectance")
	length := row.aperture * groundValue("survey-length")
	rim := row.aperture * groundValue("truss-rim")
	row.truss(assembly, pose, rim, length)
	survey.sharedBlank(row.aperture, assembly, pose)
	row.secondaryOn(assembly, pose, rim, length)
	surveyCameraOn(row.aperture, length, assembly, pose)
}

// sharedBlank joins the two visual figures through one backing volume.
func (survey surveyOptics) sharedBlank(aperture float64,
	assembly *instrumentAssembly, pose maths.Subspace,
) {
	outer, inner := aperture*spaceHalf, survey.tertiary*spaceHalf
	sag := aperture * groundValue("mirror-sag")
	gap := groundValue("survey-surface-gap")
	primary := surveyMirror{outer: outer, inner: inner + gap,
		depth: sag, backing: aperture * groundValue("mirror-thickness")}
	assembly.add(primary, pose, spaceMirror)
	depth := sag * groundValue("survey-tertiary-sag")
	join := sag * inner * inner / (outer * outer)
	height := join - depth
	shift := maths.NewSubspace(maths.YAxis.Scale(height),
		maths.Identity, spaceOne).Within(pose)
	tertiary := mirrorDish{radius: inner, depth: depth}
	assembly.add(tertiary, shift, surveyTertiary)
	thickness := aperture * groundValue("survey-blank-depth")
	radius := outer * groundValue("survey-blank-radius")
	backing := groundDrum(radius, spaceZero, thickness)
	bottom := maths.YAxis.Scale(height - gap - thickness)
	at := maths.NewSubspace(bottom, maths.Identity, spaceOne).Within(pose)
	assembly.add(backing, at, surveyBacking)
}

// surveyCameraOn keeps the housing clear of the secondary opening.
func surveyCameraOn(aperture, length float64,
	assembly *instrumentAssembly, pose maths.Subspace,
) {
	depth := aperture * groundValue("survey-camera-length")
	start := length - depth*groundValue("survey-camera-seat")
	from := pose.Out(maths.YAxis.Scale(start))
	to := pose.Out(maths.YAxis.Scale(start + depth))
	radius := aperture * groundValue("survey-camera-radius")
	assembly.rod(from, to, radius, surveyCamera)
}

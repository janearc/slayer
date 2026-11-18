package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// frame gives the compact survey instrument a broad double top ring and
// paired diagonal stays. Photographic proportions are checked authored data;
// see doc/survey-observatory.md for the Rubin mount and optical references.
func (survey surveyOptics) frame(row groundObservatory, assembly *instrumentAssembly,
	pose maths.Subspace, radius, length float64,
) {
	assembly.object.dressed[leaf] = groundPaint("Rubin optical frame", "survey-frame-reflectance")
	thickness := row.aperture * groundValue("rim-thickness")
	depth := row.aperture * groundValue("survey-top-ring-depth")
	lower := radius * groundValue("survey-frame-waist")
	for _, ring := range [][2]float64{{lower, -thickness}, {radius, length - depth}, {radius, length}} {
		at := maths.NewSubspace(maths.YAxis.Scale(ring[1]), maths.Identity, spaceOne).Within(pose)
		assembly.add(groundDrum(ring[0], ring[0]-thickness, thickness), at, leaf)
	}
	defer assembly.detailSince(len(assembly.object.parts))
	count := int(groundValue("survey-frame-stations"))
	girth := row.aperture * groundValue("survey-frame-member")
	point := func(angle, radius, y float64) maths.Vec {
		return maths.V(radius*math.Sin(angle), y, -radius*math.Cos(angle))
	}
	for index := range count {
		angle := fullTurn * float64(index) / float64(count)
		foot := point(angle, lower-thickness*spaceHalf, -thickness*spaceHalf)
		for _, sign := range []float64{-spaceOne, spaceOne} {
			top := point(angle+sign*math.Pi/float64(count), radius-thickness*spaceHalf, length-depth)
			surveyMember(assembly, pose, foot, top, girth)
		}
		from := point(angle, radius-thickness*spaceHalf, length-depth+thickness)
		to := point(angle, radius-thickness*spaceHalf, length)
		surveyMember(assembly, pose, from, to, girth)
		if index%2 == 0 {
			survey.spider(row, assembly, pose, angle, radius-thickness, length)
		}
	}
}

// spider supports M2 and the separate camera without filling the entrance pupil.
func (survey surveyOptics) spider(row groundObservatory, assembly *instrumentAssembly,
	pose maths.Subspace, angle, outer, length float64,
) {
	direction := maths.V(math.Sin(angle), 0, -math.Cos(angle))
	girth := row.aperture * groundValue("survey-spider-radius")
	for _, station := range [][2]float64{
		{length, survey.secondary * spaceHalf},
		{length*groundValue("survey-camera-station") + row.aperture*groundValue("survey-camera-length"), row.aperture * groundValue("survey-camera-radius")},
	} {
		from := direction.Scale(outer).Add(maths.YAxis.Scale(length))
		to := direction.Scale(station[1]).Add(maths.YAxis.Scale(station[0]))
		surveyMember(assembly, pose, from, to, girth)
	}
}

// surveySecondary is a closed convex visual figure with an intact centre.
// Its curvature is authored, not the Rubin optical prescription.
type surveySecondary struct{ radius, depth, backing float64 }

// skin revolves a solid backing and a face bulging toward the primary.
func (mirror surveySecondary) skin(fine cut) surface {
	points := [][coordinatePair]float64{{0, -mirror.backing}, {mirror.radius, -mirror.backing}, {mirror.radius, 0}}
	steps := max(int(groundValue("survey-radial-minimum")), fine.around/int(groundValue("survey-radial-cuts-per-step")))
	for index := steps - 1; index >= 0; index-- {
		radius := mirror.radius * float64(index) / float64(steps)
		points = append(points, [coordinatePair]float64{radius, mirror.depth * (1 - radius*radius/(mirror.radius*mirror.radius))})
	}
	return (turnedProfile{points: points}).skin(fine)
}

// surveyMember leaves distinct end caps where several closed braces meet.
func surveyMember(assembly *instrumentAssembly, pose maths.Subspace, from, to maths.Vec, radius float64) {
	gap := to.Sub(from).Unit().Scale(groundValue("rib-joint-metres"))
	assembly.rod(pose.Out(from.Add(gap)), pose.Out(to.Sub(gap)), radius, leaf)
}

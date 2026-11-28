package primates

import (
	"math"

	"github.com/janearc/slayer/bioforms/hair"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// mediumCrownPin bends a subset of the crown fibres above the existing
// paths. Their unchanged lead and varied target distances help preserve
// coverage below the loft.
func (spec ScalpHair) mediumCrownPin(guides mediumGuides, site hair.Site, heading maths.Vec,
	edge scalpEdge, part float64,
) hair.GuidePin {
	point, normal := site.Chart, site.ChartNormal
	dome := maths.Smooth(1.630, 1.676, point.Y)
	dome *= maths.Smooth(.024, .038, math.Abs(part)) * maths.Smooth(.019, .030, edge.depth)
	if dome == 0 {
		return hair.GuidePin{}
	}

	// Use the barycentric site so the two cuts share the same
	// candidate roots and this choice stays stable when a root poses.
	u := int64(site.Weights[1] * (1 << 30))
	v := int64(site.Weights[2] * (1 << 30))
	seed := spec.Seed ^ 0x63726f776e6c6f66
	pick := noise.Hash3(seed, int64(site.Face), u, v)
	const liftedShare = .38
	if pick >= liftedShare {
		return hair.GuidePin{}
	}
	phase := pick / liftedShare
	spread := noise.Hash3(seed+1, int64(site.Face), u, v)

	first, second := guides.field.Weights(site, mediumSide(part)).Leaders()
	weight := spec.CrownLift * dome * maths.Smooth(.02, .23, first.Share-second.Share)
	if weight < .01 {
		return hair.GuidePin{}
	}
	lock := guides.main[first.Anchor]
	course := heading.Sub(normal.Scale(heading.Dot(normal))).Unit()
	if course == (maths.Vec{}) {
		return hair.GuidePin{}
	}
	across := normal.Cross(course).Unit()
	converge := lock.root.Sub(point).Dot(across)
	converge = max(-.016, min(.016, converge))
	fullness := .5 + lock.lift/.0038
	step := .023 + .016*phase
	lateral := (.12+.19*spread)*converge + .005*(spread-.5)
	target := point.Add(course.Scale(step)).Add(across.Scale(lateral)).Add(normal.Scale(.005))
	lift := .006 + .013*fullness + .004*spread
	attachment, on, ok := guides.pins.Project(target, lift, (.43+.27*spread)*weight)
	if !ok || on.Sub(target).Len() > .019 || part*spec.mediumPart(on) < 0 ||
		spec.Line.edge(on, spec.Seed).depth < .007 {
		return hair.GuidePin{}
	}
	return attachment
}

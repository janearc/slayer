package primates

import (
	"math"

	"github.com/janearc/slayer/bioforms/hair"
	"github.com/janearc/slayer/maths"
)

func (spec ScalpHair) mediumTangentField(surface hair.Surface) (*hair.TangentField, error) {
	var barrier func(maths.Vec, maths.Vec) bool
	if spec.FieldWisps != nil {
		barrier = spec.mediumWispPart
	}
	return hair.NewTangentField(hair.TangentFieldSpec{
		Surface: surface,
		Domain: func(point maths.Vec) int {
			region := spec.mediumScalpRegion(point)
			if region < 0 || spec.FieldWisps == nil {
				return region
			}
			return 0
		},
		Barrier:    barrier,
		Constraint: spec.mediumTangent,
	})
}

func (spec ScalpHair) mediumScalpRegion(point maths.Vec) int {
	if point.Y < 1.512 {
		return -1
	}
	edge := spec.Line.edge(point, spec.Seed)
	if edge.depth < -.006 || edge.ear < .08 {
		return -1
	}
	return mediumSide(spec.mediumPart(point))
}

// mediumTangent seeds the part, hairline, temple and nape directions.
// The whorl is an explicit turning region with a small fixed core.
func (spec ScalpHair) mediumTangent(point, normal maths.Vec) (maths.Vec, float64) {
	part := spec.mediumPart(point)
	edge := spec.Line.edge(point, spec.Seed)
	side := math.Copysign(1, part)
	rear := scalpCourse(normal)
	tangent := func(direction maths.Vec) maths.Vec {
		return direction.Sub(normal.Scale(direction.Dot(normal))).Unit()
	}
	out := tangent(maths.V(side, 0, 0))
	down := tangent(maths.V(0, -1, 0))
	low := 1 - maths.Smooth(1.583, 1.649, point.Y)
	temple := maths.Smooth(.033, .069, math.Abs(point.X)) * low
	nape := maths.Smooth(.027, .084, -point.Z) * low
	away := .62 * maths.Smooth(.003, .049, math.Abs(part)) * (1 - .68*nape)
	joined := 0.
	if style := spec.FieldWisps; style != nil {
		joined = maths.Smooth(style.PartReach-.023, style.PartReach+.013, -point.Z)
		away *= 1 - .79*joined*(1-maths.Smooth(.011, .053, math.Abs(part)))
	}
	direction := rear.Add(out.Scale(away)).Unit()
	direction = mixPoint(direction, down, min(.89, .70*temple+.68*nape)).Unit()

	before, after := point, point
	before.Z -= .003
	after.Z += .003
	slope := (spec.mediumPart(before) - spec.mediumPart(after)) / .006
	partTangent := tangent(maths.V(-slope, 0, -1))
	blend := spec.FieldWisps != nil && spec.FieldWisps.BlendPart
	if blend {
		// Near the crown, the intersection of the scalp and part
		// tangent planes follows the curve on either side of the part.
		crown := 1 - maths.Smooth(.006, .046, point.Z)
		alongLine := normal.Cross(maths.V(1, 0, -slope)).Unit()
		partTangent = mixPoint(partTangent, alongLine, crown).Unit()
	}
	alongPart := mediumPartExtent(point) * (1 - maths.Smooth(.002, .016, math.Abs(part))) * (1 - joined)
	direction = mixPoint(direction, partTangent, .88*alongPart).Unit()

	whorl := maths.V(point.X-.023-.23*spec.Part, 0, point.Z+.044)
	if style := spec.FieldWisps; style != nil {
		whorl = maths.V(point.X-.028-.21*spec.Part, 0, point.Z+style.PartReach+.019)
	}
	radius := whorl.Len()
	if radius < .0028 && point.Y > 1.645 && !blend {
		return maths.Vec{}, 1
	}
	around := tangent(maths.V(whorl.Z, 0, -whorl.X))
	if spec.CounterClockwise {
		around = around.Neg()
	}
	spin := .94 * bell(radius/.031) * maths.Smooth(1.605, 1.656, point.Y) * (1 - alongPart)
	if spec.FieldWisps != nil {
		spin *= .86 * joined
	}
	direction = mixPoint(direction, around, spin).Unit()
	hairline := 1 - maths.Smooth(.001, .023, edge.depth)
	weight := min(.96, .12+.71*alongPart+.54*hairline+.42*spin+.19*temple+.21*nape)
	if blend {
		direction, weight = spec.mediumPartBridge(point, normal, rear, part, direction, weight)
	}
	return direction, weight
}

// mediumPartBridge carries a narrow coherent strip over the former
// seam. The separate frontal courses ease into it before they cross.
func (spec ScalpHair) mediumPartBridge(point, normal, rear maths.Vec, part float64, heading maths.Vec, weight float64) (maths.Vec, float64) {
	reach := spec.FieldWisps.PartReach
	back := -point.Z
	// Broaden the join while the two courses pass the whorl. The
	// broader interval lets the lateral heading turn out gradually.
	spread := maths.Smooth(reach+.006, reach+.022, back) *
		(1 - maths.Smooth(reach+.036, reach+.056, back))
	acrossPart := 1 - maths.Smooth(.010+.012*spread, .047+.032*spread, math.Abs(part))
	bridge := maths.Smooth(reach-.027, reach+.009, back) *
		(1 - maths.Smooth(reach+.042, reach+.074, back)) *
		acrossPart *
		maths.Smooth(1.608, 1.648, point.Y)
	hand := 1.
	if spec.CounterClockwise {
		hand = -1
	}
	across := rear.Cross(normal).Unit()
	drift := .29 * hand * maths.Smooth(reach+.002, reach+.026, back)
	bridgeHeading := rear.Add(across.Scale(drift)).Unit()
	return mixPoint(heading, bridgeHeading, .94*bridge).Unit(), max(weight, .93*bridge)
}

func (spec ScalpHair) fieldMediumStrand(root mediumRoot, covered, inside, grain, radius float64) hair.Strand {
	site := root.site
	shape := spec.mediumShape(site.Chart, site.ChartNormal, inside, grain)
	nape := shape.behind * (1 - shape.crown)
	length := mix(mix(spec.SideLength, spec.NapeLength, nape), spec.Length, shape.crown)
	length *= (.78 + .22*inside) * (1 + .055*grain + .038*shape.lock)
	contact := .043 + .052*shape.crown + .017*shape.nearLine
	lift := .0028 + .0031*shape.crown + .0014*shape.lock
	taper := hair.ShaftTaper{Start: .84 + .017*shape.lock, TipShare: .48}
	switch root.population {
	case scalpMargin:
		length *= mix(.40, .83, inside)
		radius *= .78
		taper = hair.ShaftTaper{Start: .73, TipShare: .34}
	case scalpFringe:
		length = .0045 + .004*inside + .0019*grain
		lift, contact = .0007, length
		radius *= .49
		taper = hair.ShaftTaper{}
	case scalpLoose:
		length *= 1.08
		contact *= .92
		lift *= 1.35
		radius *= .72
		taper = hair.ShaftTaper{}
	}
	lean := -shape.side * math.Tanh(spec.Part/.017)
	fall := maths.V(.085*shape.side*(1-shape.behind), -1,
		-.12+.20*spec.MediumSweep*lean*(1-shape.behind))
	path := hair.SurfaceCourseSpec{
		Segments: root.steps, Contact: min(contact, length), Length: length, Lift: lift, Fall: fall,
	}
	if root.wisps != nil && root.population != scalpFringe {
		envelope, rise := root.wisps.Shape(site, mediumSide(shape.part), root.field.Heading(site))
		share := maths.Smooth(.003, .022, math.Abs(shape.part)) * (.30 + .70*inside)
		path.Lift = max(.00055, path.Lift+share*rise)
		path.Envelope = hair.CourseEnvelope{Shoulder: envelope.Shoulder.Scale(share), Tip: envelope.Tip.Scale(share)}
	}
	course, err := root.field.Course(site, path)
	if err != nil {
		return hair.Strand{Coverage: covered}
	}
	return hair.Strand{
		Coverage: covered, Flow: site.ChartFlow(scalpCourse(site.ChartNormal)),
		Length: length, Radius: radius, Taper: taper, Course: course,
	}
}

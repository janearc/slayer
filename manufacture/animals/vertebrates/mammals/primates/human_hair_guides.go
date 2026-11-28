package primates

import (
	"math"

	"github.com/janearc/slayer/bioforms/hair"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

type mediumTargets struct {
	release, tip, heading maths.Vec
	bend                  maths.Vec // Offset at the free length's bend.
	lift                  float64   // Change to the proximal sweep in metres.
	fall                  float64   // Share of the chart-vertical drape.
}

type mediumGuides struct {
	field        *hair.SurfaceGuides
	pins         *hair.PinSurface
	main, margin []mediumLock
}

func (spec ScalpHair) mediumGuideField(surface hair.Surface) (mediumGuides, error) {
	field, err := hair.NewSurfaceGuides(hair.GuideFieldSpec{
		Surface: surface,
		Seed:    noise.Mix(spec.Seed ^ 0x6d69646c6f636b),
		Spacing: .026,
		Region: func(point maths.Vec) int {
			if point.Y < 1.512 {
				return -1
			}
			edge := spec.Line.edge(point, spec.Seed)
			if edge.depth < -.012 || edge.ear < .12 {
				return -1
			}
			return mediumSide(spec.mediumPart(point))
		},
	})
	if err != nil {
		return mediumGuides{}, err
	}
	pins, err := hair.NewPinSurface(surface, func(points [3]maths.Vec) bool {
		return min(points[0].Y, points[1].Y, points[2].Y) > 1.588
	})
	if err != nil {
		return mediumGuides{}, err
	}
	guides := mediumGuides{field: field, pins: pins}
	for index, anchor := range field.Anchors() {
		point := anchor.Point
		edge := spec.Line.edge(point, spec.Seed)
		inside := maths.Smooth(.002, .025, edge.depth)
		shape := spec.mediumShape(point, anchor.Normal, inside, 0)
		guides.main = append(guides.main, spec.mediumLock(anchor, shape, index))
		shape.trim = mediumMarginTrim(inside)
		guides.margin = append(guides.margin, spec.mediumLock(anchor, shape, index))
	}
	return guides, nil
}

// mediumPartPins follows the curved line close to the scalp. Beyond
// the visible part, the handles turn into the rear course at varying
// distances. The follicle keeps its original attachment.
func (spec ScalpHair) mediumPartPins(pins *hair.PinSurface, site hair.Site, shape mediumReach) hair.GuidePins {
	point := site.Chart
	front := maths.Smooth(-.028, 0, point.Z) * maths.Smooth(1.598, 1.627, point.Y)
	offset := math.Abs(shape.part)
	extension := .63 * shape.rearCourse * (1 - maths.Smooth(.041, .070, -point.Z))
	weight := mix(front, 1, extension) * (1 - maths.Smooth(.002, .023, offset))
	if weight < .01 {
		return hair.GuidePins{}
	}
	rearward := scalpCourse(site.ChartNormal)
	hand := 1.
	if spec.CounterClockwise {
		hand = -1
	}
	pin := func(step, diverge, lift, share float64) hair.GuidePin {
		reach := step * (1 + shape.rearCourse*(.19*shape.grain+.15*shape.lock))
		target := point.Add(rearward.Scale(reach))
		alongPart := target.X + shape.side*(offset+diverge) - spec.mediumPart(target)
		posterior := target.X + hand*(.004+.002*shape.grain)
		turn := shape.rearCourse * (1 - .80*mediumPartExtent(target))
		target.X = mix(alongPart, posterior, turn)
		target = target.Add(site.ChartNormal.Scale(.004))
		attachment, chart, ok := pins.Project(target, lift*(1-.24*turn), share*weight)
		if !ok || chart.Sub(target).Len() > .016 {
			return hair.GuidePin{}
		}
		if shape.side*spec.mediumPart(chart) < -.00035 && (turn == 0 || mediumPartExtent(chart) > .24) {
			return hair.GuidePin{}
		}
		return attachment
	}
	return hair.GuidePins{
		Lead: pin(.016, .0004, .0018, .92),
		Bend: pin(.033, .0065, .0031, .68),
	}
}

// mediumHairlinePin pulls the first handle close to the scalp just past
// the frontal hairline. The margin follows the growth course there
// before its free length leaves the scalp toward the ear.
func (spec ScalpHair) mediumHairlinePin(pins *hair.PinSurface, site hair.Site, edge scalpEdge, part float64) hair.GuidePin {
	point := site.Chart
	front := maths.Smooth(-.012, .046, point.Z) * maths.Smooth(1.595, 1.623, point.Y)
	weight := front * (1 - maths.Smooth(.003, .019, edge.depth))
	weight *= maths.Smooth(.014, .028, math.Abs(part))
	if weight < .01 {
		return hair.GuidePin{}
	}
	course := spec.flow(site, maths.Smooth(1.580, 1.685, point.Y), part, mediumPartExtent(point), .1*edge.stagger)
	target := point.Add(course.Scale(.014 + .003*edge.stagger)).Add(site.ChartNormal.Scale(.003))
	near, chart, ok := pins.Project(target, .0015, .74*weight)
	if !ok || chart.Sub(target).Len() > .012 || part*spec.mediumPart(chart) < 0 ||
		spec.Line.edge(chart, spec.Seed).depth < edge.depth-.002 {
		return hair.GuidePin{}
	}
	return near
}

func (spec ScalpHair) mediumShape(point, normal maths.Vec, inside, grain float64) mediumReach {
	part := spec.mediumPart(point)
	extent := mediumPartExtent(point)
	high := maths.Smooth(1.580, 1.685, point.Y)
	stagger := noise.Gradient(spec.Seed+63, 27*point.X, 17*point.Y, 22*point.Z)
	travel := -point.Z + .007*stagger + .003*grain
	return mediumReach{
		side: math.Copysign(1, part), part: part, inside: inside, grain: grain,
		crown:    min(1, max(0, normal.Y)) * high,
		behind:   maths.Smooth(.002, .071, -point.Z) * (1 - .48*extent),
		lock:     noise.Gradient(spec.Seed+34, 32*point.X, 32*point.Y, 32*point.Z),
		nearLine: extent * (1 - maths.Smooth(.0012, .017, math.Abs(part))),
		rearCourse: maths.Smooth(.003, .040, travel) *
			maths.Smooth(1.589, 1.653, point.Y) *
			(1 - maths.Smooth(.017, .060, math.Abs(part))),
	}
}

func mediumPartExtent(point maths.Vec) float64 {
	return maths.Smooth(-.029, -.004, point.Z) * maths.Smooth(1.600, 1.646, point.Y)
}

func mediumSide(part float64) int {
	if part < 0 {
		return 0
	}
	return 1
}

// blend gives neighboring strands shared release and end curves while
// varying the loose tips independently. Toward the nape, more of the
// local field remains so the part separator does not create a rear seam.
func (guides mediumGuides) blend(site hair.Site, shape mediumReach, local mediumTargets, population scalpPopulation) mediumTargets {
	if population == scalpLoose {
		return local
	}
	targets := guides.main
	if population == scalpMargin {
		targets = guides.margin
	}
	weights := guides.field.Weights(site, mediumSide(shape.side))
	var release, tip, heading maths.Vec
	total := 0.
	for _, choice := range weights {
		if choice.Share == 0 {
			continue
		}
		target := targets[choice.Anchor].targets
		release = release.Add(target.release.Scale(choice.Share))
		tip = tip.Add(target.tip.Scale(choice.Share))
		heading = heading.Add(target.heading.Scale(choice.Share))
		total += choice.Share
	}
	if total == 0 {
		return local
	}
	share := (.76 - .38*shape.behind) * (1 - shape.rearCourse)
	local.release = mixPoint(local.release, release, share)
	local.tip = mixPoint(local.tip, tip, share)
	local.heading = mixPoint(local.heading, heading, share).Unit()

	first, second := weights.Leaders()
	focus := maths.Smooth(.03, .28, first.Share-second.Share)
	focus *= (.42 + .58*shape.inside) * (1 - .82*shape.nearLine)
	focus *= 1 - shape.rearCourse
	return targets[first.Anchor].shape(site.Chart, local, focus)
}

// mediumHeading gives each guide an initial tangent in the chart. Close
// to the parted line growth follows the scalp towards the back before
// the guide bends out toward the temple.
func mediumHeading(point, normal, release maths.Vec, shape mediumReach) maths.Vec {
	toward := release.Sub(point)
	side := .025 * shape.side * (1 - shape.behind) * (1 - shape.rearCourse)
	direction := toward.Add(maths.V(side, .028*shape.crown, -.012))
	direction = direction.Sub(normal.Scale(direction.Dot(normal))).Unit()
	return mixPoint(direction, scalpCourse(normal), .66*shape.nearLine).Unit()
}

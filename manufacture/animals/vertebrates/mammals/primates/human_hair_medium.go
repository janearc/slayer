package primates

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/bioforms/hair"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

func (spec ScalpHair) mediumLayers(surface hair.Surface) ([]hair.Layer, error) {
	if spec.Density == 0 {
		spec.Density = 370_000
	}
	if spec.Length == 0 {
		spec.Length = .234
	}
	if spec.SideLength == 0 {
		spec.SideLength = .171
	}
	if spec.NapeLength == 0 {
		spec.NapeLength = .110
	}
	if spec.Radius == 0 {
		spec.Radius = .000076
	}
	if spec.Wavelength == 0 {
		spec.Wavelength = .142
	}
	if spec.Material == nil || !between(spec.Density, 15_000, 850_000) ||
		!between(spec.Length, .12, .34) || !between(spec.SideLength, .09, .28) ||
		!between(spec.NapeLength, .05, .22) || !between(spec.Radius, .000015, .00015) ||
		!between(spec.Part, -.038, .038) || !between(spec.Wavelength, .07, .30) ||
		!between(spec.Curl, 0, 1) || !between(spec.MediumSweep, 0, 1) ||
		!between(spec.CrownLift, 0, 1) || !spec.Line.valid() {
		return nil, fmt.Errorf("adult human: medium hair requires a material and supported cut controls")
	}
	var guides mediumGuides
	var field *hair.TangentField
	var wisps *hair.WispField
	var err error
	if spec.ScalpField {
		if spec.FieldWisps != nil {
			style := spec.FieldWisps.defaults()
			if !style.valid() {
				return nil, fmt.Errorf("adult human: scalp wisp controls are outside their supported range")
			}
			spec.FieldWisps = &style
			wisps, err = spec.mediumWisps(surface)
			if err != nil {
				return nil, err
			}
		}
		field, err = spec.mediumTangentField(surface)
	} else {
		guides, err = spec.mediumGuideField(surface)
	}
	if err != nil {
		return nil, err
	}
	layer := func(population scalpPopulation, seed uint64, count float64, steps, sides int) hair.Layer {
		return hair.Layer{
			Seed: seed, Density: count * spec.Density, Segments: steps, Sides: sides,
			Material: spec.Material, FaceFilter: scalpFace, SkinGap: .00025,
			Profile: func(site hair.Site) hair.Strand {
				return spec.mediumStrand(mediumRoot{site: site, population: population, guides: guides, field: field, wisps: wisps, steps: steps})
			},
		}
	}
	return []hair.Layer{
		layer(scalpMain, spec.Seed, 1, 14, 4),
		layer(scalpMargin, noise.Mix(spec.Seed^0x66726f6e746c696e), .28, 8, 3),
		layer(scalpFringe, noise.Mix(spec.Seed^0x6d656469756d6564), .42, 4, 3),
		layer(scalpLoose, noise.Mix(spec.Seed^0x6d656469756d7469), .017, 14, 3),
	}, nil
}

type mediumRoot struct {
	site       hair.Site
	population scalpPopulation
	guides     mediumGuides
	field      *hair.TangentField
	wisps      *hair.WispField
	steps      int
}

func (spec ScalpHair) mediumStrand(root mediumRoot) hair.Strand {
	site := root.site
	point := site.Chart
	edge := spec.Line.edge(point, spec.Seed)
	inside := maths.Smooth(.002, .025, edge.depth)
	high := maths.Smooth(1.580, 1.685, point.Y)
	againstPart := spec.mediumPart(point)
	partExtent := mediumPartExtent(point)
	nearPart := 1 - maths.Smooth(.00030, .00155, math.Abs(againstPart))
	covered := edge.coverage(root.population) * (1 - .32*partExtent*nearPart)
	if root.population == scalpLoose {
		covered *= inside * high
	}
	if covered <= 0 {
		return hair.Strand{}
	}

	grain := noise.Gradient(spec.Seed+71, 311*point.X, 311*point.Y, 311*point.Z)
	radius := spec.Radius * (.66 + .26*inside + .08*grain)
	if root.field != nil {
		return spec.fieldMediumStrand(root, covered, inside, grain, radius)
	}
	if root.population == scalpFringe {
		direction := spec.flow(site, high, againstPart, partExtent, .18*grain)
		return hair.Strand{
			Coverage: covered, Flow: site.ChartFlow(direction),
			Length: .0045 + .004*inside + .0019*grain, Radius: .49 * radius,
			Guide: spec.shaftGuide(.15, grain, 0, 0, scalpFringe), SkinFollow: .8,
		}
	}

	reach := spec.mediumShape(point, site.ChartNormal, inside, grain)
	lock := reach.lock
	taper := hair.ShaftTaper{Start: .84 + .017*lock, TipShare: .48}
	guideEnd := 1.
	if root.population == scalpMargin {
		radius *= .78
		reach.trim = mediumMarginTrim(inside)
		taper = hair.ShaftTaper{Start: .73, TipShare: .34}
		// The fine edge follows the same track as the nearby locks.
		// As roots approach the hairline, their guided lengths step
		// down into the independent short fringe.
		guideEnd = mix(.32, 1, maths.Smooth(.000, .017, edge.depth+.003*edge.stagger))
	} else if root.population == scalpLoose {
		radius *= .72
		reach.trim = -.07
		taper = hair.ShaftTaper{}
	}

	targets := root.guides.blend(site, reach, spec.mediumTargets(point, site.ChartNormal, reach), root.population)
	flow, guide, length := spec.mediumArc(site, targets, reach)
	var pins hair.GuidePins
	if root.population != scalpLoose {
		pins = spec.mediumPartPins(root.guides.pins, site, reach)
		if pins.Lead == (hair.GuidePin{}) {
			pins.Lead = spec.mediumHairlinePin(root.guides.pins, site, edge, againstPart)
		}
	}
	if spec.CrownLift > 0 && root.population == scalpMain && pins == (hair.GuidePins{}) {
		pins.Bend = spec.mediumCrownPin(root.guides, site, targets.heading, edge, againstPart)
	}
	return hair.Strand{
		Coverage: covered, Flow: flow, Radius: radius, Length: length,
		Guide: guide, GuideEnd: guideEnd, Pins: pins, Taper: taper, SkinFollow: .55 + .21*reach.behind,
	}
}

type mediumReach struct {
	side, crown, behind, inside float64
	grain, lock, trim, nearLine float64
	part, rearCourse            float64
}

func mediumMarginTrim(inside float64) float64 {
	return mix(.35, .16, inside)
}

// mediumTargets lays out the end of the scalp sweep and the end of the
// hanging lock. In front, the part separates the sweeps toward the
// temples. Beyond the part, both sides can flow down the occiput.
func (spec ScalpHair) mediumTargets(point, normal maths.Vec, shape mediumReach) mediumTargets {
	nape := shape.behind * (1 - shape.crown)
	length := mix(mix(spec.SideLength, spec.NapeLength, nape), spec.Length, shape.crown)
	length *= (.82 + .18*shape.inside) * (1 - shape.trim)
	sideX := shape.side * (.086 + .012*shape.crown + .005*shape.lock)
	hand := 1.
	if spec.CounterClockwise {
		hand = -1
	}
	turn := hand * math.Copysign(.008*shape.crown, point.X+.008)
	backX := point.X*.90 + .005*shape.side + turn
	release := maths.V(
		mix(sideX, backX, shape.behind),
		point.Y-.011-.27*length+.004*shape.lock+.002*shape.grain,
		mix(.37*point.Z-.023, min(-.074, point.Z-.010), shape.behind),
	)
	cut := 1.529 - .31*length - .012*nape
	drop := mix(release.Y-.40*length, cut, .64) + .012*shape.lock + .004*shape.grain
	drop = min(release.Y-.020, drop)
	tuck := .005 + .006*shape.lock
	backDepth := -.072 - .20*(drop-1.481)
	end := maths.V(
		mix(sideX-shape.side*tuck, backX*.93, shape.behind),
		drop,
		mix(release.Z-.010, backDepth, shape.behind)+.003*shape.grain,
	)
	// The part ends on the crown. In its rear continuation the
	// paths favour the same posterior direction and bend gently
	// with the whorl. The follicles remain independently sampled.
	whorl := hand * (.004 + .008*bell((point.Z+.038)/.026))
	// Retain spacing across the rear course so its edges ease into
	// the wider side sweeps without drawing a narrow bundle.
	alongBack := point.X + .27*shape.part + whorl
	release.X = mix(release.X, alongBack, shape.rearCourse)
	release.Z = mix(release.Z, -.103+.004*shape.lock, shape.rearCourse)
	end.X = mix(end.X, .90*alongBack, shape.rearCourse)
	end.Z = mix(end.Z, backDepth+.004*shape.grain, shape.rearCourse)
	targets := mediumTargets{release: release, tip: end}
	spec.drapeMedium(point, shape, &targets)
	targets.heading = mediumHeading(point, normal, targets.release, shape)
	return targets
}

// drapeMedium lets the broader side of an offset part arc forward
// over the cheek-side locks. The opposite side sweeps behind the ear.
// The front and crown roots carry the sweep into the free lengths
// while the low side and rear lengths stay closer to the base cut.
func (spec ScalpHair) drapeMedium(point maths.Vec, shape mediumReach, targets *mediumTargets) {
	front := maths.Smooth(-.042, .054, point.Z)
	high := maths.Smooth(1.545, 1.671, point.Y)
	width := spec.MediumSweep * front * (.31 + .69*high) * (1 - shape.rearCourse)
	side := shape.side
	lean := -side * math.Tanh(spec.Part/.017)
	span := mix(spec.SideLength/.171, spec.Length/.234, shape.crown)
	targets.release = targets.release.Add(maths.V(
		side*width*(.001+.005*lean), width*(.003+.008*lean), width*(.010+.031*lean),
	))
	targets.tip = targets.tip.Add(maths.V(
		side*width*(.002+.004*lean), -width*span*(.006+.027*lean), width*(.007+.029*lean),
	))
	targets.tip.Y = min(targets.tip.Y, targets.release.Y-.017)
	targets.bend = maths.V(
		side*width*(.005+.003*lean), 0, width*(.006+.011*lean),
	)
	targets.lift = width * (.001 + .0016*lean) * high
	targets.fall = width
}

// mediumArc maps the authored offsets into a physical follicle frame.
// It makes the base cubic leave the surface smoothly, then gives the
// second cubic the same join tangent and independently shaped ends.
func (spec ScalpHair) mediumArc(site hair.Site, targets mediumTargets, shape mediumReach) (maths.Vec, hair.Guide, float64) {
	point, normal := site.Chart, site.Normal
	flow := site.ChartFlow(targets.heading)
	flow = flow.Sub(normal.Scale(flow.Dot(normal))).Unit()
	if flow == (maths.Vec{}) {
		away := site.ChartFlow(scalpCourse(site.ChartNormal))
		flow = away.Sub(normal.Scale(away.Dot(normal))).Unit()
	}
	transverse := normal.Cross(flow).Unit()
	local := func(displacement maths.Vec) maths.Vec {
		offset := site.ChartVector(displacement)
		return maths.V(offset.Dot(flow), offset.Dot(normal), offset.Dot(transverse))
	}
	join := local(targets.release.Sub(point))
	join.X = max(.012, join.X)
	last := local(targets.tip.Sub(point))
	rear := site.ChartVector(maths.V(0, 0, -1)).Dot(transverse)
	lift := .006 + .007*shape.crown + .003*shape.lock - .008*shape.trim + targets.lift
	lift = max(.0008, lift)
	lead := maths.V((.24-.079*shape.nearLine)*join.X, lift,
		.12*join.Z+.018*shape.nearLine*rear)
	bend := maths.V(.73*join.X, .55*join.Y+.010+.010*shape.crown+.006*shape.lock, .78*join.Z)
	tail := last.Scale(.73).Add(join.Scale(.27)).Add(local(targets.bend))
	tail.Y += .005 + .003*shape.lock
	if targets.fall > 0 {
		// End the sweep in a descent and put the terminal handle above
		// the tip. The lock can keep its forward offset without bowing
		// sideways near the ear or turning back up at its end.
		down := local(maths.V(0, -1, 0)).Unit()
		fall := last.Sub(join)
		exit := fall.Unit().Add(down).Unit()
		approach := join.Sub(exit.Scale(min(.027, .50*fall.Len())))
		approach.X = max(.32*join.X, min(.94*join.X, approach.X))
		bend = mixPoint(bend, approach, .64*targets.fall)
		drop := max(.017, targets.release.Y-targets.tip.Y)
		terminal := last.Add(local(maths.V(0, .35*drop, 0)))
		tail = mixPoint(tail, terminal, targets.fall)
	}
	const split = .55
	start := join.Add(join.Sub(bend).Scale((1 - split) / split))
	length := max(lead.Len(), bend.Len(), join.Len(), start.Len(), tail.Len(), last.Len()) + .0002
	unit := 1 / length
	width := .053 * spec.Curl * (.45 + .55*shape.inside)
	guide := hair.Guide{
		Lead: lead.Scale(unit), Bend: bend.Scale(unit), Tip: join.Scale(unit),
		Tail: hair.GuideTail{At: split, Bend: tail.Scale(unit), Tip: last.Scale(unit)},
		Wave: hair.GuideWave{
			Width: width, Height: .34 * width, Turns: min(2, length/spec.Wavelength),
			Phase: 1.7*shape.lock + .60*shape.grain,
		},
	}
	return flow, guide, length
}

// mediumPart turns laterally from the forehead before losing the line
// in the crown. Its offsets are small on the scale of the style.
func (spec ScalpHair) mediumPart(point maths.Vec) float64 {
	stagger := noise.Gradient(spec.Seed+27, 92*point.Z, .61, 0)
	bend := .006 * math.Sin(54*(point.Z+.012))
	return spec.partDistance(point) - bend - .0027*stagger
}

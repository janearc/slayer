package primates

import (
	"math"

	"github.com/janearc/slayer/maths"
)

type limbSection struct {
	at, width, depth, bend float64
	centre                 maths.Vec
}

type limbSweep struct {
	rim      []int
	stations []limbSection
	site     proportionSite
	across   func(bend float64) maths.Vec
	forward  func(bend float64) maths.Vec
	detail   func(point maths.Vec, at, sine, cosine float64) maths.Vec
	weight   func(at float64, point maths.Vec, root jointWeights) jointWeights
	blendEnd float64
	rows     int
}

func (model *humanMesh) sweep(limb limbSweep) []int {
	first := limb.stations[0]
	angles := model.angles(limb.rim, model.centre(limb.rim), limb.across(first.bend),
		limb.forward(first.bend), first.width, first.depth)
	offsets := make([]maths.Vec, len(limb.rim))
	for i, index := range limb.rim {
		offsets[i] = model.vertices[index].point.Sub(limb.point(first, angles.near[i]))
	}
	previous := limb.rim
	for row := 1; row <= limb.rows; row++ {
		at := float64(row) / float64(limb.rows)
		ring := limbSectionAt(limb.stations, at)
		round := maths.Smooth(0, .52, at)
		carry := 1 - maths.Smooth(0, limb.blendEnd, at)
		site := limb.site
		site.attachment = 1 - carry
		next := make([]int, len(previous))
		for i, index := range limb.rim {
			point := limb.point(ring, angles.angle(i, round)).Add(offsets[i].Scale(carry))
			root := model.vertices[index].weights
			next[i] = model.limbPoint(point, limb.weight(at, point, root), site)
		}
		model.connect(previous, next)
		previous = next
	}
	return turnRim(previous, angles.front)
}

func (limb limbSweep) point(ring limbSection, angle float64) maths.Vec {
	sine, cosine := math.Sincos(angle)
	point := ring.centre.Add(limb.across(ring.bend).Scale(ring.width * sine)).
		Add(limb.forward(ring.bend).Scale(ring.depth * cosine))
	return limb.detail(point, ring.at, sine, cosine)
}

func hermite(start, end, from, to, share float64) float64 {
	square := share * share
	cube := share * square
	return (2*cube-3*square+1)*start + (cube-2*square+share)*from +
		(-2*cube+3*square)*end + (cube-square)*to
}

func limbSectionAt(stations []limbSection, at float64) limbSection {
	index := 0
	for index < len(stations)-2 && at > stations[index+1].at {
		index++
	}
	start, end := stations[index], stations[index+1]
	before := stations[max(0, index-1)]
	after := stations[min(len(stations)-1, index+2)]
	span := end.at - start.at
	share := (at - start.at) / span
	value := func(first, left, right, last float64) float64 {
		from := span * (right - first) / (end.at - before.at)
		to := span * (last - left) / (after.at - start.at)
		return hermite(left, right, from, to, share)
	}
	return limbSection{at: at, width: value(before.width, start.width, end.width, after.width),
		depth: value(before.depth, start.depth, end.depth, after.depth),
		bend:  value(before.bend, start.bend, end.bend, after.bend),
		centre: maths.V(value(before.centre.X, start.centre.X, end.centre.X, after.centre.X),
			value(before.centre.Y, start.centre.Y, end.centre.Y, after.centre.Y),
			value(before.centre.Z, start.centre.Z, end.centre.Z, after.centre.Z))}
}

func (anatomy humanAnatomy) arm(model *humanMesh, side humanSide, boundary skinBoundary) {
	stations := []limbSection{
		{0, .057, .055, 0, model.centre(boundary.rim)},
		{.13, .044, .045, .90, side.at(.211, 1.294, -.011)},
		{.24, .039, .041, 1.41, side.at(.216, 1.222, -.015)},
		{.37, .034, .036, 1.45, side.at(.223, 1.129, -.021)},
		{.48, .029, .029, 1.40, side.elbow()},
		{.56, .033, .035, 1.44, side.at(.239, 1.012, -.018)},
		{.70, .027, .030, 1.49, side.at(.248, .932, -.009)},
		{.87, .014, .024, 1.53, side.wrist()},
		{.91, .017, .030, 1.56, side.at(.254, .799, -.003)},
		{.955, .016, .036, 1.60, side.at(.258, .770, -.003)},
		{1, .012, .037, math.Pi / 2, side.at(.259, .744, -.003)},
	}
	end := model.connectedSweep(limbSweep{
		stations: stations, rows: 106, blendEnd: .235,
		site: proportionSite{part: armPart, side: side.sign},
		across: func(bend float64) maths.Vec {
			return maths.V(-math.Sin(bend), -side.sign*math.Cos(bend), 0)
		},
		forward: func(float64) maths.Vec { return maths.ZAxis },
		detail:  side.armDetail, weight: side.armWeight,
	}, boundary, .055)
	model.fan(end.rim, side.handDigits())
}

func (side humanSide) armDetail(point maths.Vec, at, sine, cosine float64) maths.Vec {
	point.Z += .006 * bell((at-.29)/.12) * maths.Smooth(.14, .9, cosine)
	point.Z -= .005 * bell((at-.48)/.035) * maths.Smooth(.22, .96, -cosine)
	return side.handDetail(point, at, sine, cosine)
}

func (side humanSide) handDetail(point maths.Vec, at, sine, cosine float64) maths.Vec {
	volar := maths.Smooth(.04, .88, side.sign*sine)
	dorsal := maths.Smooth(.15, .95, -side.sign*sine)
	radial := maths.Smooth(.42, .99, cosine)
	ulnar := maths.Smooth(.26, .99, -cosine)
	base := maths.Smooth(.873, .928, at)
	rim := maths.Smooth(.931, 1, at)
	// The first metacarpal diverges toward the high thumb root. The
	// transverse arch cups the two edges around the palmar hollow.
	thenar := .008 * bell((at-.943)/.047) * bell((cosine-.61)/.40)
	hypothenar := .005 * bell((at-.931)/.047) * bell((cosine+.72)/.37)
	hollow := .0027 * bell((at-.946)/.041) * bell((cosine+.07)/.36)
	point.X -= side.sign * (volar*(thenar+hypothenar-hollow) + base*(.007*radial+.005*ulnar))
	thumb := maths.Smooth(.877, 1, at) * radial
	point.Z += .005*thumb - .0025*bell((at-.866)/.022)*ulnar
	point.Y += .026*thumb + rim*(.009*ulnar-.004*bell((cosine+.03)/.31))
	// Metacarpal heads grade into the proximal phalanges. The extensor
	// relief stays low between the wrist and these dorsal prominences.
	heads := bell((cosine-.48)/.14) + bell((cosine-.02)/.15) +
		.81*bell((cosine+.45)/.13) + .62*bell((cosine+.82)/.10)
	point.X += side.sign * dorsal * heads *
		(.0022*bell((at-.990)/.036) + .0007*bell((at-.939)/.041))
	return point
}

func (side humanSide) armWeight(at float64, point maths.Vec, root jointWeights) jointWeights {
	switch {
	case at < .24:
		return root.towardsJoint(maths.Smooth(0, .24, at))
	case point.Y > .933:
		return blendWeight(side.name("upper-arm"), side.name("forearm"),
			1-maths.Smooth(1.017, 1.104, point.Y))
	default:
		return blendWeight(side.name("forearm"), side.name("hand"),
			1-maths.Smooth(.795, .854, point.Y))
	}
}

func (anatomy humanAnatomy) leg(model *humanMesh, side humanSide, boundary skinBoundary) {
	stations := []limbSection{
		{0, .087, .090, 0, side.at(.089, .787, -.010)},
		{.10, .081, .090, 0, side.at(.092, .720, -.015)},
		{.23, .070, .076, 0, side.at(.094, .629, -.015)},
		{.37, .052, .057, 0, side.at(.091, .530, .003)},
		{.48, .043, .043, 0, side.knee()},
		{.57, .044, .052, 0, side.at(.090, .375, -.024)},
		{.65, .048, .052, 0, side.at(.093, .303, -.025)},
		{.76, .034, .039, 0, side.at(.099, .193, -.013)},
		{.84, .026, .026, .06, side.at(.097, .110, -.009)},
		{.867, .027, .033, .28, side.at(.097, .082, -.011)},
		{.891, .032, .042, .65, side.at(.098, .054, -.010)},
		{.920, .035, .039, 1.03, side.at(.098, .039, .023)},
		{.949, .039, .031, 1.33, side.at(.100, .031, .065)},
		{.977, .045, .022, 1.52, side.at(.101, .024, .108)},
		{1, .046, .016, math.Pi / 2, side.at(.101, .020, .138)},
	}
	loft := limbSweep{
		rim: boundary.rim, stations: stations, rows: 126, blendEnd: .16,
		site:   proportionSite{part: legPart, side: side.sign},
		across: func(float64) maths.Vec { return maths.XAxis.Neg() },
		forward: func(bend float64) maths.Vec {
			return maths.V(0, math.Sin(bend), math.Cos(bend))
		},
		detail: side.legDetail,
		weight: side.legWeight,
	}
	var end []int
	if anatomy.spec.Anatomy == MaleAnatomy {
		loft.detail = func(point maths.Vec, at, sine, cosine float64) maths.Vec {
			return malePerineum(side.legDetail(point, at, sine, cosine))
		}
		end = model.sweep(loft)
	} else {
		end = model.femaleThighSweep(loft, boundary)
	}
	model.fan(turnRim(end, len(end)/4), side.footDigits())
}

func (side humanSide) legDetail(point maths.Vec, at, sine, cosine float64) maths.Vec {
	point.Z += .0095 * bell((at-.482)/.039) * bell(sine/.59) * maths.Smooth(.19, .90, cosine)
	point.Z -= .0065 * bell((at-.632)/.074) * maths.Smooth(.25, .91, -cosine)
	point.X -= side.sign * .0064 * bell((at-.565)/.070) * maths.Smooth(.40, .95, side.sign*sine)
	return side.footDetail(point, at, sine, cosine)
}

func (side humanSide) footDetail(point maths.Vec, at, sine, cosine float64) maths.Vec {
	medialSide := side.sign * sine
	medial := maths.Smooth(.03, .95, medialSide)
	lateral := maths.Smooth(.08, .95, -medialSide)
	plantar := maths.Smooth(.07, .95, -cosine)
	dorsal := maths.Smooth(.10, .94, cosine)
	foot := maths.Smooth(.883, .93, at)
	// The medial malleolus is higher than the lateral one. The Achilles
	// contour runs down the back of the ankle into the calcaneal heel.
	point.X += side.sign * (.0058*lateral*bell((point.Y-.071)/.016) -
		.0049*medial*bell((point.Y-.088)/.016))
	point.Z -= .004 * bell((at-.852)/.037) * plantar
	// The sole lifts and narrows between the heel pad and the heads of
	// the metatarsals. The medial column carries the higher instep.
	arch := foot * bell((point.Z-.049)/.039)
	point.X += side.sign * .010 * arch * medial * (1 - dorsal)
	point.Y += arch * (.012*medial*(1-dorsal) + .005*plantar)
	point.Y += .0028 * foot * dorsal * (.45 + medial) * bell((point.Z-.038)/.038)
	heel := maths.Smooth(.889, .922, at) * bell((point.Z+.016)/.035)
	ball := bell((point.Z - .119) / .027)
	point.Y -= plantar * (.009*heel + .004*ball)
	point.Y += .0038 * ball * plantar * (1 - maths.Smooth(.26, .86, math.Abs(sine)))
	// The metatarsal heads finish on an oblique line behind the toes.
	point.Z += maths.Smooth(.945, 1, at) * (.005*bell((medialSide-.32)/.58) - .013*lateral)
	point.Y = math.Max(0, point.Y)
	return point
}

func (side humanSide) legWeight(at float64, point maths.Vec, root jointWeights) jointWeights {
	switch {
	case at < .20:
		return blendWeight("pelvis", side.name("thigh"),
			mix(root.share, 1, maths.Smooth(0, .20, at)))
	case at < .72:
		return blendWeight(side.name("thigh"), side.name("shin"),
			1-maths.Smooth(.398, .502, point.Y))
	default:
		return blendWeight(side.name("shin"), side.name("foot"), maths.Smooth(.82, .947, at))
	}
}

package primates

import (
	"math"

	"github.com/janearc/slayer/maths"
)

type humanDigit struct {
	name             string
	joints           [4]maths.Vec
	width, pad, roll float64
	segments         int
}

type digitGroup struct {
	parent  string
	across  maths.Vec
	outward maths.Vec
	dorsal  float64
	taper   float64
	site    proportionSite
	cuts    [6]float64
	webs    [4]float64
	digits  [5]humanDigit
}

func (side humanSide) handDigits() digitGroup {
	at := side.at
	return digitGroup{
		parent: side.name("hand"), across: maths.ZAxis, outward: maths.YAxis.Neg(), dorsal: -side.sign,
		taper: .15, site: proportionSite{part: armPart, attachment: 1, side: side.sign},
		cuts: [6]float64{1, .64, .18, -.24, -.64, -1},
		webs: [4]float64{.003, .004, .004, .004},
		digits: [5]humanDigit{
			{side.name("thumb"), [4]maths.Vec{
				at(.251, .768, .034), at(.246, .742, .048), at(.240, .721, .054),
			}, .0088, .0080, side.sign * .37, 2},
			{side.name("index"), [4]maths.Vec{
				at(.259, .743, .015), at(.257, .713, .018), at(.252, .696, .019), at(.245, .679, .019),
			}, .0082, .0084, 0, 3},
			{side.name("middle"), [4]maths.Vec{
				at(.259, .739, -.004), at(.256, .705, -.003), at(.247, .685, -.003), at(.237, .669, -.002),
			}, .0084, .0086, 0, 3},
			{side.name("ring"), [4]maths.Vec{
				at(.256, .743, -.021), at(.252, .710, -.023), at(.244, .690, -.023), at(.234, .674, -.022),
			}, .0076, .0080, side.sign * -.06, 3},
			{side.name("little"), [4]maths.Vec{
				at(.252, .751, -.037), at(.247, .726, -.041), at(.239, .708, -.042), at(.230, .695, -.042),
			}, .0065, .0070, side.sign * -.12, 3},
		},
	}
}

func (side humanSide) footDigits() digitGroup {
	at := side.at
	toes := [5]humanDigit{
		{side.name("great-toe"), [4]maths.Vec{
			at(.064, .018, .139), at(.063, .016, .165), at(.062, .013, .187),
		}, .0120, .0118, 0, 2},
		{side.name("second-toe"), [4]maths.Vec{
			at(.086, .018, .143), at(.086, .017, .160), at(.085, .014, .171), at(.084, .011, .181),
		}, .0087, .0090, 0, 3},
		{side.name("third-toe"), [4]maths.Vec{
			at(.105, .017, .141), at(.105, .016, .154), at(.104, .013, .163), at(.103, .010, .171),
		}, .0081, .0088, 0, 3},
		{side.name("fourth-toe"), [4]maths.Vec{
			at(.122, .016, .135), at(.122, .015, .145), at(.121, .012, .153), at(.120, .010, .159),
		}, .0074, .0081, side.sign * -.08, 3},
		{side.name("little-toe"), [4]maths.Vec{
			at(.139, .015, .129), at(.140, .014, .137), at(.140, .011, .142), at(.139, .009, .147),
		}, .0066, .0077, side.sign * -.16, 3},
	}
	cuts := [6]float64{1, .49, .10, -.28, -.66, -1}
	webs := [4]float64{.0018, .0027, .0024, .0012}
	if side.sign < 0 {
		for i := 0; i < len(toes)/2; i++ {
			last := len(toes) - 1 - i
			toes[i], toes[last] = toes[last], toes[i]
		}
		webs = [4]float64{webs[3], webs[2], webs[1], webs[0]}
		original := cuts
		for i := range cuts {
			cuts[i] = -original[len(cuts)-1-i]
		}
	}
	return digitGroup{
		parent: side.name("foot"), across: maths.XAxis.Neg(), outward: maths.ZAxis, dorsal: -1,
		taper: .21, site: proportionSite{part: legPart, attachment: 1, side: side.sign},
		cuts: cuts, webs: webs, digits: toes,
	}
}

func (digit humanDigit) boneName(index int) string {
	return digit.name + [...]string{"-proximal", "-middle", "-distal"}[index]
}

func (digit humanDigit) segmentAt(index int) float64 {
	if digit.segments == 2 {
		return [...]float64{0, .56, 1}[index]
	}
	return [...]float64{0, .46, .75, 1}[index]
}

func (digit humanDigit) bones(group digitGroup) []humanBone {
	bones := make([]humanBone, digit.segments)
	parent := group.parent
	for i := range bones {
		name := digit.boneName(i)
		bones[i] = humanBone{name, parent, digit.joints[i], digit.joints[i+1], .55 * digit.width, group.site}
		parent = name
	}
	return bones
}

func (model *humanMesh) fan(rim []int, group digitGroup) {
	var columns [6]int
	var webs [6][5]int
	count := len(rim)
	for i, cut := range group.cuts {
		columns[i] = int(math.Round(math.Acos(cut) * float64(count) / (2 * math.Pi)))
		if i > 0 && i < len(group.cuts)-1 {
			first := model.vertices[rim[columns[i]]].point
			opposite := model.vertices[rim[count-columns[i]]].point
			for j := range webs[i] {
				share := float64(j+1) / float64(len(webs[i])+1)
				point := mixPoint(first, opposite, share).
					Add(group.outward.Scale(group.webs[i-1] * webArch(share)))
				webs[i][j] = model.limbPoint(point, fixedWeight(group.parent), group.site)
			}
		}
	}
	for i, digit := range group.digits {
		first, last := columns[i], columns[i+1]
		var boundary skinBoundary
		add := func(index int, inward maths.Vec) {
			boundary.rim = append(boundary.rim, index)
			boundary.inward = append(boundary.inward, inward)
		}
		for _, index := range rim[first : last+1] {
			add(index, group.outward)
		}
		if last < count/2 {
			for j, index := range webs[i+1] {
				add(index, group.webInward(j, len(webs[i+1]), 1))
			}
			add(rim[count-last], group.outward)
		}
		for column := count - last + 1; column < count-first; column++ {
			add(rim[column], group.outward)
		}
		if first > 0 {
			add(rim[count-first], group.outward)
			for j := len(webs[i]) - 1; j >= 0; j-- {
				add(webs[i][j], group.webInward(j, len(webs[i]), -1))
			}
		}
		model.growDigit(boundary, group, digit)
	}
}

func webArch(share float64) float64 {
	sine := math.Sin(math.Pi * share)
	return sine * sine
}

func (group digitGroup) webInward(sample, count int, side float64) maths.Vec {
	share := float64(sample+1) / float64(count+1)
	// The two branches leave the middle of a shared web in opposite
	// directions, then turn gradually down their respective digit axes.
	return mixPoint(group.outward, group.across.Scale(side), webArch(share)).Unit()
}

type digitAxes struct {
	centre, across, up maths.Vec
	dorsal, taper      float64
}

func (digit humanDigit) axes(at float64, group digitGroup) digitAxes {
	segment := 0
	for segment+1 < digit.segments && at > digit.segmentAt(segment+1) {
		segment++
	}
	start, end := digit.segmentAt(segment), digit.segmentAt(segment+1)
	before, after := max(0, segment-1), min(digit.segments, segment+2)
	first, last := digit.joints[segment], digit.joints[segment+1]
	span := end - start
	from := digit.joints[segment+1].Sub(digit.joints[before]).Scale(span / (end - digit.segmentAt(before)))
	to := digit.joints[after].Sub(digit.joints[segment]).Scale(span / (digit.segmentAt(after) - start))
	// The cubic follows the phalanx landmarks and carries a continuous
	// cross-section direction through the interphalangeal joints.
	firstControl := first.Add(from.Scale(1.0 / 3))
	lastControl := last.Sub(to.Scale(1.0 / 3))
	share := (at - start) / span
	left := mixPoint(mixPoint(first, firstControl, share), mixPoint(firstControl, lastControl, share), share)
	right := mixPoint(mixPoint(firstControl, lastControl, share), mixPoint(lastControl, last, share), share)
	direction := right.Sub(left).Unit()
	across := group.across.Sub(direction.Scale(group.across.Dot(direction))).Unit()
	up := direction.Cross(across)
	sine, cosine := math.Sincos(digit.roll)
	return digitAxes{mixPoint(left, right, share), across.Scale(cosine).Add(up.Scale(sine)),
		up.Scale(cosine).Sub(across.Scale(sine)), group.dorsal, group.taper}
}

const digitJoinAt = .34

// digitHairPatch identifies the proximal branches and their anatomical
// dorsum before skinning. Face indices survive the vertex compaction.
type digitHairPatch struct {
	first, end            int
	origin, along, dorsal maths.Vec
	hand                  bool
}

func (model *humanMesh) growDigit(boundary skinBoundary, group digitGroup, digit humanDigit) {
	rim := boundary.rim
	axes := digit.axes(0, group)
	centre := model.centre(rim)
	angles := model.angles(rim, centre, axes.up, axes.across, digit.pad, digit.width)
	joins := model.digitJoins(boundary, group, digit, angles)
	const rows = 30
	const hairLastRow = 19
	dorsum := digit.axes(.36, group)
	span := digit.axes(float64(hairLastRow)/rows, group).centre.Sub(axes.centre)
	patch := digitHairPatch{first: len(model.faces), origin: axes.centre,
		along: span.Scale(1 / span.Dot(span)), dorsal: dorsum.up.Scale(group.dorsal),
		hand: group.site.part == armPart}
	previous := rim
	for row := 1; row < rows; row++ {
		at := float64(row) / rows
		axes := digit.axes(at, group)
		round := maths.Smooth(digitJoinAt, .77, at)
		weights := digit.weight(group.parent, at)
		ring := make([]int, len(rim))
		for i := range ring {
			var point maths.Vec
			if at < digitJoinAt {
				point = joins[i].at(at / digitJoinAt)
			} else {
				point = digit.point(at, angles.angle(i, round), axes)
			}
			ring[i] = model.limbPoint(point, weights, group.site)
		}
		model.connect(previous, ring)
		if row == hairLastRow {
			patch.end = len(model.faces)
		}
		previous = ring
	}
	model.digitHair = append(model.digitHair, patch)
	tip := model.limbPoint(digit.joints[digit.segments], fixedWeight(digit.boneName(digit.segments-1)), group.site)
	model.cap(previous, tip)
}

func (model *humanMesh) digitJoins(boundary skinBoundary, group digitGroup, digit humanDigit, angles rimAngles) []skinJoin {
	const step = .0005
	before := digit.axes(digitJoinAt-step, group)
	after := digit.axes(digitJoinAt+step, group)
	targetAxes := digit.axes(digitJoinAt, group)
	joins := make([]skinJoin, len(boundary.rim))
	for i, index := range boundary.rim {
		angle := angles.near[i]
		root := model.vertices[index].point
		target := digit.point(digitJoinAt, angle, targetAxes)
		travel := target.Sub(root).Len()
		start := boundary.inward[i].Scale(.74 * travel)
		end := digit.point(digitJoinAt+step, angle, after).Sub(digit.point(digitJoinAt-step, angle, before)).
			Scale(digitJoinAt / (2 * step))
		joins[i] = skinJoin{root, start, target, end}
	}
	return joins
}

func (digit humanDigit) point(at, angle float64, axes digitAxes) maths.Vec {
	sine, cosine := math.Sincos(angle)
	dorsal := sine * axes.dorsal
	knuckles := .42 * bell(at/.11)
	crease := 0.0
	for joint := 1; joint < digit.segments; joint++ {
		distance := at - digit.segmentAt(joint)
		knuckles += bell(distance/.072) / float64(joint)
		crease += bell(distance / .029)
	}
	// Volar pads sit between flexion creases. The dorsal joint surfaces
	// remain convex and the distal pad rounds into an ellipsoidal tip.
	volar := maths.Smooth(.12, .96, -dorsal)
	end := math.Max(0, (at-.77)/.23)
	round := math.Sqrt(math.Max(0, 1-end*end))
	width := (1 - (axes.taper+.025)*at + .08*knuckles) * round
	pad := (1 - axes.taper*at + .15*volar + .14*knuckles*math.Max(0, dorsal) - .11*crease*volar) * round
	plate := .00046 * nailProfile(at, dorsal)
	return axes.centre.
		Add(axes.up.Scale(digit.pad*sine*pad + plate*axes.dorsal)).
		Add(axes.across.Scale(digit.width * cosine * width))
}

// nailProfile places a shallow plate beneath a rounded cuticle and
// tapers it into the existing dorsal surface at the tip.
func nailProfile(at, dorsal float64) float64 {
	cuticle := .665 + .12*(1-dorsal)
	return maths.Smooth(cuticle, cuticle+.043, at) *
		(1 - maths.Smooth(.915, .954, at)) * maths.Smooth(.59, .78, dorsal)
}

func (digit humanDigit) nailAt(point maths.Vec, group digitGroup) float64 {
	last := digit.segments - 1
	base := digit.joints[last]
	line := digit.joints[last+1].Sub(base)
	share := point.Sub(base).Dot(line) / line.Dot(line)
	at := mix(digit.segmentAt(last), 1, share)
	if at < .64 || at > .955 {
		return 0
	}
	axes := digit.axes(at, group)
	delta := point.Sub(axes.centre)
	up := delta.Dot(axes.up) / digit.pad
	across := delta.Dot(axes.across) / digit.width
	radius := math.Hypot(up, across)
	if radius < .25 || radius > 1.30 {
		return 0
	}
	return nailProfile(at, up*axes.dorsal/radius)
}

func (digit humanDigit) weight(parent string, at float64) jointWeights {
	for i := 0; i < digit.segments; i++ {
		joint := digit.segmentAt(i)
		if at < joint+.13 {
			return blendWeight(parent, digit.boneName(i), maths.Smooth(joint-.13, joint+.13, at))
		}
		parent = digit.boneName(i)
	}
	return fixedWeight(parent)
}

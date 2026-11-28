package primates

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// VulvarForm varies outer and inner labial fullness independently. Zero
// fullness uses the authored reference ratio of 1. Asymmetry changes the
// relative fullness of the two sides, with positive values favouring +X.
type VulvarForm struct {
	OuterFullness, InnerFullness, Asymmetry float64
}

func (form VulvarForm) resolved() (VulvarForm, error) {
	if form.OuterFullness == 0 {
		form.OuterFullness = 1
	}
	if form.InnerFullness == 0 {
		form.InnerFullness = 1
	}
	if !between(form.OuterFullness, .72, 1.32) ||
		!between(form.InnerFullness, .65, 1.50) ||
		!between(form.Asymmetry, -.25, .25) {
		return VulvarForm{}, fmt.Errorf("adult human: vulvar form outside authored range")
	}
	return form, nil
}

func femalePubicAngle(angle, y float64) float64 {
	if y >= .935 {
		return angle
	}
	fromFront := math.Remainder(angle, 2*math.Pi)
	support := (1 - maths.Smooth(0, .30, math.Abs(fromFront))) *
		(1 - maths.Smooth(.852, .935, y))
	return angle - .40*fromFront*support
}

const femaleCuffRows = 28

// femalePelvicCuff supplies the common lower pelvic rows. The thigh roots,
// cuff and lower trunk are later shaped together as one surface.
func (model *humanMesh) femalePelvicCuff(root [2][]int) [2][]int {
	var joins [humanAround]skinJoin
	for i, index := range root[0] {
		point := model.vertices[index].point
		above := model.vertices[root[1][i]].point
		angle := 2 * math.Pi * float64(i) / humanAround
		front := maths.Smooth(.51, .99, math.Cos(angle))
		mons := front * bell(point.X/.071)
		drop := .021 + .027*mons
		lower := maths.V(point.X*(1-.024*(1-mons)), point.Y-drop, point.Z-.010*front)
		incoming := point.Sub(above)
		tangent := incoming.Scale(-drop / incoming.Y)
		leaving := maths.V(1.3*(lower.X-point.X), -drop, -.016*front)
		joins[i] = skinJoin{point, tangent, lower, leaving}
	}
	previous, rim := root[1], root[0]
	for row := 1; row <= femaleCuffRows; row++ {
		share := float64(row) / femaleCuffRows
		next := make([]int, len(rim))
		for i, join := range joins {
			point := join.at(share)
			source := model.vertices[root[0][i]]
			chart := source.chart.Add(point.Sub(source.point))
			next[i] = model.add(humanVertex{point: point, chart: mixPoint(chart, point, share),
				weights: source.weights})
		}
		model.connect(next, rim)
		previous, rim = rim, next
	}
	return [2][]int{rim, previous}
}

// femaleSaddle follows the underside of the pubis before curving back
// into the perineum. Its anterior tangent follows the trunk skin.
func femaleSaddle(front, back maths.Vec, share float64) maths.Vec {
	point := mixPoint(front, back, maths.Smooth(0, 1, share))
	point.Y -= .013 * math.Sin(math.Pi*share)
	return point
}

func (model *humanMesh) thighBoundary(trunk [2][]int, bridge []int, side humanSide) skinBoundary {
	const half = humanAround / 2
	var boundary skinBoundary
	lateral := side.at(1, 0, 0)
	for i := 0; i <= half; i++ {
		column := half - i
		if side.sign < 0 {
			column = (humanAround - i) % humanAround
		}
		index := trunk[0][column]
		root := model.vertices[index].point
		above := model.vertices[trunk[1][column]].point
		down := root.Sub(above).Unit()
		saddle := 1 - maths.Smooth(.002, .034, math.Abs(root.X))
		boundary.append(index, mixPoint(down, lateral, saddle))
	}
	for i := range bridge {
		vertex := i
		if side.sign < 0 {
			vertex = len(bridge) - i - 1
		}
		boundary.append(bridge[vertex], lateral)
	}
	return boundary
}

type vulvarRelief struct{ outer, inner float64 }

// relief gives the outer pads and the underside structures along the
// common pelvic arc. The vestibular recesses retain a continuous floor.
func (form VulvarForm) relief(x, course, anal float64) vulvarRelief {
	width := 1 - maths.Smooth(.049, .072, math.Abs(x))
	innerWindow := width * maths.Smooth(-.019, .004, course) *
		(1 - maths.Smooth(anal+.021, anal+.045, course))
	folds := form.labialFolds(x, course)
	hoodSpan := bell((course - .029) / .013)
	hood := .0022 * hoodSpan * bell((math.Abs(x)-.0044)/.0050)
	glans := .0018 * bell(x/.0035) * bell((course-.040)/.0068)
	// Broad, shallow recesses give the closed vestibule anatomical scale.
	urethra := .0007 * bell(x/.0028) * bell((course-.052)/.0049)
	introitus := .0014 * bell(x/.0046) * bell((course-.074)/.010)
	perineum := .0015 * bell(x/.0095) * bell((course-.109)/.022)
	analRelief := -.0024 * bell(x/.0064) * bell((course-anal)/.0090)
	return vulvarRelief{
		outer: width * folds.outer,
		inner: innerWindow * (folds.inner + hood + glans + perineum + analRelief - urethra - introitus),
	}
}

func (form VulvarForm) labialFolds(x, course float64) vulvarRelief {
	innerLength := maths.Smooth(.027, .046, course) * (1 - maths.Smooth(.078, .097, course))
	innerAxis := .0039 + .0015*bell((course-.058)/.023)
	var outer, inner [2]float64
	for i, side := range humanSides {
		asymmetry, across := form.Asymmetry*side.sign, x*side.sign
		outer[i] = form.outerPad(across, course, asymmetry)
		inner[i] = .0038 * form.InnerFullness * (1 + asymmetry) *
			bell((across-innerAxis)/.0037)
	}
	// Neighboring outer folds nearly meet along a rounded skin seam.
	// Inner relief is more visible where the outer pads are less full.
	margin := 1 - maths.Smooth(.009, .019, math.Abs(x))
	contact := mix(.0042, .0020, maths.Smooth(.008, .083, course)) *
		maths.Smooth(0, .006, outer[0]+outer[1])
	envelope := roundedContact(outer[0], outer[1], contact)
	innerRelief := roundedContact(inner[0], inner[1], .0004) * innerLength * margin
	covered := maths.Smooth(.003, .0131, envelope) * maths.Smooth(.75, 1.15, form.OuterFullness)
	return vulvarRelief{
		outer: envelope,
		inner: innerRelief * (1 - .58*covered),
	}
}

// outerPad gives one oblique footprint from the mons to the groin. The
// lateral tissue sits higher in the chart. The same rounded field defines
// the raised region and its attachment, with zero slope at its perimeter.
func (form VulvarForm) outerPad(across, course, asymmetry float64) float64 {
	lower := maths.Smooth(-.006, .081, course)
	axis := mix(.024, .0108, lower)
	width := mix(.051, .038, lower)
	across -= axis
	long := course - .048 + .53*across
	span := mix(.102, .079, maths.Smooth(-.017, .018, long))
	radiusSquared := across*across/(width*width) + long*long/(span*span)
	root := math.Max(0, 1-radiusSquared)
	return .0131 * form.OuterFullness * (1 + .70*asymmetry) * root * root * root
}

func roundedContact(left, right, radius float64) float64 {
	gap := math.Abs(left - right)
	crest := math.Max(left, right)
	if gap >= radius {
		return crest
	}
	overlap := 1 - gap/radius
	return crest + .25*radius*overlap*overlap
}

func femaleAdductor(point maths.Vec, side float64) maths.Vec {
	across := side * point.X
	if across <= 0 || across >= .055 || point.Y <= .690 || point.Y >= .783 {
		return point
	}
	medial := 1 - maths.Smooth(.018, .055, across)
	level := maths.Smooth(.690, .738, point.Y) * (1 - maths.Smooth(.751, .783, point.Y))
	span := .0064 * medial * level * bell((point.Z-.027)/.081)
	// The medial thigh stays on its side of the saddle and retains a
	// separate contour below it. The rounded labial contour lies forward.
	limit := .59 * across
	point.X -= side * limit * (-math.Expm1(-span / limit))
	return point
}

func (model *humanMesh) shapeFemaleAdductors(first int) {
	for i := first; i < len(model.vertices); i++ {
		vertex := &model.vertices[i]
		if vertex.site.part == legPart {
			vertex.point = femaleAdductor(vertex.point, vertex.site.side)
			vertex.chart = vertex.point
		}
	}
}

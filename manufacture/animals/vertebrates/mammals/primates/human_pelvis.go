package primates

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The opening wraps across the trunk chart's anterior midline. Its low edge
// is shared with the two thigh roots. It opens into a common skin apron
// before dividing into two branches.
const (
	pubisHalfColumns = 11
	pubisTopRow      = 14
)

func malePubicOpening(column, lowerRow int) bool {
	return lowerRow < pubisTopRow &&
		(column < pubisHalfColumns || column >= humanAround-pubisHalfColumns)
}

func pubicChart(angle, y, top float64) (float64, float64) {
	fromFront := math.Remainder(angle, 2*math.Pi)
	half := 2 * math.Pi * pubisHalfColumns / humanAround
	across := math.Abs(fromFront) / half
	height := (y - pelvisY) / (top - pelvisY)
	radius := math.Max(across, height)
	if height <= 0 || radius >= 1.76 {
		return angle, y
	}
	weight := (1 - maths.Smooth(1, 1.76, radius)) *
		maths.Smooth(0, 1, height) * maths.Smooth(.20, 1.14, across)
	return angle - .16*fromFront*weight, y - .012*weight
}

func wrappedPubisColumn(column int) int {
	return (column + humanAround) % humanAround
}

func (model *humanMesh) pubicBoundary(trunk [][]int) skinBoundary {
	count := 4*pubisHalfColumns + 2*pubisTopRow
	boundary := skinBoundary{rim: make([]int, 0, count), inward: make([]maths.Vec, 0, count)}
	add := func(column, row int) {
		index := trunk[row][wrappedPubisColumn(column)]
		root := model.vertices[index].point
		var inward maths.Vec
		switch column {
		case -pubisHalfColumns:
			next := trunk[row][wrappedPubisColumn(column-1)]
			inward = root.Sub(model.vertices[next].point).Unit()
		case pubisHalfColumns:
			next := trunk[row][wrappedPubisColumn(column+1)]
			inward = root.Sub(model.vertices[next].point).Unit()
		}
		if row == 0 {
			inward = inward.Add(maths.YAxis)
		}
		if row == pubisTopRow {
			outside := trunk[row+1][wrappedPubisColumn(column)]
			inward = inward.Add(root.Sub(model.vertices[outside].point).Unit())
		}
		boundary.rim = append(boundary.rim, index)
		boundary.inward = append(boundary.inward, inward)
	}
	for column := -pubisHalfColumns; column < pubisHalfColumns; column++ {
		add(column, 0)
	}
	for row := 0; row < pubisTopRow; row++ {
		add(pubisHalfColumns, row)
	}
	for column := pubisHalfColumns; column > -pubisHalfColumns; column-- {
		add(column, pubisTopRow)
	}
	for row := pubisTopRow; row > 0; row-- {
		add(-pubisHalfColumns, row)
	}
	return boundary
}

func (model *humanMesh) maleGroin(trunk [][]int) {
	boundary := model.pubicBoundary(trunk)
	stations := []limbSection{
		{0, .038, .033, 0, model.centre(boundary.rim)},
		{.67, .033, .029, 0, maths.V(0, .829, .121)},
		{1, .028, .026, 0, maths.V(0, .825, .127)},
	}
	apron := model.connectedSweep(pubicLoft(stations, 22, .64, pubicApronRelief), boundary, .023)
	upper, lower := model.splitPubicApron(apron)
	model.shaft(upper)
	model.scrotum(lower)
}

func pubicApronRelief(point maths.Vec, at, sine, cosine float64) maths.Vec {
	fade := maths.Smooth(0, .85, at)
	point.Z += .0034 * fade * bell(sine/.57) * (.75 + .25*cosine)
	return point
}

// splitPubicApron runs one transverse arc across the end of the common
// skin. Each branch traverses the same new web in the opposing direction.
func (model *humanMesh) splitPubicApron(apron skinBoundary) (skinBoundary, skinBoundary) {
	rim := apron.rim
	quarter := len(rim) / 4
	positive, negative := quarter, 3*quarter
	var upper, lower skinBoundary
	for i := positive; i <= negative; i++ {
		upper.append(apron.rim[i], apron.inward[i])
	}
	for i := negative; i <= len(rim)+positive; i++ {
		index := i % len(rim)
		lower.append(apron.rim[index], apron.inward[index])
	}
	middle := make([]int, quarter-1)
	negativePoint := model.vertices[rim[negative]].point
	positivePoint := model.vertices[rim[positive]].point
	for i := range middle {
		fraction := float64(i+1) / float64(quarter)
		point := mixPoint(negativePoint, positivePoint, fraction)
		arch := math.Sin(math.Pi * fraction)
		point.Y += .0043 * arch
		point.Z += .0040 * arch
		middle[i] = model.point(point, fixedWeight("pelvis"))
		upper.append(middle[i], maths.YAxis)
	}
	for i := len(middle) - 1; i >= 0; i-- {
		lower.append(middle[i], maths.YAxis.Neg())
	}
	return upper, lower
}

func (boundary *skinBoundary) append(index int, inward maths.Vec) {
	boundary.rim = append(boundary.rim, index)
	boundary.inward = append(boundary.inward, inward)
}

func pubicLoft(stations []limbSection, rows int, join float64,
	detail func(maths.Vec, float64, float64, float64) maths.Vec,
) limbSweep {
	return limbSweep{
		stations: stations, rows: rows, blendEnd: join,
		across: func(float64) maths.Vec { return maths.XAxis },
		forward: func(bend float64) maths.Vec {
			sine, cosine := math.Sincos(bend)
			return maths.V(0, -cosine, -sine)
		},
		detail: detail,
		weight: func(at float64, _ maths.Vec, root jointWeights) jointWeights {
			return blendWeight("pelvis", root.joint, root.share*(1-maths.Smooth(0, .32, at)))
		},
	}
}

func (model *humanMesh) shaft(boundary skinBoundary) {
	// Neutral longitudinal sections continue from the pubic insertion
	// through the shaft, coronal sulcus, glans and recessed meatal rim.
	stations := []limbSection{
		{0, .0290, .0180, .05, model.centre(boundary.rim)},
		{.14, .0180, .0160, .40, maths.V(.000, .836, .138)},
		{.33, .0143, .0137, 1.04, maths.V(.002, .818, .151)},
		{.52, .0130, .0124, 1.36, maths.V(.003, .801, .160)},
		{.67, .0123, .0118, 1.46, maths.V(.004, .788, .162)},
		{.73, .0119, .0114, 1.51, maths.V(.004, .781, .163)},
		{.79, .0150, .0153, 1.52, maths.V(.005, .777, .163)},
		{.86, .0136, .0136, 1.53, maths.V(.005, .771, .163)},
		{.93, .0102, .0107, 1.55, maths.V(.006, .766, .163)},
		{.98, .0051, .0064, 1.55, maths.V(.006, .761, .162)},
		{1, .0009, .0030, 1.55, maths.V(.006, .760, .162)},
	}
	end := model.connectedSweep(pubicLoft(stations, 54, .31, shaftRelief), boundary, .022)
	// The cavity ends just behind its lip, with no open internal canal.
	model.cap(end.rim, model.point(maths.V(.006, .762, .162), fixedWeight("pelvis")))
}

func shaftRelief(point maths.Vec, at, sine, cosine float64) maths.Vec {
	dorsal := maths.Smooth(.13, .96, -cosine)
	corona := bell((at - .788) / .036)
	point.Z += .0019 * dorsal * corona
	point.Y -= .0012 * dorsal * corona
	ventralFold := maths.Smooth(.45, .96, cosine) * bell(sine/.28)
	point.Z -= .0014 * ventralFold * bell((at-.813)/.078)
	return point
}

func (model *humanMesh) scrotum(boundary skinBoundary) {
	stations := []limbSection{
		{0, .033, .016, .13, model.centre(boundary.rim)},
		{.22, .031, .020, .81, maths.V(.000, .801, .129)},
		{.42, .033, .023, 1.39, maths.V(-.002, .778, .120)},
		{.64, .030, .022, 1.60, maths.V(-.002, .753, .110)},
		{.82, .026, .018, 1.62, maths.V(-.002, .738, .103)},
		{.93, .018, .013, 1.64, maths.V(-.002, .731, .101)},
		{1, .007, .006, 1.65, maths.V(-.003, .728, .100)},
	}
	end := model.connectedSweep(pubicLoft(stations, 54, .31, scrotalRelief), boundary, .020)
	model.cap(end.rim, model.point(maths.V(-.003, .726, .100), fixedWeight("pelvis")))
}

func scrotalRelief(point maths.Vec, at, sine, cosine float64) maths.Vec {
	body := bell((at - .71) / .27)
	lobe := bell((math.Abs(sine) - .44) / .31)
	point.Y -= (.0052 - .0010*sine) * body * lobe
	point.Z += .0055 * body * lobe * maths.Smooth(.05, .92, -cosine)
	raphe := bell(sine/.15) * body
	point.Z += .0010 * raphe * maths.Smooth(.10, .89, -cosine)
	return point
}

func malePerineum(point maths.Vec) maths.Vec {
	x, y, z := point.X, point.Y, point.Z
	window := maths.Smooth(.721, .743, y) * (1 - maths.Smooth(.787, .817, y))
	raphe := .00135 * bell(x/.0040) * bell((z-.009)/.046)
	analDepression := .0034 * bell(x/.0070) * bell((z+.056)/.0079)
	point.Y += window * (analDepression - raphe)
	return point
}

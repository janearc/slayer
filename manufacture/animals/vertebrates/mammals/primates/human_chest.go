package primates

import (
	"math"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

type tissueMotion struct {
	gravity, shoulder float64
	side              int
}

func (anatomy humanAnatomy) chest(angle, y float64) (maths.Vec, maths.Vec, tissueMotion) {
	wall, chart := anatomy.torso.wall(angle, y)
	if y < 1.022 || y > 1.36 {
		return wall, chart, tissueMotion{}
	}
	if anatomy.spec.Anatomy == MaleAnatomy {
		if y < 1.055 {
			return wall, chart, tissueMotion{}
		}
		return anatomy.pectoral(wall, chart, angle)
	}
	angle = math.Remainder(angle, 2*math.Pi)
	wall, motion := anatomy.breastWall(wall, chart, angle)
	side, _, sign := anatomy.torso.side(chart.X)
	form := anatomy.spec.Breasts.side(sign)
	cx, cy := anatomy.torso.landmarks(side)
	front := maths.Smooth(-.15, .67, math.Cos(angle))
	height, papilla := anatomy.areolarHeight(side, form, chart, cx+form.RootOutward, cy, sign, front)
	if height > 0 {
		// Place areolar relief relative to the local tissue, including the
		// rounded pole, rather than to an analytical chest-wall normal.
		const step = .0002
		angleBefore, angleAfter := anatomy.breastAt(angle-step, y), anatomy.breastAt(angle+step, y)
		lower, upper := anatomy.breastAt(angle, y-step), anatomy.breastAt(angle, y+step)
		across := angleAfter.Sub(angleBefore)
		normal := across.Cross(upper.Sub(lower)).Unit()
		relief := normal.Scale(height)
		if form.PapillaOutwardTilt != 0 || form.PapillaUpTilt != 0 {
			relief = relief.Add(form.papillaLean(side, across, normal, sign, papilla))
		}
		wall = wall.Add(relief)
	}
	return wall, chart, motion
}

func (anatomy humanAnatomy) breastAt(angle, y float64) maths.Vec {
	wall, chart := anatomy.torso.wall(angle, y)
	point, _ := anatomy.breastWall(wall, chart, angle)
	return point
}

func (anatomy humanAnatomy) breastWall(wall, chart maths.Vec, angle float64) (maths.Vec, tissueMotion) {
	sine, cosine := math.Sincos(angle)
	if cosine <= -.10 {
		return wall, tissueMotion{}
	}
	window := maths.Smooth(-.10, .34, cosine) * maths.Smooth(1.022, 1.073, chart.Y) *
		(1 - maths.Smooth(1.312, 1.36, chart.Y))
	side, _, sign := anatomy.torso.side(chart.X)
	x, y := math.Abs(chart.X), chart.Y
	form := anatomy.spec.Breasts.side(sign)
	field := anatomy.breastField(side, form, sign*angle, x, y)

	// The anterior soft tissue obscures part of the costal margin.
	// Preserve the rib contour where it emerges into the flank.
	wall.Z -= window * field.costalCover * anatomy.torso.costalMargin(x, y, maths.Smooth(-.13, .62, cosine))

	// Keep the sternal surface on the rib cage between the two medial roots.
	// The shallow longitudinal ridge joins the clavicular and costal wall.
	sternum := .0027 * bell(chart.X/.029) * bell((chart.Y-1.219)/.074)
	wall.Z += window * maths.Smooth(.58, .97, cosine) * sternum
	outward := maths.V(.44*sine, 0, .90*cosine+.10).Unit()
	wall = wall.Add(outward.Scale(window * field.projection))
	wall.Y -= window * field.drop

	index := 0
	if sign < 0 {
		index = 1
	}
	// In the pose pass, keep the same downward displacement in world
	// space as the thorax rotates. The axillary tail tracks its own arm.
	return wall, tissueMotion{gravity: window * field.drop, shoulder: window * field.shoulder, side: index}
}

type breastField struct{ projection, drop, shoulder, costalCover float64 }

// Projection and mobility are measured at the same sagittal stations, 16 mm
// apart before scaling. The lower tissue projects farther than the surface
// above the centre. Mobility peaks there and falls to zero at the attachment.
var breastMeridian = [...][2]float64{
	{0, 0}, {.003, 0}, {.034, .014}, {.171, .10}, {.456, .41}, {.782, .73},
	{.96, 1}, {1, .92}, {.95, .70}, {.81, .41}, {.60, .15}, {.36, .036},
	{.155, 0}, {.056, 0}, {.016, 0}, {.003, 0}, {0, 0}, {0, 0},
}

// A cubic B-spline gives smooth tangents through the lower turn. Moving Y
// independently of projection rounds the loaded pole over its fixed root.
func breastProfile(height float64) (projection, mobility float64) {
	row := (height + .112) / .016
	index := int(math.Floor(row))
	share := row - float64(index)
	square := share * share
	cube := share * square
	other := 1 - share
	weights := [4]float64{
		other * other * other / 6,
		(4 - 6*square + 3*cube) / 6,
		(1 + 3*share + 3*square - 3*cube) / 6,
		cube / 6,
	}
	for i, weight := range weights {
		station := breastMeridian[max(0, min(len(breastMeridian)-1, index+i-1))]
		projection += weight * station[0]
		mobility += weight * station[1]
	}
	return
}

func (anatomy humanAnatomy) breastField(side Side, form BreastSideForm, angle, x, y float64) breastField {
	cx, cy := anatomy.torso.landmarks(side)
	cx += form.RootOutward
	height := y - cy
	upper := maths.Smooth(-.029, .085, height)
	ringWidth := sectionAt(y).w * anatomy.torso.Width
	centre := math.Asin((cx + .013*upper) / ringWidth)
	arc := (angle - centre) * ringWidth

	// Let the inner curve return to the sternum over its full available
	// arc. The outer curve follows the rib cage toward the axilla.
	medialWidth := min(side.Width*(1.07+.12*upper), (centre-.064)*ringWidth)
	lateralWidth := side.Width * (1.20 + .18*upper)
	width := mix(medialWidth, lateralWidth, maths.Smooth(-.040, .035, arc))
	across := arc / width
	medial := maths.Smooth(.005, .030, x)
	front := 1 - maths.Smooth(1.03, 1.46, angle)
	cross := math.Pow(max(0, 1-across*across), 2.25) * front
	arch := .015*(1-bell(across/.74)) + .004*math.Tanh(across)
	meridianHeight := height - (1-upper)*arch
	span := .85 + 1.65*side.Width + .90*side.Projection
	span *= 1 + .28*form.LowerPoleLength*(1-maths.Smooth(-.038, 0, meridianHeight))
	projection, mobility := breastProfile(meridianHeight / span)
	upperFullness := 1 + .52*form.UpperFullness*maths.Smooth(-.008, .068, meridianHeight)
	lowerFullness := 1 + .46*form.LowerFullness*(1-maths.Smooth(-.058, .010, meridianHeight))
	pole := cross * projection * upperFullness * lowerFullness

	// The upper tissue runs diagonally into the pectoral fascia. Its
	// lateral tail converges with the anterior axillary fold near the arm.
	bed := bell((arc-.015)/(1.18*side.Width)) * bell((height-.059)/.068) *
		maths.Smooth(-.041, .031, height) * medial
	tailHeight := cy + .067 + .045*(angle-1.10)
	tail := bell((angle-1.10)/.36) * bell((y-tailHeight)/.050) * medial
	foldHeight := 1.281 + .057*(angle-1.24)
	fold := bell((angle-1.24)/.27) * bell((y-foldHeight)/.042) * medial

	load := side.Projection * anatomy.torso.Gravity * (.066 + .30*anatomy.torso.Compliance)
	load *= 1 + .21*(anatomy.torso.AgeYears-38)/42
	// Limit the lower-pole travel smoothly at the high-load end so the
	// rows approaching the fixed inferior root stay ordered.
	const travelLimit = .025
	drop := travelLimit * math.Tanh(load/travelLimit) * cross * mobility
	tissue := maths.Smooth(0, .040, side.Projection)
	pull := .010*tail*tissue + .005*fold + .005*pole*upper*maths.Smooth(.069, .132, x)*tissue
	return breastField{
		projection:  side.Projection*(.97*pole+.12*bed*upperFullness+.13*tail) + .0069*fold,
		drop:        drop,
		shoulder:    pull,
		costalCover: .82 * bell((x-cx)/(.92*side.Width)) * tissue,
	}
}

// pectoral carries the clavicular and sternocostal heads over the rib cage,
// then narrows the lateral mass toward its upper-arm attachment.
func (anatomy humanAnatomy) pectoral(wall, chart maths.Vec, angle float64) (maths.Vec, maths.Vec, tissueMotion) {
	side, _, sign := anatomy.torso.side(chart.X)
	x, y := math.Abs(chart.X), chart.Y
	cx, cy := anatomy.torso.landmarks(side)
	sine, cosine := math.Sincos(angle)
	front := maths.Smooth(-.10, .76, cosine)
	root := front * maths.Smooth(.009, .043, x) * (1 - maths.Smooth(1.328, 1.359, y))
	abdominalEdge := cy - .037 + .21*(x-cx)
	sternoCostal := maths.Smooth(abdominalEdge, abdominalEdge+.034, y) *
		(1 - maths.Smooth(1.248, 1.319, y)) * bell((x-cx)/(.77*side.Width))
	clavicular := bell((x-cx-.013)/(.89*side.Width)) * bell((y-(1.312-.26*x))/.041)
	axillaryFold := bell((x-.141)/.028) * bell((y-1.249-.23*(x-.141))/.046)
	projection := side.Projection * (.40*sternoCostal + .17*clavicular + .24*axillaryFold) * root
	normal := maths.V(.51*sine, 0, .92*cosine+.08).Unit()
	relief, _ := anatomy.areolarHeight(side, BreastSideForm{}, chart, cx, cy, sign, front)
	wall = wall.Add(normal.Scale(projection + relief))
	index := 0
	if sign < 0 {
		index = 1
	}
	return wall, chart, tissueMotion{shoulder: .013 * root * axillaryFold, side: index}
}

func (anatomy humanAnatomy) areolarHeight(side Side, form BreastSideForm, chart maths.Vec,
	cx, cy, sign, front float64,
) (height, papillaHeight float64) {
	u, v := form.areolarChart(side, chart, cx, cy)
	radius := math.Hypot(u, v)
	if radius >= 1.75 || front == 0 || form.WithoutAreola {
		return 0, 0
	}
	papilla, tubercle := anatomy.torso.relief(side, u, v, sign)
	taper := front * (1 - maths.Smooth(1.43, 1.75, radius))
	taper *= maths.Smooth(0, .018, math.Abs(chart.X))
	return taper * (.0016*bell(radius/.77) + side.PapillaProjection*papilla + .00063*tubercle),
		taper * side.PapillaProjection * papilla
}

func (human *Human) moveTissue(pose *skeleton.SkinPose, points []maths.Vec) {
	thorax, _ := pose.Frame("thorax")
	down := maths.YAxis.Neg()
	gravity := down.Sub(thorax.Direction(down))
	var pull [2]maths.Vec
	for i, side := range humanSides {
		frame, _ := pose.Frame(side.name("upper-arm"))
		site := proportionSite{part: armPart, attachment: 1, side: side.sign}
		shoulder := human.spec.measure(side.shoulder(), site)
		elbow := human.spec.measure(side.elbow(), site)
		line := elbow.Sub(shoulder).Unit()
		from := thorax.Direction(line)
		to := frame.Direction(line)
		// Cap travel independently of rotation magnitude and follow the ribs.
		pull[i] = to.Sub(from).Scale(.70)
	}
	scale := human.spec.Stature / humanHeight
	for i, motion := range human.tissue {
		if motion.gravity != 0 || motion.shoulder != 0 {
			offset := gravity.Scale(motion.gravity).Add(pull[motion.side].Scale(motion.shoulder))
			points[i] = points[i].Add(offset.Scale(scale))
		}
	}
}

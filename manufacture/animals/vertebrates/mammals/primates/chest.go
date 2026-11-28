package primates

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"math"
)

// wall builds the transverse thorax independently of the mobile tissue.
// The medial pectoral bed is continuous at x=0 for every section.
func (s Torso) wall(a, y float64) (maths.Vec, maths.Vec) {
	ring := sectionAt(y)
	sa, ca := math.Sincos(a)
	x := ring.w * s.Width * sa
	rest := maths.V(x, y, (.5*(ring.front+ring.back)*ca+.5*(ring.front-ring.back))*s.Depth)
	p := rest
	ax := math.Abs(x)
	front := maths.Smooth(-.13, .62, ca)
	// The front of the rib basket turns more sharply into its side wall
	// than an ellipse. The lateral costal plane dies out below the ribs.
	costal := bell((y - 1.155) / .115)
	p.Z += s.Depth * (.031*costal + .012*bell((y-.874)/.071)) * ca * sa * sa
	// Below the anterior axillary line the caudal rib cage falls medially.
	// The narrow lower wall continues down into the oblique flank.
	p.X -= .014 * sa * sa * sa * bell((y-1.110)/.077)
	// The abdominal wall crosses the waist anterior to the flanks. Its
	// paired obliques descend medially from the costal margin to the pelvis.
	p.Z += front * (.016*bell(x/.090)*bell((y-.916)/.069) +
		.006*bell((ax-(.078+.37*(y-.947)))/.039)*bell((y-.991)/.068))
	crest := .902 - .25*(.142-ax)
	p.Z += .005 * front * bell((ax-.123)/.048) * bell((y-crest)/.023)
	p.X += sa * .005 * bell((y-.899)/.031) * sa * sa
	// Paired neck tendons descend toward the manubrium, with a recessed
	// fossa between the lateral clavicle and trapezius.
	neck := .021 + .33*(y-1.346)
	p.Z += front * .0032 * bell((ax-neck)/.012) * bell((y-1.394)/.047)
	p.Z -= front * .005 * bell((ax-.096)/.039) * bell((y-1.373)/.017)
	// The sternum supports the anterior wall. The bed is broadest above the
	// mammary attachment and descends diagonally into the anterior axillary fold.
	pec := .016 * bell(x/(.131*s.Width)) * bell((y-1.253)/.075)
	pec += .015 * bell(x/(.058*s.Width)) * bell((y-1.246)/.080)
	pec += .009 * bell(x/(.098*s.Width)) * bell((y-1.194)/.066)
	p.Z += front * pec
	p.Z += front * .006 * bell((y-(1.342+.014*math.Sin(math.Pi*ax/(.172*s.Width))))/.012) * maths.Smooth(.004, .027, ax)
	p.Z -= front * .005 * bell((y-1.345)/.014) * bell(x/.017)
	p.Z += s.costalMargin(ax, y, front)
	p.Z -= front * .0055 * bell(x/.010) * bell((y-.975)/.011)
	p.Z += .010 * bell((ax-.071*s.Width)/.047) * bell((y-1.272)/.079) * maths.Smooth(.13, .85, -ca)
	return p, rest
}

func (s Torso) costalMargin(x, y, front float64) float64 {
	ribY := 1.125 - .044*math.Pow(x/(.151*s.Width), 1.48)
	return front * .0038 * bell((y-ribY)/.018) * maths.Smooth(.010, .050, x)
}

// Paired sagittal values specify anterior projection and mobility independently.
// A uniform cubic B-spline gives C2 joins. Under higher load the lower pole
// descends past part of its fixed return, possible because Y is a coordinate
// of this surface rather than its independent variable.
var meridianStations = [...][2]float64{
	{0, 0}, {0, 0}, {.095, .065}, {.43, .31}, {.82, .90}, {.99, 1.02},
	{1, .82}, {.90, .58}, {.74, .39}, {.56, .23}, {.41, .12}, {.28, .06},
	{.17, .026}, {.075, .008}, {.02, 0}, {0, 0}, {0, 0},
}

func meridian(y float64) (depth, load float64) {
	u := (y + .108) / .018
	k := int(math.Floor(u))
	t := u - float64(k)
	w := [4]float64{math.Pow(1-t, 3) / 6, (4 - 6*t*t + 3*t*t*t) / 6, (1 + 3*t + 3*t*t - 3*t*t*t) / 6, t * t * t / 6}
	for j, weight := range w {
		p := meridianStations[max(0, min(len(meridianStations)-1, k+j-1))]
		depth += weight * p[0]
		load += weight * p[1]
	}
	return
}

// chestPoint evaluates the supporting wall and its distributed mobile load.
// The inferior return results from differential travel between the lower
// lobe and its fixed perimeter. There is no separately cut crease stripe.
func (s Torso) chestPoint(a, y float64) (maths.Vec, maths.Vec) {
	a = math.Remainder(a, 2*math.Pi)
	p, rest := s.wall(a, y)
	ax := math.Abs(rest.X)
	travel := maths.Smooth(.86, 1.39, y)
	roll := maths.Rotation(maths.XAxis, s.Pitch*travel)
	yaw := maths.Rotation(maths.YAxis, s.Twist*travel)
	gravity := roll.In(yaw.In(maths.YAxis.Neg()))
	front := maths.Smooth(-.27, .53, math.Cos(a))
	ringWidth := sectionAt(y).w * s.Width
	side, arm, sign := s.side(rest.X)
	cx, cy := s.landmarks(side)
	upper := maths.Smooth(-.032, .095, y-cy)
	center := math.Asin(math.Min(.92, (cx+.022*upper)/ringWidth))
	// Measure the root around the rib cage. The mobile medial root narrows inferiorly;
	// its lateral tail fans superiorly over the fixed supporting wall.
	angle := math.Remainder(sign*a-center, 2*math.Pi)
	arc := angle * ringWidth / (side.Width * (.95 + .67*maths.Smooth(-.62, .52, angle) + .20*upper))
	innerEdge := .048 + .022*(1-maths.Smooth(-.042, .018, y-cy))
	blend := math.Exp(-1.50*arc*arc) * front * maths.Smooth(0, innerEdge, sign*rest.X)
	span := .87 + 1.5*side.Width
	sector := .011*math.Tanh(arc) + .025*(1-math.Exp(-1.5*arc*arc))
	at := (y - cy - (1-upper)*sector) / span
	project, mobility := meridian(at)
	elevation := math.Sin(.62 * arm)
	depth := side.Projection * project * blend * (1 - .59*maths.Smooth(.024, .091, y-cy)) * (1 - .12*elevation)
	// The axillary tail wraps around the flank. At lateral meridians
	// its visible volume sits higher than the inferior anterior pole.
	tailAngle := math.Remainder(sign*a-1.36, 2*math.Pi)
	tailY := cy + .031 + .018*math.Tanh(tailAngle)
	tail := bell(tailAngle/.51) * bell((y-tailY)/.066)
	p.X += sign * .34 * side.Projection * tail * maths.Smooth(0, .055, sign*rest.X)
	p.Z += math.Cos(.6*a) * depth
	p.X += math.Sin(a) * .59 * depth
	drop := s.Gravity * side.Projection * (.16 + .48*s.Compliance) * (1 + .22*(s.AgeYears-38)/42)
	p = p.Add(gravity.Scale(drop * mobility * blend))
	p.Y += elevation * .010 * blend * project
	elevate := elevation
	axillary := bell((ax-.144*s.Width)/.032) * bell((y-1.264-.23*(ax-.144*s.Width))/.062)
	p.Z += maths.Smooth(.03, .74, math.Cos(a)) * (.009 + .009*elevate) * axillary
	p.Y += elevate * .031 * axillary
	pivot := maths.V(0, .91, 0)
	return yaw.Out(roll.Out(p.Sub(pivot))).Add(pivot), rest
}

func (s Torso) sample(a, y float64, seed uint64) (maths.Vec, maths.Vec) {
	if s.LegacyContour {
		return s.legacySample(a, y, seed)
	}
	p, rest := s.chestPoint(a, y)
	side, _, sign := s.side(rest.X)
	cx, cy := s.landmarks(side)
	u := (math.Abs(rest.X) - cx) / side.AreolaRadius
	v := (y - cy + .008) / (.93 * side.AreolaRadius)
	radius := math.Hypot(u, v)
	front := maths.Smooth(.008, .072, rest.Z)
	if front > 0 && radius < 1.8 {
		// A surface differential supplies the local posed normal before relief.
		aa, _ := s.chestPoint(a+.0002, y)
		yy, _ := s.chestPoint(a, y+.00003)
		n := aa.Sub(p).Cross(yy.Sub(p)).Unit()
		pap, tub := s.relief(side, u, v, sign)
		p = p.Add(n.Scale(front * (.0018*math.Exp(-1.28*radius*radius) + side.PapillaProjection*pap + .00068*tub)))
	}
	p.Z += front * .00009 * noise.Gradient(seed, p.X*530, p.Y*530, p.Z*530)
	return p, rest
}

// Each bump has its own angular spacing, radius and amplitude in the chart.
func (s Torso) relief(side Side, u, v, sign float64) (float64, float64) {
	radial := math.Hypot(u, v) * side.AreolaRadius / side.PapillaRadius
	pap := math.Exp(-.63*math.Pow(radial, 4) - .37*radial*radial)
	tub := 0.
	for i := 0; i < 9; i++ {
		k := float64(i) + .2*sign
		angle := k*2.39996323 + .31*math.Sin(1.8*k+sign)
		r := .41 + .36*(.5+.5*math.Sin(13.74*k+sign))
		size := .083 + .048*(.5+.5*math.Cos(7.7*k))
		d := math.Hypot(u-r*math.Cos(angle), v-r*math.Sin(angle)) / size
		tub += math.Exp(-d*d) * (.59 + .41*math.Pow(math.Sin(6.3*k+.5), 2))
	}
	return pap, tub
}

// lowerTurn locates the steepest forward rise on the inferior sagittal
// profile. It is a measured surface proxy, not a prescribed fold height.
func (s Torso) lowerTurn(side Side, sign float64) float64 {
	x, cy := s.landmarks(side)
	point := func(y float64) maths.Vec {
		a := sign * math.Asin(x/(sectionAt(y).w*s.Width))
		p, _ := s.chestPoint(a, y)
		return p
	}
	best, bestY := -math.Pi, cy
	for j := 0; j <= 120; j++ {
		y := cy - .114 + .00074*float64(j)
		d := point(y + .00025).Sub(point(y - .00025))
		angle := math.Atan2(d.Z, d.Y)
		if angle > best {
			best, bestY = angle, y
		}
	}
	return bestY
}

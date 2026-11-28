package primates

import (
	"math"

	"github.com/janearc/slayer/maths"
)

func orbitalRelief(x, y float64) float64 {
	width := (x - eyeHalfSpacing) / .024
	socket := -.0022 * bell(width) * bell((y-eyeHeight)/.014)
	browLine := 1.580 + .003*bell((x-.037)/.020)
	brow := .0048 * bell((x-.027)/.028) * bell((y-browLine)/.012)
	lower := .0018 * bell(width) * bell((y-1.549+.17*(x-eyeHalfSpacing))/.0075)
	return socket + brow + lower
}

func mandibularShape(point maths.Vec, sine, cosine float64) maths.Vec {
	y := point.Y
	if y >= 1.537 {
		return point
	}
	// The body of the mandible sweeps up toward the ramus, in front of
	// the ear. The sweep fades into the nape before the rear midline.
	angle := maths.Smooth(.18, .89, math.Abs(sine)) * maths.Smooth(-.64, -.13, cosine)
	band := maths.Smooth(1.461, 1.495, y) * (1 - maths.Smooth(1.504, 1.537, y))
	point.Y += .015 * angle * band
	// The masseter thickens the outer jaw just ahead of the ramus.
	masseter := maths.Smooth(headBase, 1.497, y) * (1 - maths.Smooth(1.521, 1.537, y))
	point.X += sine * .0042 * bell((cosine-.25)/.32) * masseter
	return point
}

const mouthHalfWidth = .025

type lipContour struct {
	line, upper, lower, span float64
}

func lipsAt(x float64) lipContour {
	across := x / mouthHalfWidth
	span := math.Max(0, 1-across*across)
	line := 1.5138 + .0007*bell((x-.0065)/.0048) - .0015*math.Pow(across, 4)
	upper := span * (.0035 + .0020*bell((x-.0065)/.005))
	lower := span * (.0055 + .0010*bell(x/.011))
	return lipContour{line: line, upper: upper, lower: lower, span: span}
}

func mouthRelief(x, y float64) float64 {
	lips := lipsAt(x)
	offset := y - lips.line
	upper := .0030 * bell((offset-.53*lips.upper)/(.0018+.40*lips.upper))
	lower := .0039 * bell((offset+.53*lips.lower)/(.0021+.42*lips.lower))
	seam := .00115 * bell(offset/.0013)
	alveolar := .0033 * bell(x/.034) * bell((y-1.517)/.014)
	chin := .0060 * bell(x/.030) * bell((y-1.498)/.0105)
	groove := .0015 * bell(x/.023) * bell((y-1.504)/.0055)
	philtrum := .00115 * bell((x-.0043)/.0030) * bell((y-1.521)/.0047)
	return lips.span*(upper+lower-seam) + alveolar + chin + philtrum - groove
}

func (lips lipContour) pigment(y float64) float64 {
	offset := y - lips.line
	upper := 1 - maths.Smooth(lips.upper-.0008, lips.upper+.0008, offset)
	lower := maths.Smooth(-lips.lower-.0008, -lips.lower+.0008, offset)
	return upper * lower * maths.Smooth(0, .26, lips.span)
}

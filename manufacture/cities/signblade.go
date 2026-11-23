package cities

import (
	"math"
	"strings"

	"github.com/janearc/slayer/maths"
)

// blade is a blade sign: a cabinet that stands out of a wall on a bracket
// of two arms, braced by a stay, with the letters of a short word of a
// trade one under the next, upright, on both of its faces, which face along
// the wall, each face made as the sign's make says. Its frame has its x out
// of the wall, its y up from the top of the bracket, and its z across, the
// cabinet between the two faces, from the top of the bracket down.
func (job signJob) blade(mount signMount) []placedElement {
	drawing := job.drawing
	word := strings.ReplaceAll(job.tradeWord("short"), " ", "")
	cells := float64(len([]rune(word)))
	pitch, margin := drawing.fixed("sign-blade-pitch"),
		drawing.fixed("sign-blade-margin")
	em := math.Min(drawing.number("sign-blade-em", job.at),
		mount.High/(cells*pitch+2*margin))
	wide := em * (drawing.number("sign-blade-width"+job.signing.suffix,
		job.at) + 2*margin)
	high := em * (cells*pitch + 2*margin)
	thick, arm := drawing.fixed("sign-blade-depth"),
		drawing.fixed("sign-blade-arm")
	reach := math.Max(mount.Reach, wide+arm)
	var lines []textLine
	for index, letter := range []rune(word) {
		top := em * (cells*pitch + margin - float64(index)*pitch)
		lines = append(lines, textLine{words: string(letter), em: em,
			neon: job.neon, base: top - em*(pitch+job.cap)/half})
	}
	face, err := job.dressed(facePlan{wide: wide, high: high,
		depth: thick / half, lines: lines, spot: job.spot, sides: 2,
		luminance: drawing.fixed("sign-blade-luminance")})
	drawing.fail(err)
	if drawing.trouble != nil {
		return nil
	}
	parts := job.bracket(reach, high, arm)
	parts = append(parts, within(face, maths.NewSubspace(maths.V(reach-wide,
		-high, thick/half), maths.Identity, 0))...)
	parts = append(parts, within(face, maths.NewSubspace(maths.V(reach, -high,
		-thick/half), maths.Orientation{X: maths.XAxis.Neg(), Y: maths.YAxis,
		Z: maths.ZAxis.Neg()}, 0))...)
	return append(parts, job.bladeRig(reach, wide, high, thick, arm)...)
}

// bracket is the two arms a blade sign's cabinet hangs from, which run out
// of the wall, one near the top of the cabinet and one near its foot, each
// on a plate bolted to the wall.
func (job signJob) bracket(reach, high, arm float64) []placedElement {
	drawing := job.drawing
	hang, plate := drawing.fixed("sign-blade-hang"),
		drawing.fixed("sign-plate-side")
	across := -arm / half
	var parts []placedElement
	for _, low := range []float64{-hang - arm, hang - high} {
		parts = append(parts, put("sign arm", Box{reach, arm, arm}, job.frame,
			maths.V(0, low, across)))
		if job.lettered() {
			parts = append(parts, job.plated(maths.V(0, low+arm/half, 0),
				maths.XAxis, maths.Up, plate, plate)...)
		}
	}
	return parts
}

// bladeRig is what else holds a blade sign and feeds it, near by: caps on
// the top and the foot of the cabinet, wider than it, a stay from the wall
// above the bracket down to the top arm, and the feed: a conduit up from
// the cabinet's top, along to the wall, and into a junction box.
func (job signJob) bladeRig(reach, wide, high, thick, arm float64) []placedElement {
	if !job.lettered() {
		return nil
	}
	drawing := job.drawing
	over, lid := drawing.fixed("sign-cap-over"), drawing.fixed("sign-cap-high")
	pipe, junction := drawing.fixed("sign-conduit-side"),
		drawing.fixed("sign-junction-side")
	hang, plate := drawing.fixed("sign-blade-hang"),
		drawing.fixed("sign-plate-side")
	left, middle := reach-wide-over, reach-wide/half
	arm0 := -hang - arm/half
	anchor := maths.V(0, arm0+drawing.fixed("sign-stay-rise"), 0)
	level := lid + drawing.fixed("sign-blade-riser")
	parts := []placedElement{
		put("sign cap", Box{wide + 2*over, lid, thick + 2*over}, job.frame,
			maths.V(left, 0, -thick/half-over)),
		put("sign cap", Box{wide + 2*over, lid, thick + 2*over}, job.frame,
			maths.V(left, -high-lid, -thick/half-over)),
		strut("sign stay", anchor, maths.V(reach*drawing.fixed(
			"sign-stay-at"), arm0, 0), drawing.fixed("sign-stay-side"),
			job.frame),
		strut("sign conduit", maths.V(middle, lid, 0),
			maths.V(middle, level, 0), pipe, job.frame),
		strut("sign conduit", maths.V(middle, level, 0),
			maths.V(junction, level, 0), pipe, job.frame),
		put("sign junction", Box{junction, junction, junction}, job.steel,
			maths.V(0, level-junction/half, -junction/half))}
	return append(parts, job.plated(anchor, maths.XAxis, maths.Up, plate,
		plate)...)
}

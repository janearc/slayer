package cities

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The ways a roof sign's frame is made.
const (
	frameTruss = "truss"
	frameLegs  = "legs"
)

// roof is a roof sign: one or two lines of the words of a trade on a
// board that stands on a steel frame at the edge of a roof, the face made
// as the sign's make says. The letters are as big as the culture draws,
// made smaller to fit the room the mount gives. Its frame has its x along
// the edge, middle at the mount's point, its y up from the roof, and its z
// out, the back of its board at the point.
func (job signJob) roof(mount signMount) []placedElement {
	drawing := job.drawing
	words := []string{job.tradeWord("word")}
	// a second line says another word of the trade: where the trade has
	// no other, it is not set, as it would say the first again
	if drawing.count("sign-roof-lines", job.at) > 1 &&
		job.tradeHasAnother("short", words[0]) {
		words = append(words, job.tradeWord("short"))
	}
	lift := drawing.number("sign-roof-lift", job.at)
	margin := drawing.number("sign-roof-margin", job.at)
	_, wide, high := job.lines(words, 1, margin)
	em := math.Min(drawing.number("sign-roof-em", job.at), math.Min(
		mount.Wide*drawing.fixed("sign-roof-fit")/wide,
		(mount.High-lift)/high))
	lines, wide, high := job.lines(words, em, margin)
	depth := drawing.fixed("sign-roof-panel-depth")
	face, err := job.dressed(facePlan{wide: wide, high: high, depth: depth,
		lines: lines, spot: job.spot, sides: 1,
		luminance: drawing.fixed("sign-roof-luminance")})
	drawing.fail(err)
	if drawing.trouble != nil {
		return nil
	}
	parts := within(face, standingAt(maths.V(-wide/half, lift, depth)))
	return append(parts, job.rig(wide, high, lift, depth)...)
}

// rig is what holds a roof sign's board up: from far off two posts, and
// near by a steel frame of the culture's style, the conduit and the
// transformer that feed it, and plates with bolts where it stands on the
// roof.
func (job signJob) rig(wide, high, lift, depth float64) []placedElement {
	if !job.lettered() {
		return job.posts(wide, lift, depth)
	}
	drawing := job.drawing
	var parts []placedElement
	if drawing.pick("sign-frame-style", drawing.options("sign-frame-style"),
		job.at).Name == frameLegs {
		parts = job.legs(wide, lift, depth)
	} else {
		parts = job.truss(wide, high, lift)
	}
	return append(parts, job.feed(wide, lift)...)
}

// posts are the two posts a roof sign's board stands on, at its two ends.
func (job signJob) posts(wide, lift, depth float64) []placedElement {
	post := job.drawing.fixed("sign-roof-post")
	back := maths.V(0, 0, (depth-post)/half)
	return []placedElement{
		put("sign post", Box{post, lift, post}, job.frame,
			back.Add(maths.V(-wide/half+post, 0, 0))),
		put("sign post", Box{post, lift, post}, job.frame,
			back.Add(maths.V(wide/half-2*post, 0, 0)))}
}

// truss is a lattice behind a roof sign's board: a column at the end of
// each bay of the culture's width, a chord across the foot and the top of
// the board, a diagonal in each bay, one way and then the other, and
// struts from the ends, and the middle if it is wide, back to the roof.
// Each column and strut stands on a plate bolted to the roof.
func (job signJob) truss(wide, high, lift float64) []placedElement {
	drawing := job.drawing
	side, brace := drawing.fixed("sign-frame-member"),
		drawing.fixed("sign-frame-brace")
	bays := max(1, int(math.Ceil(wide/drawing.fixed("sign-frame-bay"))))
	left, right, top, back := -wide/half+side, wide/half-side, lift+high,
		-side/half
	column := func(index int) float64 {
		return left + (right-left)*float64(index)/float64(bays)
	}
	parts := []placedElement{
		strut("sign chord", maths.V(left, lift, back),
			maths.V(right, lift, back), side, job.frame),
		strut("sign chord", maths.V(left, top-side/half, back),
			maths.V(right, top-side/half, back), side, job.frame)}
	for index := 0; index <= bays; index++ {
		foot := maths.V(column(index), 0, back)
		parts = append(parts, strut("sign column", foot,
			maths.V(foot.X, top, back), side, job.frame))
		parts = append(parts, job.plated(foot, maths.Up, maths.ZAxis,
			drawing.fixed("sign-plate-side"),
			drawing.fixed("sign-plate-side"))...)
		if index < bays {
			parts = append(parts, job.diagonal(column(index),
				column(index+1), lift, top, back, index, side))
		}
	}
	stays := []float64{left, right}
	if float64(bays) >= drawing.fixed("sign-frame-middle") {
		stays = append(stays, (left+right)/half)
	}
	for _, x := range stays {
		foot := maths.V(x, 0, -brace)
		parts = append(parts, strut("sign strut", maths.V(x, lift+high*
			drawing.fixed("sign-frame-strut"), back), foot, side, job.frame))
		parts = append(parts, job.plated(foot, maths.Up, maths.ZAxis,
			drawing.fixed("sign-plate-side"),
			drawing.fixed("sign-plate-side"))...)
	}
	return parts
}

// diagonal is the brace across one bay of a truss between its chords,
// rising to the right in a bay of an even number and to the left in an odd
// one.
func (job signJob) diagonal(from, to, low, top, back float64, bay int,
	side float64) placedElement {
	start, end := maths.V(from, low, back), maths.V(to, top, back)
	if bay%half == 1 {
		start, end = maths.V(from, top, back), maths.V(to, low, back)
	}
	return strut("sign diagonal", start, end, side, job.frame)
}

// legs are the frame of a low roof sign: at each station along the board
// a pair of legs, one standing in front of its board and one behind, that
// meet under it, braced across between them, and a beam along the foot of
// the board that carries it. Each leg stands on a bolted plate.
func (job signJob) legs(wide, lift, depth float64) []placedElement {
	drawing := job.drawing
	side, brace := drawing.fixed("sign-frame-member"),
		drawing.fixed("sign-frame-brace")
	front := depth + drawing.fixed("sign-leg-front")
	stations := max(2, int(math.Ceil(wide/drawing.fixed("sign-leg-pitch"))))
	left, right := -wide/half+side, wide/half-side
	apex := depth / half
	parts := []placedElement{strut("sign beam", maths.V(left, lift-side/half, apex),
		maths.V(right, lift-side/half, apex), side, job.frame)}
	plate := drawing.fixed("sign-plate-side")
	for index := range stations {
		x := left + (right-left)*float64(index)/float64(stations-1)
		for _, foot := range []maths.Vec{maths.V(x, 0, -brace),
			maths.V(x, 0, front)} {
			parts = append(parts, strut("sign leg", foot, maths.V(x, lift,
				apex), side, job.frame))
			parts = append(parts, job.plated(foot, maths.Up, maths.ZAxis,
				plate, plate)...)
		}
		level := drawing.fixed("sign-leg-cross")
		parts = append(parts, strut("sign cross", maths.V(x, lift*level,
			-brace+(apex+brace)*level), maths.V(x, lift*level,
			front-(front-apex)*level), side, job.frame))
	}
	return parts
}

// feed is what lights a roof sign from the building: a transformer box
// on the roof behind the frame near its right end, a conduit up to a
// junction box under the board, and another conduit back across the roof
// to a junction box at the building's wall.
func (job signJob) feed(wide, lift float64) []placedElement {
	drawing := job.drawing
	pipe, junction := drawing.fixed("sign-conduit-side"),
		drawing.fixed("sign-junction-side")
	x, over := wide/half-drawing.fixed("sign-feed-inset"),
		drawing.fixed("sign-feed-behind")
	width, high, deep := drawing.fixed("sign-transformer-wide"),
		drawing.fixed("sign-transformer-high"),
		drawing.fixed("sign-transformer-deep")
	parts := []placedElement{
		put("sign transformer", Box{width, high, deep}, job.steel,
			maths.V(x-width/half, 0, -over-deep/half)),
		strut("sign conduit", maths.V(x, high, -over),
			maths.V(x, lift-junction, -over), pipe, job.frame),
		put("sign junction", Box{junction, junction, junction}, job.steel,
			maths.V(x-junction/half, lift-2*junction, -over-junction/half))}
	run := -over - deep/half - drawing.fixed("sign-feed-run")
	return append(parts, strut("sign conduit", maths.V(x, high/half,
		-over-deep/half), maths.V(x, high/half, run), pipe, job.frame),
		put("sign junction", Box{junction, junction, junction}, job.steel,
			maths.V(x-junction/half, high/half-junction/half, run-junction)))
}

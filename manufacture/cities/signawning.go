package cities

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// awningMounts is the place an awning may hang on a wall, over the foot of
// it: across the middle, as wide as the wall less a margin at each end, up
// to the room the culture gives an awning, if the wall is upright, high
// and long enough and nothing covers the stretch.
func (drawing *drawer) awningMounts(wall wallEdge) []signMount {
	wide := math.Min(wall.length-2*drawing.fixed("sign-awning-margin"),
		drawing.fixed("sign-awning-room"))
	rise := drawing.fixed("sign-awning-rise")
	if wide < drawing.fixed("sign-least-wide") ||
		wall.height < rise+drawing.fixed("sign-awning-high") ||
		!wall.clear(-wide/half, wide/half) {
		return nil
	}
	return []signMount{{Kind: MountAwning,
		Point: wall.mid.Add(maths.Up.Scale(rise)), Out: wall.out,
		Along: wall.along(), Wide: wide,
		High:  drawing.fixed("sign-awning-high"),
		Reach: drawing.fixed("sign-awning-reach-most")}}
}

// awning is an awning over a shop's door: a canopy of cloth in stripes on
// ribs that slope down from the wall, a valance of cloth that hangs from
// its front edge with the words of a trade on it, in paint or in lit
// letters, and under it lamps. It is held by stays and plates at the wall.
// Its frame has its x along the wall, middle at the mount's point, which
// is the foot of the valance, its y up and its z out of the wall.
func (job signJob) awning(mount signMount) []placedElement {
	drawing := job.drawing
	wide := math.Min(drawing.number("sign-awning-wide", job.at), mount.Wide)
	reach := math.Min(drawing.number("sign-awning-reach", job.at),
		mount.Reach)
	valance := drawing.number("sign-awning-valance", job.at)
	cloth := drawing.pick("sign-cloth", drawing.options("sign-cloth"), job.at)
	stripe := drawing.pick("sign-cloth", drawing.options("sign-cloth"),
		job.at, cloth.Name)
	base := paint("sign cloth", cloth)
	parts := job.canopy(wide, reach, valance, base,
		paint("sign cloth", stripe))
	thick := drawing.fixed("sign-valance-thick")
	parts = append(parts, put("awning valance", Box{wide, valance, thick},
		base, maths.V(-wide/half, 0, reach)))
	if !job.lettered() {
		return append(parts, job.awningLamps(wide, reach, valance)...)
	}
	parts = append(parts, job.valanceLetters(wide, reach, valance, thick)...)
	parts = append(parts, job.awningRig(wide, reach, valance)...)
	return append(parts, job.awningLamps(wide, reach, valance)...)
}

// canopy is the sloping top of an awning: from the wall, a drop above the
// valance's top edge, down to the front edge, in stripes of two cloths of
// the culture's width, and when near, the ribs under it and the rail along
// its front edge.
func (job signJob) canopy(wide, reach, valance float64, cloth,
	stripe *thing.Material) []placedElement {
	drawing := job.drawing
	drop, thick := drawing.fixed("sign-awning-drop"),
		drawing.fixed("sign-canopy-thick")
	width := drawing.fixed("sign-awning-stripe")
	count := int(math.Ceil(wide / width))
	var parts []placedElement
	for index := range count {
		left := -wide/half + float64(index)*width
		across := math.Min(width, wide/half-left)
		material := cloth
		if index%half == 1 {
			material = stripe
		}
		middle := left + across/half
		parts = append(parts, beam("awning canopy",
			maths.V(middle, valance+drop, 0), maths.V(middle, valance, reach),
			thick, across, material))
	}
	if !job.lettered() {
		return parts
	}
	rib := drawing.fixed("sign-awning-rib")
	parts = append(parts, strut("awning rail", maths.V(-wide/half, valance,
		reach), maths.V(wide/half, valance, reach), rib, job.frame))
	pitch := drawing.fixed("sign-awning-rib-pitch")
	for index := 0; index <= int(math.Ceil(wide/pitch)); index++ {
		x := math.Min(-wide/half+float64(index)*pitch, wide/half)
		parts = append(parts, strut("awning rib", maths.V(x, valance+drop-
			thick-rib/half, 0), maths.V(x, valance-rib/half, reach), rib,
			job.frame))
	}
	return parts
}

// valanceLetters are the words of a trade on an awning's valance, one line
// in the middle of it, as tall as the culture fills it and made smaller to
// fit its width: painted flat in a light paint, or in lit channel letters.
func (job signJob) valanceLetters(wide, reach, valance,
	thick float64) []placedElement {
	drawing := job.drawing
	words := job.tradeWord("word")
	span, err := job.width(words)
	drawing.fail(err)
	em := math.Min(valance*drawing.fixed("sign-valance-fill")/job.cap,
		wide*drawing.fixed("sign-valance-fit")/span)
	plan := facePlan{wide: wide, high: valance, spot: job.spot,
		lines: []textLine{{words: words, em: em, neon: job.neon}}}
	made, err := job.ink(words, em, maths.V(0, (valance-job.cap*em)/half,
		reach+thick), job.neon, plan.dark(wearLetter, 0,
		letterCount(words)))
	drawing.fail(err)
	plan.depth = thick
	return append(made, within(job.streaks(plan, 0), standingAt(maths.V(
		-wide/half, 0, reach+thick)))...)
}

// awningRig is what holds an awning to its wall: a stay at each end that
// runs from a plate on the wall, bolted, above the canopy's back edge down
// to the front corner of the canopy.
func (job signJob) awningRig(wide, reach, valance float64) []placedElement {
	drawing := job.drawing
	side, plate := drawing.fixed("sign-stay-side"),
		drawing.fixed("sign-plate-side")
	high := valance + drawing.fixed("sign-awning-drop") +
		drawing.fixed("sign-awning-stay-rise")
	var parts []placedElement
	for _, end := range []float64{-1, 1} {
		x := end * (wide/half - side)
		anchor := maths.V(x, high, 0)
		parts = append(parts, strut("awning stay", anchor,
			maths.V(x, valance, reach), side, job.frame))
		parts = append(parts, job.plated(anchor, maths.ZAxis, maths.Up,
			plate, plate)...)
	}
	return parts
}

// leastLampsToSpare is how many warm lamps an awning has before one of them
// may be out: two, as an awning of one lamp put out would throw no light,
// and an awning is a sign that lights its street.
const leastLampsToSpare = 2

// awningLamps are the lamps under an awning's front edge: for a painted
// valance, warm lamps on short stalks, as many as the width has room for,
// each throwing the awning's light on the pavement under it; for a lit
// valance, lamps in the colour of its letters in front of it. One of the
// warm lamps may be out, if there are at least leastLampsToSpare.
func (job signJob) awningLamps(wide, reach, valance float64) []placedElement {
	drawing := job.drawing
	if job.look == lookLitValance {
		return job.lamps(wide, maths.V(0, valance/half, reach+
			drawing.fixed("sign-valance-thick")+job.stand(valance)+
			drawing.fixed("sign-lamp-out")), drawing.fixed(
			"sign-awning-luminance"), 1, job.glow(job.neon),
			colourOf(job.neon))
	}
	count := max(1, int(math.Ceil(wide/drawing.fixed("sign-lamp-spacing"))))
	warm := drawing.pick("sign-lamp-warm", drawing.options("sign-lamp-warm"),
		job.at)
	glass := job.shining("awning lamp", warm.Name, colourOf(warm),
		"sign-lamp-glow")
	out := facePlan{spot: job.spot, lines: []textLine{{}}}.darkOne(wearLamp,
		count)
	if count < leastLampsToSpare {
		out = -1
	}
	// the awning's luminance is the awning's: the lamps that burn share it
	burning := count
	if out >= 0 {
		burning--
	}
	lamp := &Lamp{Luminance: drawing.fixed("sign-awning-luminance") /
		float64(burning), Tint: colourOf(warm)}
	head, stalk := drawing.fixed("sign-lamp-head"),
		drawing.fixed("sign-awning-stalk")
	back := drawing.fixed("sign-awning-lamp-back")
	roof := valance + drawing.fixed("sign-awning-drop")*back/reach
	var parts []placedElement
	for index := range count {
		x := -wide/half + wide*(float64(index)+1.0/half)/float64(count)
		hung := maths.V(x, roof, reach-back)
		top := hung.Sub(maths.V(0, stalk, 0))
		lit, shining := glass, lamp
		if index == out {
			lit, shining = job.unlit("awning lamp"), nil
		}
		// the stalk is near detail, as plates and bolts are: from far
		// the head is all there is
		if job.lettered() {
			parts = append(parts, strut("awning stalk", hung, top,
				drawing.fixed("sign-awning-rib"), job.frame))
		}
		parts = append(parts, placedElement{Element: solid{Name: "sign lamp",
			Form: Box{head, head, head}, Material: lit, Lamp: shining},
			Place: standingAt(top.Sub(maths.V(head/half, head, head/half)))})
	}
	return parts
}

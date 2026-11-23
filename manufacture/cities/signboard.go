package cities

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/typography"
)

// The ways a sign's face is made, which is the make of the sign. A
// screen's own make, lit as a field of tiles, is led, and an awning's are
// its valance painted or lit.
const (
	lookLamped     = "lamped"
	lookBox        = "box"
	lookBulb       = "bulb"
	lookLED        = "led"
	lookValance    = "valance"
	lookLitValance = "lit-valance"
)

// The kinds of wear a sign shows, which a make may carry.
const (
	wearLetter = "letter"
	wearTube   = "tube"
	wearPanel  = "panel"
	wearBulb   = "bulb"
	wearLamp   = "lamp"
)

// wornSpot is the one thing about a sign that has gone wrong, if
// anything: what kind of wear, and by shares from nought to one which of
// its lines and which of that line's things.
type wornSpot struct {
	kind       string
	line, item float64
}

// shareOf is which of a count of things a share from nought to one is.
func shareOf(share float64, count int) int {
	return min(int(share*float64(count)), count-1)
}

// wear is the wear of a sign of a make, drawn for a place: none by the
// chance the culture gives, and otherwise a kind that make shows.
func (drawing *drawer) wear(look string, at place) wornSpot {
	if !drawing.chance("sign-wear-share", at) {
		return wornSpot{}
	}
	name := "sign-wear-" + look
	return wornSpot{kind: drawing.pick(name, drawing.options(name), at).Name,
		line: drawing.number("sign-wear-line", at),
		item: drawing.number("sign-wear-item", at)}
}

// textLine is one line of letters on a face: its words, its em, and the
// height of its baseline above the foot of the face. The middle of the
// line is the middle of the face.
type textLine struct {
	words    string
	em, base float64
	neon     option
}

// facePlan is a face to be made: the board's width and height and its
// depth behind the face, the lines of letters on it, the sign's wear, and
// the luminance its lamps share among themselves and the sides it has.
type facePlan struct {
	wide, high, depth float64
	lines             []textLine
	spot              wornSpot
	luminance         float64
	sides             int
}

// dark is which of a count of things in a line is out, or minus one: the
// thing of the sign's wear, if it is that kind and that line's.
func (plan facePlan) dark(kind string, line, count int) int {
	spot := plan.spot
	if spot.kind != kind || count == 0 ||
		line != shareOf(spot.line, len(plan.lines)) {
		return -1
	}
	return shareOf(spot.item, count)
}

// darkOne is which of a count of things on the whole face is out, or
// minus one, for wear that belongs to no line of letters.
func (plan facePlan) darkOne(kind string, count int) int {
	return plan.dark(kind, shareOf(plan.spot.line, len(plan.lines)), count)
}

// letterCount is how many letters of a line's words have an outline.
func letterCount(words string) int {
	count := 0
	for _, letter := range words {
		if letter != ' ' {
			count++
		}
	}
	return count
}

// unlit is the dark glass of a lit thing that has gone out, of a kind.
func (job signJob) unlit(of string) *thing.Material {
	return &thing.Material{Name: "cities sign " + of + " out",
		Colour: maths.Grey(job.drawing.fixed("sign-unlit"))}
}

// slab is the dark board of a face, standing back from its front.
func (job signJob) slab(plan facePlan, material *thing.Material) placedElement {
	return put("sign board", Box{plan.wide, plan.high, plan.depth}, material,
		maths.V(0, 0, -plan.depth))
}

// lines are words set one line under the next on a board, in ems of the
// first line's, as lines of letters, and the width and height of the
// board that holds them with a margin, a share of the em, all round. Each
// line after the first is as small again as the culture's ratio.
func (job signJob) lines(words []string, em, margin float64) ([]textLine,
	float64, float64) {
	drawing := job.drawing
	ratio, gap := drawing.fixed("sign-line-ratio"),
		drawing.fixed("sign-line-gap")
	var set []textLine
	wide, high, scale := 0.0, 0.0, em
	for index, one := range words {
		span, err := job.width(one)
		drawing.fail(err)
		wide = math.Max(wide, span*scale)
		high += job.cap * scale
		if index > 0 {
			high += gap * em
		}
		set = append(set, textLine{words: one, em: scale, neon: job.neon})
		scale *= ratio
	}
	wide, high = wide+2*margin*em, high+2*margin*em
	foot := high - margin*em
	for index := range set {
		foot -= job.cap * set[index].em
		set[index].base = foot
		foot -= gap * em
	}
	return set, wide, high
}

// dressed is a face made: the board and what is on it, in the frame of the
// face, which has its front at z nought, its left foot at the origin, the
// board behind it and whatever stands out of it toward +z. It is one glowing
// panel and its lamps when the sign is too far off to read.
func (job signJob) dressed(plan facePlan) ([]placedElement, error) {
	if !job.lettered() {
		return append([]placedElement{job.slab(plan, job.board())},
			job.lamps(plan.wide, maths.V(plan.wide/half, plan.high/half,
				job.drawing.fixed("sign-lamp-out")), plan.luminance,
				plan.sides, job.glow(job.neon), colourOf(job.neon))...), nil
	}
	switch job.look {
	case lookTube:
		return job.tubeFace(plan)
	case lookLamped:
		return job.lampedFace(plan)
	case lookBox:
		return job.boxFace(plan)
	case lookBulb:
		return job.bulbFace(plan)
	}
	return job.channelFace(plan)
}

// streaks are the dark stains that run down a face from its top edge, on
// the share of signs the culture says, as many as it draws, each a stripe a
// share of the face's height long, standing a little out of the plane given.
func (job signJob) streaks(plan facePlan, plane float64) []placedElement {
	drawing := job.drawing
	if !drawing.chance("sign-streak-share", job.at) {
		return nil
	}
	count := drawing.count("sign-streak-count", job.at)
	stain := paint("sign stain", drawing.pick("sign-stain",
		drawing.options("sign-stain"), job.at))
	depth := drawing.fixed("sign-streak-depth")
	var parts []placedElement
	for index := range count {
		at := job.at
		at.Dwelling = index + 1
		wide := drawing.number("sign-streak-width", at)
		long := plan.high * drawing.number("sign-streak-length", at)
		left := (plan.wide - wide) * drawing.number("sign-streak-at", at)
		parts = append(parts, put("sign streak", Box{wide, long, depth},
			stain, maths.V(left, plan.high-long, plane)))
	}
	return parts
}

// tubeFace is a face of open tube letters on a backing: the backing
// board, a tube round it, and for each letter a tube bent to the edge of
// its shape, standing off the backing on posts.
func (job signJob) tubeFace(plan facePlan) ([]placedElement, error) {
	drawing := job.drawing
	parts := []placedElement{job.slab(plan, job.steel)}
	parts = append(parts, job.rim(plan)...)
	for index := range plan.lines {
		made, err := job.tubes(plan, index)
		if err != nil {
			return nil, err
		}
		parts = append(parts, made...)
	}
	em := plan.lines[0].em
	stand := em * (drawing.fixed("sign-tube-stand") +
		drawing.fixed("sign-tube-radius"))
	parts = append(parts, job.streaks(plan, 0)...)
	return append(parts, job.lamps(plan.wide, maths.V(plan.wide/half,
		plan.high/half, stand+drawing.fixed("sign-lamp-out")),
		plan.luminance, plan.sides, job.glow(job.neon),
		colourOf(job.neon))...), nil
}

// channelFace is a face of channel letters, deep dark bodies with a face
// that glows, set on a dark board.
func (job signJob) channelFace(plan facePlan) ([]placedElement, error) {
	parts := []placedElement{job.slab(plan, job.steel)}
	for index, line := range plan.lines {
		made, err := job.ink(line.words, line.em, maths.V(plan.wide/half,
			line.base, 0), line.neon, plan.dark(wearLetter, index,
			letterCount(line.words)))
		if err != nil {
			return nil, err
		}
		parts = append(parts, made...)
	}
	parts = append(parts, job.streaks(plan, 0)...)
	return append(parts, job.lamps(plan.wide, maths.V(plan.wide/half,
		plan.high/half, job.stand(plan.lines[0].em)+
			job.drawing.fixed("sign-lamp-out")), plan.luminance, plan.sides,
		job.glow(job.neon), colourOf(job.neon))...), nil
}

// lampedFace is a painted board with its letters painted flat in a colour
// against it, and no light of its own: a row of lamps on arms stands out
// above it and shines down its face.
func (job signJob) lampedFace(plan facePlan) ([]placedElement, error) {
	drawing := job.drawing
	parts := []placedElement{job.slab(plan, job.board())}
	parts = append(parts, job.trims(plan, drawing.fixed("sign-trim"),
		drawing.fixed("sign-trim-depth"), job.frame)...)
	for _, line := range plan.lines {
		made, err := job.ink(line.words, line.em, maths.V(plan.wide/half,
			line.base, 0), line.neon, -1)
		if err != nil {
			return nil, err
		}
		parts = append(parts, made...)
	}
	parts = append(parts, job.streaks(plan, 0)...)
	return append(parts, job.arms(plan)...), nil
}

// trims are four bars round the front of a board, as wide as given and
// standing out of it as far as given, in a material.
func (job signJob) trims(plan facePlan, across, out float64,
	material *thing.Material) []placedElement {
	return []placedElement{
		put("sign trim", Box{plan.wide, across, out}, material, maths.V(0, 0,
			0)),
		put("sign trim", Box{plan.wide, across, out}, material,
			maths.V(0, plan.high-across, 0)),
		put("sign trim", Box{across, plan.high - 2*across, out}, material,
			maths.V(0, across, 0)),
		put("sign trim", Box{across, plan.high - 2*across, out}, material,
			maths.V(plan.wide-across, across, 0))}
}

// boxFace is a lit box: a dark cabinet with a face of translucent panels
// that glow, in a frame, and the letters dark flat on the glowing face.
// One panel may be gone, showing the cabinet behind it.
func (job signJob) boxFace(plan facePlan) ([]placedElement, error) {
	drawing := job.drawing
	bezel, seam := drawing.fixed("sign-box-bezel"),
		drawing.fixed("sign-seam-width")
	skin, lip := drawing.fixed("sign-panel-skin"),
		drawing.fixed("sign-box-lip")
	glass := job.shining("box face", job.neon.Name, colourOf(job.neon),
		"sign-box-glow")
	parts := []placedElement{job.slab(plan, job.steel)}
	parts = append(parts, job.trims(plan, bezel, lip, job.frame)...)
	panels := max(1, int(math.Ceil((plan.wide-2*bezel)/
		drawing.fixed("sign-panel-wide"))))
	across, missing := (plan.wide-2*bezel)/float64(panels),
		plan.darkOne(wearPanel, panels)
	for index := range panels {
		left := bezel + float64(index)*across
		if index > 0 {
			parts = append(parts, put("sign mullion", Box{seam,
				plan.high - 2*bezel, lip}, job.frame,
				maths.V(left-seam/half, bezel, 0)))
		}
		if index != missing {
			parts = append(parts, put("sign panel", Box{across - seam,
				plan.high - 2*bezel, skin}, glass,
				maths.V(left+seam/half, bezel, 0)))
		}
	}
	for _, line := range plan.lines {
		made, err := job.ink(line.words, line.em, maths.V(plan.wide/half,
			line.base, skin), line.neon, -1)
		if err != nil {
			return nil, err
		}
		parts = append(parts, made...)
	}
	parts = append(parts, job.streaks(plan, skin)...)
	return append(parts, job.lamps(plan.wide, maths.V(plan.wide/half,
		plan.high/half, skin+drawing.fixed("sign-lamp-out")), plan.luminance,
		plan.sides, glass, colourOf(job.neon))...), nil
}

// bulbFace is a board with channel letters and a border of bulbs round it,
// a bulb to each step of the culture's pitch, each on a socket. One bulb
// may be out.
func (job signJob) bulbFace(plan facePlan) ([]placedElement, error) {
	drawing := job.drawing
	parts, err := job.channelFace(plan)
	if err != nil {
		return nil, err
	}
	colour := drawing.pick("sign-bulb-colour",
		drawing.options("sign-bulb-colour"), job.at)
	lit := job.shining("bulb", colour.Name, colourOf(colour), "sign-bulb-glow")
	spots := job.bulbSpots(plan)
	out := plan.darkOne(wearBulb, len(spots))
	size, socket := drawing.fixed("sign-bulb-side"),
		drawing.fixed("sign-bulb-socket")
	for index, at := range spots {
		glass := lit
		if index == out {
			glass = job.unlit("bulb")
		}
		parts = append(parts, put("sign socket", Box{size / half, size / half,
			socket}, job.frame, at.Sub(maths.V(size/4, size/4, 0))),
			put("sign bulb", Box{size, size, size}, glass,
				at.Sub(maths.V(size/half, size/half, -socket))))
	}
	return parts, nil
}

// bulbSpots are the places of the bulbs round a board: along its four
// edges at the culture's inset and pitch, a bulb at each corner.
func (job signJob) bulbSpots(plan facePlan) []maths.Vec {
	drawing := job.drawing
	inset, pitch := drawing.fixed("sign-bulb-inset"),
		drawing.fixed("sign-bulb-pitch")
	across, up := int(math.Ceil((plan.wide-2*inset)/pitch)),
		int(math.Ceil((plan.high-2*inset)/pitch))
	var spots []maths.Vec
	for step := range across {
		x := inset + (plan.wide-2*inset)*float64(step)/float64(across)
		spots = append(spots, maths.V(x, inset, 0),
			maths.V(plan.wide-x, plan.high-inset, 0))
	}
	for step := range up {
		y := inset + (plan.high-2*inset)*float64(step)/float64(up)
		spots = append(spots, maths.V(plan.wide-inset, y, 0),
			maths.V(inset, plan.high-y, 0))
	}
	return spots
}

// arms are the lamps on arms that light a painted board from above: each
// a stem up from a plate on the board's top, an arm out over its face, and
// a lamp at the end shining down. The luminance is the culture's, shared
// among them. One may be out.
func (job signJob) arms(plan facePlan) []placedElement {
	drawing := job.drawing
	count := max(2, min(int(math.Ceil(plan.wide/
		drawing.fixed("sign-arm-spacing"))), int(drawing.fixed("sign-arm-most"))))
	rise, reach := drawing.fixed("sign-arm-rise"), drawing.fixed("sign-arm-reach")
	side, head := drawing.fixed("sign-arm-side"), drawing.fixed("sign-lamp-head")
	warm := drawing.pick("sign-lamp-warm", drawing.options("sign-lamp-warm"),
		job.at)
	glass, out := job.shining("arm lamp", warm.Name, colourOf(warm),
		"sign-lamp-glow"), plan.darkOne(wearLamp, count)
	lamp := &Lamp{Luminance: drawing.fixed("sign-lamped-luminance") /
		float64(count), Tint: colourOf(warm)}
	var parts []placedElement
	for index := range count {
		x := plan.wide * (float64(index) + 1.0/half) / float64(count)
		foot, elbow := maths.V(x, plan.high, -plan.depth/half),
			maths.V(x, plan.high+rise, -plan.depth/half)
		tip := maths.V(x, plan.high+rise, reach)
		parts = append(parts, job.plated(foot, maths.Up, maths.ZAxis,
			side*half, side*half)...)
		parts = append(parts, strut("sign stem", foot, elbow, side,
			job.frame), strut("sign arm", elbow, tip, side, job.frame))
		lit, burning := glass, lamp
		if index == out {
			lit, burning = job.unlit("arm lamp"), nil
		}
		parts = append(parts, placedElement{Element: solid{Name: "sign lamp",
			Form: Box{head, head, head}, Material: lit, Lamp: burning},
			Place: standingAt(tip.Sub(maths.V(head/half, head, head/half)))})
	}
	return parts
}

// rim is the tube round the edge of a backing board: a closed rectangle
// of tube, an inset in from the edge, standing off the board on posts like
// the letters.
func (job signJob) rim(plan facePlan) []placedElement {
	drawing := job.drawing
	em := plan.lines[0].em
	inset := em * drawing.fixed("sign-rim-inset")
	stand := em * drawing.fixed("sign-tube-stand")
	path := []maths.Vec{maths.V(inset, inset, stand),
		maths.V(plan.wide-inset, inset, stand),
		maths.V(plan.wide-inset, plan.high-inset, stand),
		maths.V(inset, plan.high-inset, stand)}
	glass := job.glow(job.neon)
	parts := []placedElement{{Element: solid{Name: "sign rim", Form: meshForm{
		sweep(path, true, em*drawing.fixed("sign-tube-radius"))},
		Material: glass}, Place: standingAt(maths.Vec{})}}
	return append(parts, job.standoffs(append(path, path[0]), em, stand)...)
}

// tubes are the tube letters of one line of a plan: for each letter, for
// each ring of its outline, a tube bent to the ring with a gap in it, held
// off the backing on posts and ending in electrodes. A tube may be out, or
// a whole letter.
func (job signJob) tubes(plan facePlan, index int) ([]placedElement, error) {
	line := plan.lines[index]
	made, err := typography.Outlines(job.faceFor(), typography.Text{
		Words: line.words, Size: line.em, Tolerance: job.tolerance()})
	if err != nil {
		return nil, err
	}
	var letters []typography.Outline
	rings := 0
	for _, one := range made {
		if len(one.Rings) > 0 {
			letters = append(letters, one)
			rings += len(one.Rings)
		}
	}
	middle := maths.V(plan.wide/half, line.base+job.rise*line.em, 0)
	dead, out := plan.dark(wearLetter, index, len(letters)),
		plan.dark(wearTube, index, rings)
	var parts []placedElement
	ring := 0
	for number, one := range letters {
		for _, round := range one.Rings {
			glass := job.glow(line.neon)
			if number == dead || ring == out {
				glass = job.unlit("tube")
			}
			parts = append(parts, job.tubeRing(line, middle.Add(one.At),
				round, glass)...)
			ring++
		}
	}
	return parts, nil
}

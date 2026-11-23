package cities

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// screen is a Times Square style screen on a wall: a dark housing held off
// the wall on posts with a frame round it, lines of words, each in a colour
// of its own, and a face made as the sign's make says: a field of lit tiles
// with the letters on it, a lit box, or a board with a border of bulbs. Its
// frame has its x along the wall, middle at the mount's point, its y up
// from the screen's foot, its z out of the wall.
func (job signJob) screen(mount signMount) []placedElement {
	drawing := job.drawing
	stand, depth := drawing.fixed("sign-screen-stand"),
		drawing.fixed("sign-screen-depth")
	plan := facePlan{wide: mount.Wide, high: mount.High, depth: depth,
		lines: job.screenRows(mount), spot: job.spot, sides: 1,
		luminance: drawing.fixed("sign-screen-luminance")}
	var face []placedElement
	var err error
	if job.look == lookLED {
		face, err = job.ledFace(plan)
	} else {
		face, err = job.dressed(plan)
	}
	drawing.fail(err)
	if drawing.trouble != nil {
		return nil
	}
	parts := within(face, standingAt(maths.V(-mount.Wide/half, 0,
		stand+depth)))
	return append(parts, job.screenRig(mount, stand, depth)...)
}

// screenRows are the lines of words on a screen's face, from the top of
// its panel down, each in the middle of a row of its own and each as tall
// as the culture fills a row, made smaller to fit its width. Every line is
// said before any is measured, so that the sign is set in one face.
func (job signJob) screenRows(mount signMount) []textLine {
	drawing := job.drawing
	bezel := drawing.fixed("sign-screen-bezel")
	rows := drawing.count("sign-screen-lines", job.at)
	tall := (mount.High - 2*bezel) / float64(rows)
	setter := job
	if job.look == lookLED {
		setter = job.tubed()
	}
	var lines []textLine
	for row := range rows {
		at := job.at
		at.Storey = row
		lines = append(lines, textLine{words: job.word("screen-line", at),
			neon: drawing.pick("sign-neon", drawing.options("sign-neon"),
				at)})
	}
	for row := range lines {
		span, err := setter.width(lines[row].words)
		drawing.fail(err)
		em := math.Min(tall*drawing.fixed("sign-screen-fill")/job.cap,
			(mount.Wide-2*bezel)*drawing.fixed("sign-screen-fit")/span)
		lines[row].em = em
		lines[row].base = mount.High - bezel - float64(row+1)*tall +
			(tall-job.cap*em)/half
	}
	return lines
}

// ledFace is the face of a screen that is a field of tiles: a dark housing
// with a panel in it that glows a dark tint of its own, a frame round the
// panel, seams between its tiles, and on it the lines of letters in
// pillowed tube neon. One tile may be dead.
func (job signJob) ledFace(plan facePlan) ([]placedElement, error) {
	drawing := job.drawing
	bezel, skin := drawing.fixed("sign-screen-bezel"),
		drawing.fixed("sign-screen-skin")
	tint := drawing.pick("sign-screen-colour",
		drawing.options("sign-screen-colour"), job.at)
	glass := job.shining("screen", tint.Name, colourOf(tint),
		"sign-screen-glow")
	parts := []placedElement{job.slab(plan, job.steel),
		put("screen face", Box{plan.wide - 2*bezel, plan.high - 2*bezel, skin},
			glass, maths.V(bezel, bezel, 0))}
	parts = append(parts, job.trims(plan, bezel,
		drawing.fixed("sign-box-lip"), job.frame)...)
	parts = append(parts, job.tiles(plan, bezel, skin)...)
	setter := job.tubed()
	for index, line := range plan.lines {
		if !job.lettered() {
			break
		}
		made, err := setter.ink(line.words, line.em, maths.V(plan.wide/half,
			line.base, skin), line.neon, plan.dark(wearLetter, index,
			letterCount(line.words)))
		if err != nil {
			return nil, err
		}
		parts = append(parts, made...)
	}
	parts = append(parts, job.streaks(plan, skin)...)
	return append(parts, job.lamps(plan.wide, maths.V(plan.wide/half,
		plan.high/half, skin+drawing.fixed("sign-lamp-out")), plan.luminance,
		plan.sides, glass, colourOf(tint))...), nil
}

// tiles are the seams between the tiles of a screen's panel, dark strips
// across it each way at the culture's tile size, and a dead tile, dark,
// if the wear is one.
func (job signJob) tiles(plan facePlan, bezel, skin float64) []placedElement {
	drawing := job.drawing
	seam, out := drawing.fixed("sign-seam-width"),
		drawing.fixed("sign-seam-depth")
	wide, high := plan.wide-2*bezel, plan.high-2*bezel
	across := max(1, int(math.Round(wide/drawing.fixed("sign-tile-wide"))))
	up := max(1, int(math.Round(high/drawing.fixed("sign-tile-high"))))
	var parts []placedElement
	for index := 1; index < across; index++ {
		parts = append(parts, put("sign seam", Box{seam, high, out},
			job.steel, maths.V(bezel+wide*float64(index)/float64(across)-
				seam/half, bezel, skin)))
	}
	for index := 1; index < up; index++ {
		parts = append(parts, put("sign seam", Box{wide, seam, out},
			job.steel, maths.V(bezel, bezel+high*float64(index)/
				float64(up)-seam/half, skin)))
	}
	dead := plan.darkOne(wearPanel, across*up)
	if dead < 0 {
		return parts
	}
	tileWide, tileHigh := wide/float64(across), high/float64(up)
	return append(parts, put("sign dead tile", Box{tileWide - seam,
		tileHigh - seam, out}, job.unlit("tile"), maths.V(
		bezel+tileWide*float64(dead%across)+seam/half,
		bezel+tileHigh*float64(dead/across)+seam/half, skin)))
}

// screenRig is what holds a screen to its wall: posts that stand its
// housing off the wall, each on a bolted plate, along its top and its foot,
// and the feed: a junction box on the wall beside it, a conduit from the
// box to the housing, and another running up the wall with clips.
func (job signJob) screenRig(mount signMount, stand, depth float64) []placedElement {
	if !job.lettered() {
		return nil
	}
	drawing := job.drawing
	side, plate := drawing.fixed("sign-post-side"),
		drawing.fixed("sign-plate-side")
	count := max(2, int(math.Ceil(mount.Wide/drawing.fixed(
		"sign-post-pitch")))+1)
	var parts []placedElement
	for index := range count {
		x := -mount.Wide/half + side + (mount.Wide-2*side)*float64(index)/
			float64(count-1)
		for _, y := range []float64{side, mount.High - 2*side} {
			parts = append(parts, put("sign post", Box{side, side, stand},
				job.frame, maths.V(x-side/half, y, 0)))
			parts = append(parts, job.plated(maths.V(x, y+side/half, 0),
				maths.ZAxis, maths.Up, plate, plate)...)
		}
	}
	return append(parts, job.screenFeed(mount, stand, depth)...)
}

// screenFeed is the junction box on the wall beside a screen, the conduit
// from it out to the screen's housing, and the conduit up the wall with a
// clip at each end of it.
func (job signJob) screenFeed(mount signMount, stand, depth float64) []placedElement {
	drawing := job.drawing
	pipe, junction := drawing.fixed("sign-conduit-side"),
		drawing.fixed("sign-junction-side")
	x := mount.Wide/half + drawing.fixed("sign-feed-inset")
	y := mount.High * drawing.fixed("sign-feed-share")
	run := drawing.fixed("sign-feed-run")
	into := stand + depth/half
	return []placedElement{
		put("sign junction", Box{junction, junction, junction}, job.steel,
			maths.V(x-junction/half, y-junction/half, 0)),
		strut("sign conduit", maths.V(x, y, junction),
			maths.V(x, y, into), pipe, job.frame),
		strut("sign conduit", maths.V(x, y, into),
			maths.V(mount.Wide/half, y, into), pipe, job.frame),
		strut("sign conduit", maths.V(x, y+junction/half, pipe/half),
			maths.V(x, y+run, pipe/half), pipe, job.frame),
		put("sign clip", Box{pipe * half, pipe * half, pipe}, job.frame,
			maths.V(x-pipe, y+junction, 0)),
		put("sign clip", Box{pipe * half, pipe * half, pipe}, job.frame,
			maths.V(x-pipe, y+run-pipe, 0))}
}

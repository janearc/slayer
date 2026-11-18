package main

import (
	"image"
	"image/color"
	"math"
	"strconv"

	"github.com/janearc/slayer/cheeptune"
)

// Layout, in reference pixels: a frame 1080 high, scaled to the frame.
// Picked 2026-11-16 so the position reads across a room and the lanes
// take the rest.
const (
	reference       = 1080.0
	readoutBaseline = 205
	shadowShift     = 5 // the readout's pink and cyan edges, each side
	squareTop       = 240
	squareSide      = 80
	squareGap       = 28
	tickTop         = 336
	tickHeight      = 18
	tickTall        = 30  // the tick on each beat
	tickRow         = 640 // the width all the ticks take
	laneTop         = 392
	laneBottom      = 952
	laneGap         = 8
	labelWidth      = 230
	margin          = 20
	numberBaseline  = 990
	floorTop        = 1008
	floorLines      = 6    // the perspective floor's crossing lines
	floorSpokes     = 9    // its lines toward the horizon, each side
	floorNarrow     = 0.18 // how much the spokes close at the horizon
	notePad         = 6    // between a pitched lane's edge and its notes
	glowReach       = 3
	playheadWidth   = 4
)

// barsShown is how many bars the lanes show at once, and playheadAt
// where in them the playhead stands, as a fraction from the left.
const (
	barsShown  = 2
	playheadAt = 1.0 / 3
)

// Lighting. A hit or a beat flashes and falls to a third over flash
// beats; a note that is not sounding is drawn at resting of its colour,
// and the current beat's square keeps held of its light.
const (
	flash   = 0.25
	resting = 0.3
	held    = 0.55
	ghost   = 0.35 // the smallest a hit's mark is, as a share of its row
)

// film is what every frame shares: the score's clock, the lanes, the
// layout, and the plate drawn once beneath them.
type film struct {
	score  *cheeptune.Score
	lanes  []lane
	unit   float64
	writer *writer
	plate  *image.RGBA
	bands  []image.Rectangle
	roll   image.Rectangle
}

// newFilm lays out the lanes for a frame size and draws the plate.
func newFilm(score *cheeptune.Score, lanes []lane, width, height int) (
	*film, error) {
	unit := float64(height) / reference
	w, err := newWriter(unit)
	if err != nil {
		return nil, err
	}
	f := &film{score: score, lanes: lanes, unit: unit, writer: w}
	f.roll = image.Rect(f.px(labelWidth), f.px(laneTop),
		width-f.px(margin), f.px(laneBottom))
	f.bands = f.layBands()
	f.plate = f.drawPlate(width, height)
	return f, nil
}

// px is a reference length in this frame's pixels.
func (f *film) px(reference float64) int {
	return int(math.Round(reference * f.unit))
}

// layBands shares the lanes' height by weight, a gap between each.
func (f *film) layBands() []image.Rectangle {
	total := 0
	for _, l := range f.lanes {
		total += l.weight()
	}
	gaps := f.px(laneGap) * (len(f.lanes) - 1)
	room := float64(f.roll.Dy() - gaps)
	var bands []image.Rectangle
	top := f.roll.Min.Y
	for _, l := range f.lanes {
		tall := int(room * float64(l.weight()) / float64(total))
		bands = append(bands, image.Rect(0, top, f.roll.Max.X, top+tall))
		top += tall + f.px(laneGap)
	}
	return bands
}

// drawPlate draws what never moves: the ground, the lanes' bands and
// their names, and the horizon over the floor.
func (f *film) drawPlate(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		fill(img, image.Rect(0, y, width, y+1),
			mix(groundTop, groundBottom, float64(y)/float64(height)))
	}
	for i, l := range f.lanes {
		band, colour := f.bands[i], laneColours[i%len(laneColours)]
		blend(img, band, laneGround, 1)
		fill(img, image.Rect(0, band.Min.Y, f.px(playheadWidth),
			band.Max.Y), colour)
		f.writer.left(img, textName, f.px(margin),
			band.Min.Y+f.px(textSizes[textName]+notePad), colour, l.name)
		for row, hit := range l.hits {
			f.writer.right(img, textRow, f.roll.Min.X-f.px(margin/2),
				f.rowMiddle(band, len(l.hits), row)+f.px(textSizes[textRow]/3),
				faint, rowName(hit))
		}
	}
	fill(img, image.Rect(0, f.px(floorTop), width, f.px(floorTop)+2), cyan)
	return img
}

// rowMiddle is the middle of a percussion row.
func (f *film) rowMiddle(band image.Rectangle, rows, row int) int {
	tall := float64(band.Dy()) / float64(rows)
	return band.Min.Y + int(tall*(float64(row)+0.5))
}

// perBeat is the width of one beat on the lanes, in pixels.
func (f *film) perBeat() float64 {
	return float64(f.roll.Dx()) / float64(barsShown*f.score.Meter)
}

// playheadX is the column the playhead stands in.
func (f *film) playheadX() int {
	return f.roll.Min.X + int(float64(f.roll.Dx())*playheadAt)
}

// xAt is the column a beat falls in, with now at the playhead.
func (f *film) xAt(beat, now float64) int {
	return f.playheadX() + int(math.Round((beat-now)*f.perBeat()))
}

// frame draws the moment some seconds into the song into img.
func (f *film) frame(img *image.RGBA, seconds float64) {
	copy(img.Pix, f.plate.Pix)
	now := f.score.BeatsAt(seconds)
	f.drawReadout(img, now)
	f.drawCount(img, now)
	f.drawGrid(img, now)
	for i := range f.lanes {
		f.drawLane(img, i, now)
	}
	x := f.playheadX()
	glow(img, image.Rect(x-f.px(playheadWidth)/2, f.roll.Min.Y,
		x+f.px(playheadWidth)/2, f.roll.Max.Y), img.Bounds(), sunny,
		f.px(glowReach))
	f.drawFloor(img, now)
}

// drawReadout writes BAR.BEAT.SIXTEENTH large, with neon edges.
func (f *film) drawReadout(img *image.RGBA, now float64) {
	words := readout(now, f.score.Meter)
	middle, base := img.Bounds().Dx()/2, f.px(readoutBaseline)
	shift := f.px(shadowShift)
	f.writer.centre(img, textReadout, middle-shift, base, cyan, words)
	f.writer.centre(img, textReadout, middle+shift, base, pink, words)
	f.writer.centre(img, textReadout, middle, base, ink, words)
}

// drawCount draws the beats of the bar as squares and the sixteenths as
// ticks: done ones half lit, the current one lit.
func (f *film) drawCount(img *image.RGBA, now float64) {
	meter := f.score.Meter
	_, beat, sixteenth := position(now, meter)
	since := now - math.Floor(now+nudge)
	side, gap := f.px(squareSide), f.px(squareGap)
	left := (img.Bounds().Dx() - meter*side - (meter-1)*gap) / 2
	for k := range meter {
		r := image.Rect(left+k*(side+gap), f.px(squareTop),
			left+k*(side+gap)+side, f.px(squareTop)+side)
		on := cyan
		if k == 0 {
			on = pink
		}
		if k != beat-1 {
			fill(img, r, unlit)
			continue
		}
		level := held + (1-held)*math.Exp(-since/flash)
		glow(img, r, img.Bounds(), mix(unlit, on, level), f.px(glowReach))
	}
	f.drawTicks(img, (beat-1)*sixteenths+sixteenth-1)
}

// drawTicks draws the bar's sixteenths, the current one lit.
func (f *film) drawTicks(img *image.RGBA, current int) {
	count := f.score.Meter * sixteenths
	slot := float64(f.px(tickRow)) / float64(count)
	left := float64(img.Bounds().Dx()-f.px(tickRow)) / 2
	for i := range count {
		tall := f.px(tickHeight)
		if i%sixteenths == 0 {
			tall = f.px(tickTall)
		}
		x := int(left + slot*float64(i))
		r := image.Rect(x+2, f.px(tickTop), x+int(slot)-2, f.px(tickTop)+tall)
		switch {
		case i == current:
			glow(img, r, img.Bounds(), sunny, f.px(glowReach))
		case i < current:
			fill(img, r, purple)
		default:
			fill(img, r, unlit)
		}
	}
}

// drawGrid draws a line at every sixteenth on the lanes, stronger on the
// beat and pink on the bar, and numbers the bars along the foot.
func (f *film) drawGrid(img *image.RGBA, now float64) {
	meter := f.score.Meter
	from := now - float64(f.playheadX()-f.roll.Min.X)/f.perBeat()
	to := now + float64(f.roll.Max.X-f.playheadX())/f.perBeat()
	first, last := int(math.Ceil(from*sixteenths)), to*sixteenths
	for s := first; float64(s) <= last; s++ {
		x := f.xAt(float64(s)/sixteenths, now)
		r := image.Rect(x, f.roll.Min.Y, x+1, f.roll.Max.Y)
		switch {
		case s%(meter*sixteenths) == 0:
			fill(img, r.Inset(-1), pink)
			f.writer.centre(img, textNumber, x, f.px(numberBaseline), pink,
				strconv.Itoa(s/(meter*sixteenths)+1))
		case s%sixteenths == 0:
			blend(img, r, purple, held)
		default:
			blend(img, r, purple, resting/2)
		}
	}
}

// drawLane draws one lane's notes or hits in view.
func (f *film) drawLane(img *image.RGBA, index int, now float64) {
	l, band := f.lanes[index], f.bands[index]
	colour := laneColours[index%len(laneColours)]
	clip := image.Rect(f.roll.Min.X, band.Min.Y, f.roll.Max.X, band.Max.Y)
	for _, e := range l.events {
		x0, x1 := f.xAt(e.Beats, now), f.xAt(e.Beats+e.Length, now)
		if x1 < clip.Min.X || x0 > clip.Max.X {
			continue
		}
		if len(l.hits) > 0 {
			f.drawHit(img, l, e, band, clip, colour, now)
			continue
		}
		lit := now >= e.Beats && now < e.Beats+e.Length
		for _, note := range e.Notes {
			r := f.noteRect(l, band, note, x0, x1)
			if lit {
				glow(img, r, clip, colour, f.px(glowReach))
				continue
			}
			fill(img, r.Intersect(clip), mix(laneGround, colour, resting))
		}
	}
}

// noteRect is where a note is drawn: its pitch's row, from its start to
// just short of its end.
func (f *film) noteRect(l lane, band image.Rectangle, note, x0,
	x1 int) image.Rectangle {
	inner := float64(band.Dy() - 2*f.px(notePad))
	tall := inner / float64(l.high-l.low+1)
	y := band.Min.Y + f.px(notePad) + int(tall*float64(l.high-note))
	return image.Rect(x0, y, max(x0+2, x1-2), y+max(3, int(tall)-1))
}

// drawHit draws one hit as a mark on its row, as tall as it is loud,
// flashing as it sounds.
func (f *film) drawHit(img *image.RGBA, l lane, e cheeptune.Event,
	band, clip image.Rectangle, colour color.RGBA, now float64) {
	rows := len(l.hits)
	middle := f.rowMiddle(band, rows, l.row(e.Hit))
	share := math.Max(ghost, e.Loud/l.loudest)
	half := int(float64(band.Dy()) / float64(rows) * 0.4 * share)
	x := f.xAt(e.Beats, now)
	wide := int(f.perBeat() / sixteenths * 0.7)
	r := image.Rect(x, middle-max(2, half), x+wide, middle+max(2, half))
	since := now - e.Beats
	if since >= 0 && since < 1 {
		level := resting + (1-resting)*math.Exp(-since/flash)
		glow(img, r, clip, mix(laneGround, colour, level), f.px(glowReach))
		return
	}
	fill(img, r.Intersect(clip), mix(laneGround, colour, resting+0.15))
}

// drawFloor draws the perspective floor under the bar numbers, its
// crossing lines coming forward one a beat.
func (f *film) drawFloor(img *image.RGBA, now float64) {
	bounds := img.Bounds()
	top, bottom := float64(f.px(floorTop)), float64(bounds.Max.Y)
	middle, wide := float64(bounds.Dx())/2, float64(bounds.Dx())
	phase := now - math.Floor(now)
	for k := range floorLines {
		p := (float64(k) + phase) / floorLines
		y := top + (bottom-top)*p*p
		line(img, 0, y, wide, y, pink, resting+0.5*p)
	}
	for k := -floorSpokes; k <= floorSpokes; k++ {
		foot := middle + float64(k)*wide/floorSpokes
		head := middle + (foot-middle)*floorNarrow
		line(img, head, top, foot, bottom, purple, held)
	}
}

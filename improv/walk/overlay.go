package walk

import (
	"image"
	"image/color"
	"math"
)

// bilinear is a still's colour at a point between pixel middles, from
// its four nearest, the edge held past the still's sides.
func bilinear(source *image.RGBA, x, y float64) color.RGBA {
	size := source.Rect.Size()
	x = math.Max(0, math.Min(float64(size.X-1), x))
	y = math.Max(0, math.Min(float64(size.Y-1), y))
	left, top := int(x), int(y)
	right, foot := min(left+1, size.X-1), min(top+1, size.Y-1)
	across, down := x-float64(left), y-float64(top)
	upper := mix(source.RGBAAt(left, top), source.RGBAAt(right, top), across)
	lower := mix(source.RGBAAt(left, foot), source.RGBAAt(right, foot),
		across)
	return mix(upper, lower, down)
}

// overlay draws, on the picture through a window: the scale bar; the
// labels, each standing clear of what was drawn before it where it can;
// and, when not lit as in the show, the tag saying how it is lit, in
// the first corner clear of them.
func (shot Shot) overlay(frame *image.RGBA, part window,
	tones palette) error {
	box, err := shot.drawBar(frame, part, tones)
	if err != nil {
		return err
	}
	taken := []image.Rectangle{box}
	for _, label := range shot.labels {
		box, err := label.draw(frame, part, tones, taken)
		if err != nil {
			return err
		}
		point := onFrame(label.at, part)
		taken = append(taken, box, image.Rectangle{Min: point,
			Max: point.Add(image.Pt(1, 1))}.Inset(0-dotRadius-dotRing))
	}
	if shot.tag == "" {
		return nil
	}
	return tagged(frame, shot.tag, tones, taken)
}

// tagged draws a tag in the first corner of the picture clear of what
// is taken: top left, top right, foot left, foot right; or top left.
func tagged(frame *image.RGBA, tag string, tones palette,
	taken []image.Rectangle) error {
	wide, err := measure(tag, tagSize)
	if err != nil {
		return err
	}
	size := image.Pt(wide+2*labelPad, int(tagSize)+2*labelPad)
	area := pictureArea().Inset(tagInset)
	corners := []image.Point{area.Min, {area.Max.X - size.X, area.Min.Y},
		{area.Min.X, area.Max.Y - size.Y}, area.Max.Sub(size)}
	at := corners[0]
	for _, corner := range corners {
		box := image.Rectangle{Min: corner, Max: corner.Add(size)}
		if clearOf(box, taken) {
			at = corner
			break
		}
	}
	_, err = placedPlate(frame, tag, at, tagSize, tones)
	return err
}

// clearOf is whether a box keeps off every box taken.
func clearOf(box image.Rectangle, taken []image.Rectangle) bool {
	for _, other := range taken {
		if box.Overlaps(other.Inset(0 - labelPad)) {
			return false
		}
	}
	return true
}

// pictureArea is the picture's part of the frame.
func pictureArea() image.Rectangle {
	return image.Rect(0, pictureTop, pictureSize.X, pictureTop+pictureSize.Y)
}

// placedPlate draws words in ink on a plate of ground, its top left at
// a point, kept inside the picture, and says where the plate went.
func placedPlate(frame *image.RGBA, words string, at image.Point,
	size float64, tones palette) (image.Rectangle, error) {
	wide, err := measure(words, size)
	if err != nil {
		return image.Rectangle{}, err
	}
	box := image.Rect(0, 0, wide+2*labelPad, int(size)+2*labelPad)
	box = box.Add(at)
	box = keptInside(box, pictureArea())
	fill(frame, box, tones["ground"])
	baseline := box.Min.Y + labelPad + int(size*capShare)
	return box, write(frame, words, image.Pt(box.Min.X+labelPad, baseline),
		size, tones["ink"])
}

// capShare is how far down a line of type its baseline sits, as a share
// of its size. Measured on Go's regular face by eye: its capitals stand
// about four fifths of the size.
const capShare = 0.8

// keptInside is a box moved to lie inside an area.
func keptInside(box, area image.Rectangle) image.Rectangle {
	shift := image.Point{}
	if box.Max.X > area.Max.X {
		shift.X = area.Max.X - box.Max.X
	}
	if box.Min.X+shift.X < area.Min.X {
		shift.X = area.Min.X - box.Min.X
	}
	if box.Max.Y > area.Max.Y {
		shift.Y = area.Max.Y - box.Max.Y
	}
	if box.Min.Y+shift.Y < area.Min.Y {
		shift.Y = area.Min.Y - box.Min.Y
	}
	return box.Add(shift)
}

// draw draws a label: a leader from its words to a ringed dot on its
// place. Its words stand on its own side if they fit there clear of the
// place and of what is taken, else on the first other side that does,
// else on its own side. A place the window has moved off the picture is
// not drawn. It says where its words went.
func (label Label) draw(frame *image.RGBA, part window, tones palette,
	taken []image.Rectangle) (image.Rectangle, error) {
	point := onFrame(label.at, part)
	if !point.In(pictureArea()) {
		return image.Rectangle{}, nil
	}
	wide, err := measure(label.words, labelSize)
	if err != nil {
		return image.Rectangle{}, err
	}
	box := labelBox(image.Pt(wide+2*labelPad, int(labelSize)+2*labelPad),
		point, label.side, taken)
	foot := nearestEdge(box, point)
	line(frame, foot, point, leaderWidth+2*dotRing, tones["ground"])
	line(frame, foot, point, leaderWidth, tones["ink"])
	if _, err := placedPlate(frame, label.words, box.Min, labelSize,
		tones); err != nil {
		return box, err
	}
	disc(frame, point, dotRadius+dotRing, tones["ground"])
	disc(frame, point, dotRadius, tones["ok"])
	return box, nil
}

// labelBox is where a label's words stand: its own side first, then the
// opposite, then the other two; the first kept inside the picture that
// clears its place's dot and every box taken.
func labelBox(size, point, side image.Point,
	taken []image.Rectangle) image.Rectangle {
	across := image.Pt(side.Y, side.X)
	tries := []image.Point{side, side.Mul(-1), across, across.Mul(-1)}
	var first image.Rectangle
	for i, way := range tries {
		centre := point.Add(way.Mul(leaderReach))
		box := keptInside(image.Rectangle{Max: size}.Add(
			centre.Sub(size.Div(2))), pictureArea())
		if i == 0 {
			first = box
		}
		if clear(box, point, taken) {
			return box
		}
	}
	return first
}

// clear is whether a box keeps off a place's dot and every box taken.
func clear(box image.Rectangle, point image.Point,
	taken []image.Rectangle) bool {
	dot := image.Rect(point.X, point.Y, point.X+1, point.Y+1).Inset(
		0 - dotRadius - dotRing - labelPad)
	return !box.Overlaps(dot) && clearOf(box, taken)
}

// nearestEdge is the point of a box's edge nearest a point outside it.
func nearestEdge(box image.Rectangle, point image.Point) image.Point {
	return image.Pt(max(box.Min.X, min(point.X, box.Max.X-1)),
		max(box.Min.Y, min(point.Y, box.Max.Y-1)))
}

// drawBar draws the shot's scale bar, if it has one, and says where its
// words went: measured at its place and centred on it, true there; or
// given outright, at the picture's foot on the left.
func (shot Shot) drawBar(frame *image.RGBA, part window,
	tones palette) (image.Rectangle, error) {
	scale := float64(pictureSize.X) / part.width
	var length int
	var words string
	var middle image.Point
	switch {
	case shot.bar != nil:
		bar, err := shot.view.ScaleBar(*shot.bar, barMostShare*part.width)
		if err != nil {
			return image.Rectangle{}, err
		}
		length, words = int(math.Round(bar.Pixels()*scale)), bar.Words()
		middle = onFrame(bar.At(), part)
	case shot.outright != nil:
		length = int(math.Round(shot.outright.pixels * scale))
		words = shot.outright.words
		middle = image.Pt(barInset+length/2,
			pictureTop+pictureSize.Y-barInset)
	default:
		return image.Rectangle{}, nil
	}
	if !middle.In(pictureArea()) {
		return image.Rectangle{}, nil
	}
	strip := image.Rect(middle.X-length/2, middle.Y-barHeight/2,
		middle.X-length/2+length, middle.Y+barHeight/2)
	fill(frame, strip.Inset(0-dotRing), tones["ground"])
	fill(frame, strip, tones["ink"])
	fill(frame, image.Rect(strip.Min.X+length/2, strip.Min.Y,
		strip.Max.X, strip.Max.Y), tones["ok"])
	wide, err := measure(words, barSize)
	if err != nil {
		return image.Rectangle{}, err
	}
	box, err := placedPlate(frame, words, image.Pt(middle.X-wide/2-labelPad,
		strip.Min.Y-int(barSize)-2*labelPad-barInset/2), barSize, tones)
	return box.Union(strip.Inset(0 - dotRing)), err
}

// panel draws the panel under the picture: the name of what is looked
// at, its figures, and the sentence.
func (shot Shot) panel(frame *image.RGBA, tones palette,
	hosted bool) error {
	if err := write(frame, shot.name, image.Pt(margin, nameBaseline),
		nameSize, tones["ink"]); err != nil {
		return err
	}
	if err := shot.drawFigures(frame, tones); err != nil {
		return err
	}
	rows, err := wrapped(shot.sentence, sentenceSize,
		sentenceWidth(hosted))
	if err != nil {
		return err
	}
	for i, row := range rows[:min(len(rows), sentenceRows)] {
		at := image.Pt(margin, sentenceTop+int(sentenceSize)+i*sentenceLead)
		if err := write(frame, row, at, sentenceSize,
			tones["ink"]); err != nil {
			return err
		}
	}
	return nil
}

// drawFigures draws the figures side by side: each a large number in
// the accent, its unit beside it, and its words under it.
func (shot Shot) drawFigures(frame *image.RGBA, tones palette) error {
	if len(shot.figures) == 0 {
		return nil
	}
	column := (frameSize.X - 2*margin) / len(shot.figures)
	baseline := figureTop + int(math.Round(figureSize*capShare))
	for i, figure := range shot.figures {
		left := margin + i*column
		if err := write(frame, figure.value, image.Pt(left, baseline),
			figureSize, tones["ok"]); err != nil {
			return err
		}
		wide, err := measure(figure.value, figureSize)
		if err != nil {
			return err
		}
		if err := write(frame, figure.unit, image.Pt(left+wide+unitGap,
			baseline), unitSize, tones["ink"]); err != nil {
			return err
		}
		if err := write(frame, figure.says, image.Pt(left,
			baseline+figureSaysDrop), saysSize, tones["ink"]); err != nil {
			return err
		}
	}
	return nil
}

// unitGap is the space between a figure and its unit, in pixels. Picked
// here.
const unitGap = 12

// Caption draws words in ink on a plate of ground at a point of any
// frame, as a label's words are drawn: for naming a pane.
func Caption(frame *image.RGBA, words string, at image.Point,
	size float64) error {
	tones, err := colours()
	if err != nil {
		return err
	}
	wide, err := measure(words, size)
	if err != nil {
		return err
	}
	box := image.Rect(0, 0, wide+2*labelPad, int(size)+2*labelPad).Add(at)
	fill(frame, box, tones["ground"])
	return write(frame, words, image.Pt(box.Min.X+labelPad,
		box.Min.Y+labelPad+int(size*capShare)), size, tones["ink"])
}

// Words draws words in a colour with their baseline's left end at a
// point, in the face the frame's words are set in, at a size in pixels.
func Words(frame *image.RGBA, words string, at image.Point, size float64,
	ink color.RGBA) error {
	return write(frame, words, at, size, ink)
}

// Wrapped is words broken into rows no wider than a width, at a size.
func Wrapped(words string, size float64, width int) ([]string, error) {
	return wrapped(words, size, width)
}

// Measure is how wide words are, in pixels, at a size.
func Measure(words string, size float64) (int, error) {
	return measure(words, size)
}

// Fill paints a rectangle of a frame one colour.
func Fill(frame *image.RGBA, area image.Rectangle, colour color.RGBA) {
	fill(frame, area, colour)
}

// Disc paints a filled circle.
func Disc(frame *image.RGBA, centre image.Point, radius int,
	colour color.RGBA) {
	disc(frame, centre, radius, colour)
}

// Mix is one colour toward another by a share from nought to one.
func Mix(from, to color.RGBA, share float64) color.RGBA {
	return mix(from, to, share)
}

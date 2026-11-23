package cities

import (
	"math"
)

// tallAt is how much of the height of the face stands at a place along
// it, from nought to one: all of it on a face that does not taper, and on
// one that does, less toward its ends, where the face has narrowed to its
// top and the plan has gone in.
func (job faceJob) tallAt(along float64) float64 {
	frame := job.frame
	if !(frame.topLength < frame.length) {
		return 1
	}
	return math.Max(0, math.Min(1, (frame.length/2-math.Abs(along))/
		((frame.length-frame.topLength)/2)))
}

// gothicSpan is a span of stone buttresses, each as tall as the face and
// so unbroken to the top, with a tall window between every two, a piece at
// each storey. The top segment of a shaft has pinnacles on its buttresses
// and tracery under its crown, and a setback leaves a ledge that holds a
// figure in every bay.
func (job faceJob) gothicSpan(low, high, pitch float64,
	columns int) []placedElement {
	drawing, stone := job.drawing, job.looks.skin.stone
	width, proud := drawing.fixed("gothic-pier-width"),
		drawing.fixed("gothic-pier-proud")
	lancet := drawing.fixed("gothic-lancet-width")
	parts := job.bars("buttress", low, pitch, columns, width, proud, stone)
	parts = append(parts, job.eachStorey(low, high, func(index int,
		foot, storey, left, right float64) []placedElement {
		return job.eachWindow(index, low, pitch, columns,
			func(column int, start, end float64) []placedElement {
				at := (start+end)/2 - lancet/2
				made := []placedElement{job.frame.onFace("lancet", Box{lancet, storey,
					job.pane}, job.looks.skin.window, at, foot,
					-job.recess)}
				return append(made, job.lit(index, column, at, lancet,
					foot+job.spandrel/2, storey-job.spandrel,
					-job.recess+job.pane)...)
			})
	})...)
	if job.last {
		parts = append(parts, job.pinnacles(low, pitch, columns)...)
		parts = append(parts, job.tracery(low, pitch, columns)...)
	}
	if job.ledge >= drawing.fixed("gothic-step-least") {
		parts = append(parts, job.figures(low, pitch, columns)...)
	}
	return parts
}

// pinnacles are the pinnacles on the tops of the buttresses of a span:
// a square pyramid on each, from its foot to its point.
func (job faceJob) pinnacles(low, pitch float64, columns int) []placedElement {
	drawing, frame := job.drawing, job.frame
	base, tip := drawing.fixed("gothic-pinnacle-base"),
		drawing.fixed("gothic-pinnacle-tip")
	form := prism{Bottom: polygon{boxCorners(base, base)},
		Top:    polygon{boxCorners(tip, tip)},
		Height: drawing.fixed("gothic-pinnacle-height")}
	out := (drawing.fixed("gothic-pier-proud") - job.recess) / 2
	var parts []placedElement
	for column := 0; column <= columns; column++ {
		along := low + float64(column)*pitch
		if tall := job.tallAt(along); tall > 0 {
			parts = append(parts, placedElement{Element: solid{Name: "pinnacle",
				Form: form, Material: job.looks.skin.stone},
				Place: frame.at(along, tall*frame.slope, out)})
		}
	}
	return parts
}

// tracery is the band of tracery under a crown: in each bay two slim
// lancets and a transom across them, in stone standing out of the face.
func (job faceJob) tracery(low, pitch float64, columns int) []placedElement {
	drawing, frame := job.drawing, job.frame
	deep := drawing.fixed("gothic-tracery-height")
	bar, depth := drawing.fixed("gothic-tracery-bar"),
		drawing.fixed("gothic-tracery-depth")
	lancet := drawing.fixed("gothic-lancet-width")
	if !(frame.slope > 2*deep) {
		return nil
	}
	var parts []placedElement
	for column := range columns {
		start := low + float64(column)*pitch
		if start+pitch > frame.half(1) || start < -frame.half(1) {
			continue
		}
		middle := start + pitch/2
		for _, side := range []float64{-1, 1} {
			parts = append(parts, frame.onFace("tracery", Box{bar, deep,
				job.recess + depth}, job.looks.skin.stone,
				middle+side*lancet/4-bar/2, frame.slope-deep, -job.recess))
		}
		parts = append(parts, frame.onFace("tracery", Box{lancet, bar,
			job.recess + depth}, job.looks.skin.stone, middle-lancet/2,
			frame.slope-deep+drawing.fixed("gothic-transom-share")*deep,
			-job.recess))
	}
	return parts
}

// figures are the figures on the ledge a setback leaves, one in the middle
// of every bay: a plinth, a body on it and a head on that, each a box.
func (job faceJob) figures(low, pitch float64, columns int) []placedElement {
	drawing, frame := job.drawing, job.frame
	plinth, rise := drawing.fixed("gothic-plinth"),
		drawing.fixed("gothic-plinth-height")
	wide, tall := drawing.fixed("gothic-figure-width"),
		drawing.fixed("gothic-figure-height")
	deep, head := drawing.fixed("gothic-figure-depth"),
		drawing.fixed("gothic-figure-head")
	ledge := drawing.fixed("gothic-figure-ledge")
	stone := job.looks.skin.figure
	var parts []placedElement
	for column := range columns {
		middle := low + (float64(column)+0.5)*pitch
		if math.Abs(middle)+plinth > frame.half(0) {
			continue
		}
		parts = append(parts,
			frame.onFace("plinth", Box{plinth, rise, plinth}, stone,
				middle-plinth/2, 0, ledge),
			frame.onFace("figure", Box{wide, tall, deep}, stone,
				middle-wide/2, rise, ledge+(plinth-deep)/2),
			frame.onFace("figure head", Box{head, head, head}, stone,
				middle-head/2, rise+tall, ledge+(plinth-head)/2))
	}
	return parts
}

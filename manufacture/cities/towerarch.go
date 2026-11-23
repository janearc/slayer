package cities

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// archAngle is the angle in degrees at the apex of an equilateral
// arch between the radius to the springing and the vertical: an
// equilateral arch is struck from the opposite springing with a radius of
// its own width, so its arcs end sixty degrees from the horizontal.
const archAngle = 60.0

// archCorners are the corners of a pointed equilateral arch of a width, in
// the plane of the face with x along it and z down it, so that the arch
// rises to negative z: the springing line at nought, the two arcs cut into
// the number of pieces given.
func archCorners(width float64, pieces int) []outlineCorner {
	var corners []outlineCorner
	for step := 0; step <= pieces; step++ {
		angle := (halfTurn - archAngle*float64(step)/float64(pieces)) *
			radiansPerDegree
		corners = append(corners, outlineCorner{width/2 + width*math.Cos(angle),
			-width * math.Sin(angle)})
	}
	for step := 1; step <= pieces; step++ {
		angle := (archAngle - archAngle*float64(step)/float64(pieces)) *
			radiansPerDegree
		corners = append(corners, outlineCorner{-width/2 + width*math.Cos(angle),
			-width * math.Sin(angle)})
	}
	return corners
}

// facing is the place standing on the face at a place along and up it,
// turned so that a prism of a plan drawn in x and z rises out of the face,
// the plan's z running down the face.
func (frame faceFrame) facing(along, up, out float64) maths.Subspace {
	point := frame.origin.Add(frame.along.Scale(along)).
		Add(frame.up.Scale(up)).Add(frame.out.Scale(out))
	return maths.NewSubspace(point, maths.Orientation{X: frame.along,
		Y: frame.out, Z: frame.up.Neg()}, 0)
}

// arch is a pointed arch of a width standing on the face with its
// springing line at a height, its middle at a place along, reaching from
// out of the face to out plus depth.
func (job faceJob) arch(name string, width, depth float64,
	look *thing.Material, along, up, out float64) placedElement {
	corners := archCorners(width, int(job.drawing.fixed("gt-arch-sides")))
	return placedElement{Element: solid{Name: name, Form: prism{
		Bottom: polygon{corners}, Top: polygon{corners}, Height: depth},
		Material: look}, Place: job.frame.facing(along, up, out)}
}

// gothicFine is a span of gothic at the demo level: buttresses stepped out
// at their feet, and in every bay of every storey a spandrel panel with a
// blind tracery under a lancet that ends in a pointed arch in a stone hood.
// The top segment has pinnacles on shafts, tracery under its crown, and a
// figure on every ledge, as the quick level draws them.
func (job faceJob) gothicFine(low, high, pitch float64,
	columns int) []placedElement {
	drawing, stone := job.drawing, job.looks.skin.stone
	parts := job.bars("buttress", low, pitch, columns,
		drawing.fixed("gothic-pier-width"),
		drawing.fixed("gothic-pier-proud"), stone)
	parts = append(parts, job.offsets(low, pitch, columns)...)
	parts = append(parts, job.eachStorey(low, high, func(index int,
		foot, storey, left, right float64) []placedElement {
		floor := job.floorSlab(left, right, foot)
		return append(floor, job.eachWindow(index, low, pitch, columns,
			func(column int, start, end float64) []placedElement {
				return job.gothicBay(index, column, (start+end)/2, foot,
					storey, pitch)
			})...)
	})...)
	if job.last {
		parts = append(parts, job.shafts(low, pitch, columns)...)
		parts = append(parts, job.tracery(low, pitch, columns)...)
	}
	if job.ledge >= drawing.fixed("gothic-step-least") {
		parts = append(parts, job.figures(low, pitch, columns)...)
	}
	return parts
}

// offsets are the steps at the foot of each buttress of the first storeys
// of a face: each wider and further out than the one above it.
func (job faceJob) offsets(low, pitch float64, columns int) []placedElement {
	drawing, stone := job.drawing, job.looks.skin.stone
	if job.firstStorey > 0 {
		return nil
	}
	width, proud := drawing.fixed("gothic-pier-width"),
		drawing.fixed("gothic-pier-proud")
	step, rise := drawing.fixed("gt-offset-step"),
		drawing.fixed("gt-offset-height")
	var parts []placedElement
	for level := range int(drawing.fixed("gt-offset-count")) {
		grown := float64(int(drawing.fixed("gt-offset-count"))-level) * step
		for column := 0; column <= columns; column++ {
			along := low + float64(column)*pitch
			if math.Abs(along)+width/2+grown > job.frame.half(0) {
				continue
			}
			parts = append(parts, job.frame.onFace("offset", Box{width +
				2*grown, rise, job.deep() + proud + grown}, stone,
				along-width/2-grown, float64(level)*rise, -job.deep()))
		}
	}
	return parts
}

// gothicBay is one bay of one storey, its middle at a place along the
// face and the floor foot up it: the spandrel panel with its tracery, the
// lancet and the arch over it, in its hood.
func (job faceJob) gothicBay(index, column int, middle, foot,
	storey, pitch float64) []placedElement {
	drawing, stone := job.drawing, job.looks.skin.stone
	lancet := drawing.fixed("gothic-lancet-width")
	panel := drawing.fixed("gt-panel-height")
	hood, deep := drawing.fixed("gt-hood"), drawing.fixed("gt-arch-deep")
	spring := foot + storey - lancet*math.Sin(archAngle*radiansPerDegree)
	glass := -job.recess + deep
	parts := job.spandrelTracery(middle, foot, panel)
	parts = append(parts, job.jambs(middle, foot, storey, pitch)...)
	parts = append(parts, job.frame.onFace("lancet", Box{lancet,
		spring - foot - panel, job.pane}, job.looks.fine.glazing,
		middle-lancet/2, foot+panel, glass),
		job.arch("hood", lancet+2*hood, job.cavity+deep, stone, middle,
			spring, -job.deep()),
		job.arch("lancet head", lancet, job.pane, job.looks.skin.window,
			middle, spring, glass))
	return append(parts, job.room(index, column, middle-lancet/2,
		middle+lancet/2, foot, foot+panel, spring-foot-panel)...)
}

// jambs are the stone on either side of a lancet, from the lancet's edge
// to the inside of the buttress beside it, a storey high and standing out
// a little past the hood, so the window is set into the wall.
func (job faceJob) jambs(middle, foot, storey, pitch float64) []placedElement {
	drawing, stone := job.drawing, job.looks.skin.stone
	lancet := drawing.fixed("gothic-lancet-width")
	width := (pitch-lancet)/2 - drawing.fixed("gothic-pier-width")/2
	if !(width > 0) {
		return nil
	}
	size := Box{width, storey, job.deep() + drawing.fixed("gt-arch-deep")}
	return []placedElement{
		job.frame.onFace("jamb", size, stone, middle-lancet/2-width, foot,
			-job.deep()),
		job.frame.onFace("jamb", size, stone, middle+lancet/2, foot,
			-job.deep())}
}

// spandrelTracery is the spandrel panel at the foot of a lancet, a block of
// stone standing out of the face, with a lozenge of tracery on it.
func (job faceJob) spandrelTracery(middle, foot, panel float64) []placedElement {
	drawing, stone := job.drawing, job.looks.skin.stone
	wide := drawing.fixed("gothic-lancet-width")
	proud := drawing.fixed("gt-panel-proud")
	foil, stand := drawing.fixed("gt-foil"), drawing.fixed("gt-foil-proud")
	lozenge := []outlineCorner{{-foil / 2, 0}, {0, -foil / 2}, {foil / 2, 0},
		{0, foil / 2}}
	return []placedElement{
		job.frame.onFace("spandrel panel", Box{wide, panel, job.deep() +
			proud}, stone, middle-wide/2, foot, -job.deep()),
		{Element: solid{Name: "tracery lozenge", Form: prism{
			Bottom: polygon{lozenge}, Top: polygon{lozenge}, Height: stand},
			Material: stone}, Place: job.frame.facing(middle,
			foot+panel/2, proud)}}
}

// shafts are the pinnacles of a span with the shaft each stands on: a
// square shaft up from the top of the buttress, the pyramid on it and a
// finial on its point.
func (job faceJob) shafts(low, pitch float64, columns int) []placedElement {
	drawing, frame, stone := job.drawing, job.frame, job.looks.skin.stone
	side, high := drawing.fixed("gt-shaft-side"),
		drawing.fixed("gt-shaft-height")
	base, tip := drawing.fixed("gothic-pinnacle-base"),
		drawing.fixed("gothic-pinnacle-tip")
	finial := drawing.fixed("gt-finial")
	out := (drawing.fixed("gothic-pier-proud") - job.recess) / 2
	cone := prism{Bottom: polygon{boxCorners(base, base)},
		Top:    polygon{boxCorners(tip, tip)},
		Height: drawing.fixed("gothic-pinnacle-height")}
	var parts []placedElement
	for column := 0; column <= columns; column++ {
		along := low + float64(column)*pitch
		tall := job.tallAt(along)
		if !(tall > 0) {
			continue
		}
		top := tall * frame.slope
		parts = append(parts,
			frame.onFace("pinnacle shaft", Box{side, high, side}, stone,
				along-side/2, top, out-side/2),
			placedElement{Element: solid{Name: "pinnacle", Form: cone,
				Material: stone}, Place: frame.at(along, top+high, out)},
			frame.onFace("finial", Box{finial, finial, finial}, stone,
				along-finial/2, top+high+cone.Height, out-finial/2))
	}
	return parts
}

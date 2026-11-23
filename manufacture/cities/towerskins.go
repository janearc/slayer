package cities

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// radiansPerDegree is how many radians are in a degree.
const radiansPerDegree = math.Pi / halfTurn

// windowLight is the light in the window of a column at a storey of the
// shaft, counted within the segment, and whether it is on. An office lights
// whole floors: every window of a floor the culture lights, together, in
// one cool light. A home lights each window by itself, in a warm light of
// its own temperature. A plant floor is never lit: its callers skip it.
func (job faceJob) windowLight(index, column int) (*thing.Material, bool) {
	drawing, global := job.drawing, job.firstStorey+index
	if job.use == useOffice {
		floor := place{Bay: job.shaft, Storey: global}
		if !drawing.chance("office-lit", floor) {
			return nil, false
		}
		return drawing.lamp(drawing.fixed("office-heat"),
			drawing.fixed("office-glow")), true
	}
	at := place{Bay: job.shaft, Storey: global,
		Opening: job.face*faceStride + column}
	if !drawing.chance("home-lit", at) {
		return nil, false
	}
	step := drawing.fixed("light-heat-step")
	heat := math.Round(drawing.number("home-heat", at)/step) * step
	return drawing.lamp(heat, drawing.fixed("home-glow")), true
}

// lit is the pane of a window that is alight, and its curtain if it has
// one: the window's place and size on the face, along from the middle and
// up the slope, out of the face by out. A window that is dark has nothing.
func (job faceJob) lit(index, column int, along, width, up, height,
	out float64) []placedElement {
	look, on := job.windowLight(index, column)
	if !on || look == nil {
		return nil
	}
	parts := []placedElement{job.frame.onFace("pane", Box{width, height, job.pane},
		look, along, up, out)}
	return append(parts, job.curtain(index, column, along, width, up,
		height, out+job.pane)...)
}

// curtain is the curtain across part of a lit window of a home, if the
// culture gives that window one. Offices have none.
func (job faceJob) curtain(index, column int, along, width, up, height,
	out float64) []placedElement {
	drawing := job.drawing
	at := place{Bay: job.shaft, Storey: job.firstStorey + index,
		Opening: job.face*faceStride + column}
	if job.use != useHome || !drawing.chance("curtain", at) {
		return nil
	}
	cloth := paint("curtain", drawing.pick("curtain-colour",
		drawing.options("curtain-colour"), at))
	cover := width * drawing.number("curtain-cover", at)
	return []placedElement{job.frame.onFace("curtain", Box{cover, height,
		drawing.fixed("curtain-thickness")}, cloth, along, up, out)}
}

// bar is a vertical bar standing out of the face by proud, from the
// body behind it, centred on a place along the face and as tall as the
// face is there: on a face that tapers, a bar near its end is shorter, and
// one past the end of the top is nothing.
func (job faceJob) bar(name string, along, width, proud float64,
	look *thing.Material) (placedElement, bool) {
	frame, tall := job.frame, 1.0
	if frame.topLength < frame.length {
		tall = math.Min(1, (frame.length/2-math.Abs(along))/
			((frame.length-frame.topLength)/2))
	}
	if !(tall > 0) {
		return placedElement{}, false
	}
	return frame.onFace(name, Box{width, tall * frame.slope,
		job.deep() + proud}, look, along-width/2, 0, -job.deep()), true
}

// bars are the bars of a name at each edge of each column of a span,
// from the first to the last.
func (job faceJob) bars(name string, low, pitch float64, columns int,
	width, proud float64, look *thing.Material) []placedElement {
	var found []placedElement
	for column := 0; column <= columns; column++ {
		if one, ok := job.bar(name, low+float64(column)*pitch, width, proud,
			look); ok {
			found = append(found, one)
		}
	}
	return found
}

// eachStorey is what draw makes for each storey of a span that is not a
// plant floor, with the left and right of the span at that storey, which
// a face that tapers makes narrower higher up. A plant floor is the dark
// plant, louvred, and nothing else.
func (job faceJob) eachStorey(low, high float64, draw func(index int,
	foot, storey, left, right float64) []placedElement) []placedElement {
	var parts []placedElement
	frame := job.frame
	storey := frame.slope / float64(job.storeys)
	for index := range job.storeys {
		half := frame.half((float64(index) + 0.5) / float64(job.storeys))
		left, right := math.Max(low, -half), math.Min(high, half)
		if !(right > left) {
			continue
		}
		foot := float64(index) * storey
		if job.demo && job.firstStorey+index == 0 {
			continue
		}
		if job.mechanical(job.firstStorey + index) {
			parts = append(parts, job.plant(left, right, foot, storey)...)
			continue
		}
		parts = append(parts, draw(index, foot, storey, left, right)...)
	}
	return parts
}

// eachWindow is what draw makes for each column of a span at a storey,
// given the column's bay from its start to its end along the face. A bay
// that would stand off the face at the top of the storey is left out.
func (job faceJob) eachWindow(index int, low, pitch float64, columns int,
	draw func(column int, start, end float64) []placedElement) []placedElement {
	half := job.frame.half((float64(index) + 1) / float64(job.storeys))
	var parts []placedElement
	for column := range columns {
		start := low + float64(column)*pitch
		if start+pitch > half || start < -half {
			continue
		}
		parts = append(parts, draw(column, start, start+pitch)...)
	}
	return parts
}

// mirrorSpan is a span of mirror glass curtain wall: caps on the mullions,
// an opaque band at each slab, and in each window a unitized panel of
// mirror, a little out of true so that the wall reflects in waves. A window
// that is alight shows its light in place of its panel.
func (job faceJob) mirrorSpan(low, high, pitch float64,
	columns int) []placedElement {
	drawing, skin := job.drawing, job.looks.skin
	cap, proud := drawing.fixed("mirror-cap-width"),
		drawing.fixed("mirror-cap-proud")
	band, standing := drawing.fixed("mirror-spandrel-height"),
		drawing.fixed("mirror-spandrel-proud")
	parts := job.bars("mullion cap", low, pitch, columns, cap, proud,
		skin.cap)
	return append(parts, job.eachStorey(low, high, func(index int,
		foot, storey, left, right float64) []placedElement {
		made := []placedElement{job.frame.onFace("spandrel", Box{right - left, band,
			job.recess + standing}, skin.spandrel, left, foot, -job.recess)}
		return append(made, job.eachWindow(index, low, pitch, columns,
			func(column int, start, end float64) []placedElement {
				start, end = start+cap/2, end-cap/2
				if lit := job.lit(index, column, start, end-start,
					foot+band, storey-band, -job.recess); lit != nil {
					return lit
				}
				return []placedElement{job.panel(index, column, start, end,
					foot+band, storey-band)}
			})...)
	})...)
}

// panel is a unitized glass panel of a mirror wall, between two places
// along the face, standing on the body, leaned a little about the way
// along the face and the way up it, by the draws of its place.
func (job faceJob) panel(index, column int, start, end, up,
	height float64) placedElement {
	drawing, frame := job.drawing, job.frame
	at := place{Bay: job.shaft, Storey: job.firstStorey + index,
		Opening: job.face*faceStride + column}
	other := at
	other.Row = 1
	each, _ := drawing.found("mirror-wave", drawRange)
	middle := (each.Least + each.Most) / 2
	across := math.Sin((drawing.number("mirror-wave", at) - middle) *
		radiansPerDegree)
	upward := math.Sin((drawing.number("mirror-wave", other) - middle) *
		radiansPerDegree)
	along := frame.along.Add(frame.out.Scale(across)).Unit()
	rise := frame.up.Add(frame.out.Scale(upward)).Unit()
	out := along.Cross(rise).Unit()
	rise = out.Cross(along).Unit()
	origin := frame.at(start, up, -job.recess).Origin
	look := job.looks.skin.panel
	if job.demo && job.looks.fine.mirror != nil {
		look = job.looks.fine.mirror
	}
	return placedElement{Element: solid{Name: "panel", Form: Box{end - start,
		height, drawing.fixed("mirror-panel-depth")},
		Material: look}, Place: maths.NewSubspace(origin,
		maths.Orientation{X: along, Y: rise, Z: out}, 0)}
}

// punchedSpan is a span of stone with windows punched in it: a pier of
// stone between the windows standing out of the wall, a window of glass
// and a sill under it in every bay of every storey.
func (job faceJob) punchedSpan(low, high, pitch float64,
	columns int) []placedElement {
	drawing, stone := job.drawing, job.looks.skin.stone
	wide, tall := drawing.fixed("granite-window-width"),
		drawing.fixed("granite-window-height")
	sill, lift := drawing.fixed("granite-sill"),
		drawing.fixed("granite-sill-height")
	parts := job.bars("pier", low, pitch, columns, pitch-wide,
		drawing.fixed("granite-pier-proud"), stone)
	return append(parts, job.eachStorey(low, high, func(index int,
		foot, storey, left, right float64) []placedElement {
		return job.eachWindow(index, low, pitch, columns,
			func(column int, start, end float64) []placedElement {
				at := (start+end)/2 - wide/2
				parts := []placedElement{job.frame.onFace("sill", Box{wide, lift,
					job.recess + drawing.fixed("granite-sill-proud")}, stone,
					at, foot+sill-lift, -job.recess)}
				if lit := job.lit(index, column, at, wide, foot+sill, tall,
					-job.recess); lit != nil {
					return append(parts, lit...)
				}
				return append(parts, job.frame.onFace("window", Box{wide,
					tall, job.pane}, job.looks.skin.window, at, foot+sill,
					-job.recess))
			})
	})...)
}

// ribbonSpan is a span with a ribbon of glass at every storey between
// bands of the wall: stone or steel, standing out of the face. A steel wall
// also has a fin over every ribbon. A ribbon is dark, or lit where its
// windows are: a floor at a time in an office.
func (job faceJob) ribbonSpan(low, high, pitch float64,
	columns int) []placedElement {
	drawing, stone := job.drawing, job.looks.skin.stone
	steel := job.skin == skinStainless
	sill, wide := drawing.fixed("granite-ribbon-sill"),
		drawing.fixed("granite-ribbon-height")
	proud := drawing.fixed("granite-band-proud")
	if steel {
		sill, wide = drawing.fixed("stainless-window-sill"),
			drawing.fixed("stainless-window-height")
	}
	return job.eachStorey(low, high, func(index int, foot, storey, left,
		right float64) []placedElement {
		parts := []placedElement{
			job.frame.onFace("stone band", Box{right - left, sill,
				job.recess + proud}, stone, left, foot, -job.recess),
			job.frame.onFace("ribbon window", Box{right - left, wide,
				job.pane}, job.looks.skin.window, left, foot+sill,
				-job.recess)}
		if steel {
			parts = append(parts, job.fin(left, right, foot+sill+wide))
		}
		return append(parts, job.eachWindow(index, low, pitch, columns,
			func(column int, start, end float64) []placedElement {
				return job.lit(index, column, start, end-start, foot+sill,
					wide, -job.recess+job.pane)
			})...)
	})
}

// fin is the horizontal fin of a steel wall, standing out of the face
// at the head of a window over the width of a span.
func (job faceJob) fin(left, right, up float64) placedElement {
	drawing := job.drawing
	return job.frame.onFace("fin", Box{right - left,
		drawing.fixed("stainless-fin-height"), job.recess +
			drawing.fixed("stainless-fin-proud")}, job.looks.skin.stone,
		left, up, -job.recess)
}

// concreteSpan is a span of board-marked concrete: a slab edge expressed
// at every floor, marked with the grain of its boards, a pier between the
// windows standing out of the wall, and a window set deep between them.
func (job faceJob) concreteSpan(low, high, pitch float64,
	columns int) []placedElement {
	drawing, skin := job.drawing, job.looks.skin
	slab := drawing.fixed("concrete-slab-thick")
	width := drawing.fixed("concrete-pier-width")
	parts := job.bars("pier", low, pitch, columns, width,
		drawing.fixed("concrete-pier-proud"), skin.stone)
	return append(parts, job.eachStorey(low, high, func(index int,
		foot, storey, left, right float64) []placedElement {
		made := job.slabEdge(left, right, foot)
		return append(made, job.eachWindow(index, low, pitch, columns,
			func(column int, start, end float64) []placedElement {
				start, end = start+width/2, end-width/2
				if lit := job.lit(index, column, start, end-start,
					foot+slab, storey-slab, -job.recess); lit != nil {
					return lit
				}
				return []placedElement{job.frame.onFace("window", Box{end - start,
					storey - slab, job.pane}, job.looks.skin.window, start,
					foot+slab, -job.recess)}
			})...)
	})...)
}

// slabEdge is the edge of a floor slab expressed on a face of concrete,
// from left to right at a height, with a few thin marks across it in the
// colour of the shadow in the grain of the boards.
func (job faceJob) slabEdge(left, right, foot float64) []placedElement {
	drawing, skin := job.drawing, job.looks.skin
	thick, proud := drawing.fixed("concrete-slab-thick"),
		drawing.fixed("concrete-slab-proud")
	parts := []placedElement{job.frame.onFace("slab edge", Box{right - left, thick,
		job.recess + proud}, skin.stone, left, foot, -job.recess)}
	lines := drawing.count("concrete-boards", place{})
	mark := drawing.fixed("concrete-board-mark")
	for line := range lines {
		up := foot + float64(line+1)*thick/float64(lines+1) - mark/2
		parts = append(parts, job.frame.onFace("board mark", Box{
			right - left, mark, mark}, skin.board, left, up, proud))
	}
	return parts
}

// cantileverSpan is a span of glass with a floor plate thrown out of it
// at every storey, each as far as its place draws, and a cable net across
// the glass: two cables crossing every bay, from a storey to the second
// one up. The windows are the glass of the body.
func (job faceJob) cantileverSpan(low, high, pitch float64,
	columns int) []placedElement {
	drawing, skin := job.drawing, job.looks.skin
	thick := drawing.fixed("cantilever-thick")
	parts := job.eachStorey(low, high, func(index int, foot, storey, left,
		right float64) []placedElement {
		at := place{Bay: job.shaft, Storey: job.firstStorey + index,
			Opening: job.face * faceStride}
		reach := drawing.number("cantilever-reach", at)
		return []placedElement{job.frame.onFace("floor plate", Box{right - left,
			thick, job.recess + reach}, skin.plate, left, foot,
			-job.recess)}
	})
	parts = append(parts, job.cables(low, pitch, columns)...)
	return append(parts, job.panes(low, pitch, columns)...)
}

// cables are the cables of a cable net over a span: in each bay and each
// run of storeys, two cables from corner to corner, crossing.
func (job faceJob) cables(low, pitch float64, columns int) []placedElement {
	drawing, frame := job.drawing, job.frame
	rise := int(drawing.fixed("cantilever-cable-storeys"))
	storey := frame.slope / float64(job.storeys)
	proud := drawing.fixed("cantilever-cable-proud")
	var parts []placedElement
	for index := 0; index+rise <= job.storeys; index += rise {
		half := frame.half(float64(index+rise) / float64(job.storeys))
		for column := range columns {
			start := low + float64(column)*pitch
			if start+pitch > half || start < -half {
				continue
			}
			foot, head := float64(index)*storey, float64(index+rise)*storey
			parts = append(parts,
				job.cable(start, foot, start+pitch, head, proud),
				job.cable(start+pitch, foot, start, head, proud))
		}
	}
	return parts
}

// cable is one cable from a place on the face to another, each along
// the face and up its slope, standing out of the glass by proud.
func (job faceJob) cable(fromAlong, fromUp, toAlong, toUp,
	proud float64) placedElement {
	frame := job.frame
	run := frame.along.Scale(toAlong - fromAlong).Add(
		frame.up.Scale(toUp - fromUp))
	along := run.Unit()
	rise := frame.out.Cross(along).Unit()
	thick := job.drawing.fixed("cantilever-cable")
	origin := frame.at(fromAlong, fromUp, proud).Origin
	return placedElement{Element: solid{Name: "cable", Form: Box{run.Len(), thick,
		thick}, Material: job.looks.skin.cable}, Place: maths.NewSubspace(
		origin, maths.Orientation{X: along, Y: rise, Z: frame.out}, 0)}
}

// vaporSpan is a span of pastel: a band at each slab, in the colour of
// the gradient at that height, glass block in the windows in a paler one,
// and marble bars across the block, up between the bays and along them.
func (job faceJob) vaporSpan(low, high, pitch float64,
	columns int) []placedElement {
	drawing, skin := job.drawing, job.looks.skin
	band, proud := drawing.fixed("vapor-band-height"),
		drawing.fixed("vapor-band-proud")
	width := drawing.fixed("vapor-bar")
	across := drawing.count("vapor-bars", place{})
	parts := job.bars("block bar", low, pitch, columns, width, proud,
		skin.stone)
	return append(parts, job.eachStorey(low, high, func(index int,
		foot, storey, left, right float64) []placedElement {
		step := min(len(skin.steps)-1,
			(job.firstStorey+index)*len(skin.steps)/job.shaftStoreys)
		made := []placedElement{job.frame.onFace("pastel band", Box{right - left,
			band, job.recess + proud}, skin.steps[step], left, foot,
			-job.recess)}
		for bar := range across {
			up := foot + band + float64(bar+1)*(storey-band)/
				float64(across+1)
			made = append(made, job.frame.onFace("block bar", Box{
				right - left, width, job.recess + proud}, skin.stone, left,
				up, -job.recess))
		}
		return append(made, job.eachWindow(index, low, pitch, columns,
			func(column int, start, end float64) []placedElement {
				start, end = start+width/2, end-width/2
				if lit := job.lit(index, column, start, end-start,
					foot+band, storey-band, -job.recess); lit != nil {
					return lit
				}
				return []placedElement{job.frame.onFace("glass block", Box{
					end - start, storey - band, job.pane}, skin.blocks[step],
					start, foot+band, -job.recess)}
			})...)
	})...)
}

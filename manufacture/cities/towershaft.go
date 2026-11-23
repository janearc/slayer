package cities

import (
	"fmt"
	"math"
)

// baseShape is the plan of a tower at its foot, drawn from the culture:
// every measure of every plan, of which a plan uses the ones it needs.
func (tower *Tower) baseShape(plan Plan) footprintShape {
	drawing, at := tower.drawing, place{}
	width := drawing.number("plan-width", at) * tower.scale
	if tower.width > 0 {
		width = tower.width * tower.scale
	}
	depth := width * drawing.number("plan-aspect", at)
	shape := footprintShape{Plan: plan, Width: width, Depth: depth,
		Steps: int(drawing.fixed("arc-steps")),
		Round: min(width, depth) * drawing.number("round-share", at),
		Sides: drawing.count("polygon-sides", at),
		Bow:   width * drawing.number("sail-bow-share", at)}
	switch plan {
	case planEllipse:
		shape.Sides = int(drawing.fixed("ellipse-sides"))
	case planLobed:
		shape.Width *= drawing.number("star-share", at)
		shape.Lobe = shape.Width * drawing.fixed("lobe-share")
		shape.Steps = int(drawing.fixed("lobe-steps"))
	case planStar:
		shape.Width *= drawing.number("star-share", at)
	case planY:
		length := width * drawing.number("wing-share", at)
		shape.Wings = [wingCount]float64{length, length, length}
		shape.WingWidth = width * drawing.number("wing-width-share", at)
	}
	return shape
}

// segmentStoreys divides a shaft's storeys between its segments, by the
// weights the culture draws for them, each getting at least the least a
// segment may have, the last taking what is left.
func (tower *Tower) segmentStoreys(at place, segments, storeys int) []int {
	drawing := tower.drawing
	least := int(drawing.fixed("least-segment-storeys"))
	weights := make([]float64, segments)
	var total float64
	for index := range weights {
		at.Storey = index
		weights[index] = drawing.number("segment-weight", at)
		total += weights[index]
	}
	spare := storeys - least*segments
	shares := make([]int, segments)
	left := storeys
	for index := range shares {
		shares[index] = least + int(float64(spare)*weights[index]/total)
		left -= shares[index]
	}
	shares[segments-1] += left
	return shares
}

// shaft draws one shaft of storeys floors: its segments, and what is on
// its top. A segment may taper, may step back from the one below, and may
// turn on it; the pins, where given, say yes or no for all of them.
func (tower *Tower) shaft(at place, pins Pins, storeys int) (shaft, error) {
	drawing, reach := tower.drawing, tower.Irregularity
	least := int(drawing.fixed("least-segment-storeys"))
	segments := max(1, min(drawing.reach("segments", at, reach),
		storeys/least))
	if pins.Segments > 0 {
		segments = max(1, min(pins.Segments, storeys/least))
	}
	if storeys < least {
		return shaft{}, fmt.Errorf("cities: a shaft of %d storeys is "+
			"under the least segment of %d", storeys, least)
	}
	shaft := shaft{Storeys: storeys}
	if shaft.Stacked = drawing.gate("stack", at, reach, pins.Stack); shaft.Stacked {
		shaft.Segments = tower.stacked(at, storeys)
	} else {
		shaft.Segments = tower.segmented(at, pins, segments, storeys)
	}
	if tower.Span != SpanDeck && drawing.gate("nest", at, reach, pins.Nest) {
		shaft.Nested = true
		tower.nested(&shaft, at)
	}
	tower.used(&shaft, at, pins)
	tower.crowned(&shaft, at, pins)
	return shaft, drawing.trouble
}

// segmented is the shaft's segments in series: each may taper, may step
// back from the one below, and may turn on it.
func (tower *Tower) segmented(at place, pins Pins, segments,
	storeys int) []segment {
	drawing, reach := tower.drawing, tower.Irregularity
	if pins.Spiral != nil && *pins.Spiral {
		return tower.spiral(storeys)
	}
	var made []segment
	bottom := tower.baseShape(tower.Plan)
	for index, count := range tower.segmentStoreys(at, segments, storeys) {
		at.Storey = index
		segment := segment{Bottom: bottom, Top: bottom, Storeys: count}
		if drawing.gate("taper", at, reach, pins.Taper) {
			segment.Top = tower.shrunk(bottom, drawing.number("taper-scale",
				at))
		}
		if drawing.gate("twist", at, reach, pins.Twist) {
			segment.Turn = drawing.number("twist-angle", at)
		}
		made = append(made, segment)
		bottom = tower.stepped(at, pins, index, segment.Top)
	}
	return made
}

// stacked is a stack of modules, one segment repeated upward, each narrow
// at its foot and flaring out to the full plan at its top, so that the top
// outline of each is larger than its bottom and each steps in from the one
// below. The storeys are shared out evenly.
func (tower *Tower) stacked(at place, storeys int) []segment {
	drawing := tower.drawing
	least := int(drawing.fixed("least-segment-storeys"))
	modules := max(2, min(drawing.count("stack-modules", at), storeys/least))
	full := tower.baseShape(tower.Plan)
	neck := full.scaled(drawing.number("stack-neck", at))
	made := make([]segment, modules)
	left := storeys
	for index := range made {
		count := storeys / modules
		if index == modules-1 {
			count = left
		}
		made[index] = segment{Bottom: neck, Top: full, Storeys: count}
		left -= count
	}
	return made
}

// nested carries a slim core on above the body of a shaft: a segment as
// the plan of the body's top scaled down, standing on it, for more storeys.
// The body is wrapped round the core's foot.
func (tower *Tower) nested(shaft *shaft, at place) {
	drawing := tower.drawing
	top := shaft.Segments[len(shaft.Segments)-1].Top
	core := top.scaled(drawing.number("nest-core-share", at))
	extra := drawing.count("nest-extra-storeys", at)
	shaft.Segments = append(shaft.Segments, segment{Bottom: core, Top: core,
		Storeys: extra})
	shaft.Storeys += extra
}

// stepped is the bottom of the next segment: the top of this one, or, if
// the segment above steps back, a smaller one. A Y steps back one wing at
// a time, round its core, which is what makes a spiral; any other plan
// steps back all round.
func (tower *Tower) stepped(at place, pins Pins, index int,
	top footprintShape) footprintShape {
	drawing := tower.drawing
	if !drawing.gate("setback", at, tower.Irregularity, pins.Setbacks) {
		return top
	}
	if top.Plan == planY {
		wing := index % wingCount
		stepped := top.Wings[wing] * drawing.number("wing-step", at)
		if stepped >= tower.leastRadius() {
			top.Wings[wing] = stepped
		}
		return top
	}
	return tower.shrunk(top, drawing.number("setback-scale", at))
}

// leastRadius is how small a tower's plan may become by tapers and
// setbacks: the culture's share of the radius of the plan at its foot.
func (tower *Tower) leastRadius() float64 {
	return tower.baseShape(tower.Plan).radius() *
		tower.drawing.fixed("least-plan-share")
}

// shrunk is a shape scaled down, but not past the least radius a plan
// may have, so that a run of tapers and setbacks never closes a tower up.
func (tower *Tower) shrunk(shape footprintShape, scale float64) footprintShape {
	if smaller := shape.scaled(scale); smaller.radius() >=
		tower.leastRadius() {
		return smaller
	}
	return shape.scaled(math.Max(tower.leastRadius()/shape.radius(), scale))
}

// crowned is what is on a shaft's top, drawn: a crown, or a penthouse in
// its place, and over them, if the culture and the irregularity give them,
// an observation deck and a spire.
func (tower *Tower) crowned(shaft *shaft, at place, pins Pins) {
	drawing, reach := tower.drawing, tower.Irregularity
	shaft.Penthouse = drawing.gate("penthouse", at, reach, nil)
	shaft.Crown = !shaft.Penthouse
	shaft.Deck = drawing.gate("observation-deck", at, reach, nil)
	shaft.Spire = drawing.gate("spire", at, reach, pins.Spire)
	shaft.LitCrown = shaft.Crown &&
		drawing.gate("crown-lit", at, reach, pins.LitCrown)
}

// place gives each shaft its centre: side by side along x, each a gap
// from the next, the whole centred on the origin. A shaft takes the room
// of the circle round its widest plan, so shafts never overlap however
// they turn.
func (tower *Tower) place() {
	radii := make([]float64, len(tower.Shafts))
	var total float64
	for index, shaft := range tower.Shafts {
		radii[index] = shaft.radius()
		total += 2 * radii[index]
		if index > 0 {
			total += tower.gap(index)
		}
	}
	along := -total / 2
	for index := range tower.Shafts {
		along += radii[index]
		tower.Shafts[index].Centre = outlineCorner{X: along}
		along += radii[index] + tower.gap(index+1)
	}
}

// gap is the gap before a shaft: the culture's draw for it, or the least
// gap if the tower has to fit a room.
func (tower *Tower) gap(index int) float64 {
	if tower.room > 0 {
		return tower.drawing.fixed("fit-gap")
	}
	return tower.drawing.number("shaft-gap", place{Bay: index})
}

// radius is how far the farthest corner of any plan of the shaft, at the
// bottom or the top of any segment, is from its centre: the room it takes.
func (shaft shaft) radius() float64 {
	var widest float64
	for _, segment := range shaft.Segments {
		widest = math.Max(widest, math.Max(segment.Bottom.radius(),
			segment.Top.radius()))
	}
	return widest
}

// radius is how far the farthest corner of the plan is from its centre.
func (shape footprintShape) radius() float64 {
	parts, err := shape.parts()
	if err != nil {
		return 0
	}
	var farthest float64
	for _, part := range parts {
		for _, corner := range part {
			farthest = math.Max(farthest, math.Hypot(corner.X, corner.Z))
		}
	}
	return farthest
}

// layer is the parts of the plan at the bottom and the top of a segment
// of a shaft, turned by all the turns up to and including the segment's,
// standing on the shaft's centre.
func (shaft shaft) layer(index int) (bottom, top [][]outlineCorner, err error) {
	var turn float64
	for _, segment := range shaft.Segments[:index+1] {
		turn += segment.Turn
	}
	segment := shaft.Segments[index]
	for _, ends := range []struct {
		shape footprintShape
		into  *[][]outlineCorner
	}{{segment.Bottom, &bottom}, {segment.Top, &top}} {
		parts, err := ends.shape.parts()
		if err != nil {
			return nil, nil, err
		}
		for _, part := range parts {
			*ends.into = append(*ends.into,
				moved(turned(part, turn), shaft.Centre))
		}
	}
	return bottom, top, nil
}

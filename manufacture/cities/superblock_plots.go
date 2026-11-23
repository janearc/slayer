package cities

// width is the rectangle's extent along x.
func (box rect) width() float64 { return box.X1 - box.X0 }

// depth is the rectangle's extent along z.
func (box rect) depth() float64 { return box.Z1 - box.Z0 }

// area is the ground the rectangle covers, in square metres.
func (box rect) area() float64 { return box.width() * box.depth() }

// joins is whether two rectangles share a stretch of edge longer than
// a least: corners that meet, and edges that share only a crack, do not
// join.
func (box rect) joins(other rect, least float64) bool {
	along := min(box.X1, other.X1) - max(box.X0, other.X0)
	across := min(box.Z1, other.Z1) - max(box.Z0, other.Z0)
	return (along >= -meeting && across > least) ||
		(across >= -meeting && along > least)
}

// reachesEdge is whether the rectangle stands on the edge of the parcel.
func (box rect) reachesEdge(parcel rect) bool {
	return box.X0-parcel.X0 < meeting || parcel.X1-box.X1 < meeting ||
		box.Z0-parcel.Z0 < meeting || parcel.Z1-box.Z1 < meeting
}

// cutting is a parcel being cut into plots and alleys: the rectangles
// of each so far, how much alley is still allowed, and which cut it is.
type cutting struct {
	drawing      *drawer
	least, most  float64
	stop, reach  float64
	gapMost      float64
	budget       float64
	next         int
	lots, alleys []rect
}

// cut leaves a lane round the court under the slabs' decks, cuts the rest
// of the court into plots and alleys, and finds which alleys do not reach
// the court's edge: the light wells. The plots are not yet built.
func (growing *accretion) cut() {
	drawing := growing.drawing
	court := growing.out.Court
	lane := drawing.fixed("lane-width")
	inner := court.inset(lane)
	cuts := &cutting{drawing: drawing,
		least: drawing.fixed("plot-least"), most: drawing.fixed("plot-most"),
		stop:    drawing.fixed("plot-stop-reach"),
		reach:   drawing.fixed("alley-reach"),
		budget:  drawing.fixed("alley-share-most") * inner.area(),
		gapMost: drawing.most("alley-width")}
	cuts.split(inner, 0)
	growing.out.Alleys = append(laneRound(court, inner), cuts.alleys...)
	growing.out.Sealed = sealed(growing.out.Alleys, court,
		drawing.fixed("alley-opening"))
	growing.lots = cuts.lots
}

// laneRound is the lane round the inside of a court, as four alleys: the
// back and front as long as the court is wide, and the sides between them.
func laneRound(court, inner rect) []rect {
	return []rect{{court.X0, court.Z0, court.X1, inner.Z0},
		{court.X0, inner.Z1, court.X1, court.Z1},
		{court.X0, inner.Z0, inner.X0, inner.Z1},
		{inner.X1, inner.Z0, court.X1, inner.Z1}}
}

// split cuts one rectangle: it is left as a plot, or cut across its
// longer side into two, with an alley between them if the draw says so.
func (cut *cutting) split(box rect, depth int) {
	at := place{Row: depth, Bay: cut.next}
	cut.next++
	along := box.width() >= box.depth()
	length, cross := box.width(), box.depth()
	if !along {
		length, cross = cross, length
	}
	if !cut.divides(length, at) {
		cut.lots = append(cut.lots, box)
		return
	}
	gap := 0.0
	if cut.gapped(length, cross, at) {
		gap = cut.drawing.number("alley-width", at)
	}
	first := cut.firstLength(length-gap, at)
	one, strip, two := box.cutAt(along, first, gap)
	if gap > 0 {
		cut.alleys = append(cut.alleys, strip)
		cut.budget -= strip.area()
	}
	cut.split(one, depth+1)
	cut.split(two, depth+1)
}

// divides is whether a rectangle with a longer side of a length is cut:
// always if it is longer than the most, never if it cannot give two
// plots of the least and room for an alley, and by the draw if it is
// short enough to be left.
func (cut *cutting) divides(length float64, at place) bool {
	switch {
	case length > cut.most:
		return true
	case length < 2*cut.least+cut.gapMost:
		return false
	case length <= cut.stop:
		return !cut.drawing.chance("plot-stop", at)
	}
	return true
}

// gapped is whether a cut leaves an alley: not across a side longer than
// the alleys reach, not when the alleys have used the share they may, and
// otherwise by the draw.
func (cut *cutting) gapped(length, cross float64, at place) bool {
	room := cross * cut.gapMost
	return cross <= cut.reach && cut.budget >= room &&
		length >= 2*cut.least+cut.gapMost &&
		cut.drawing.chance("alley-chance", at)
}

// firstLength is how long the first plot of a cut is along the side
// cut, out of the length there is, from where the cut falls but never so
// short that either plot is under the least.
func (cut *cutting) firstLength(length float64, at place) float64 {
	first := length * cut.drawing.number("split-at", at)
	return min(max(first, cut.least), length-cut.least)
}

// cutAt is the rectangle cut across its longer side, along x or z: the
// plot that is first by a length, the gap of a width, and the rest.
func (box rect) cutAt(along bool, first, gap float64) (one, strip,
	two rect) {
	if along {
		one = rect{box.X0, box.Z0, box.X0 + first, box.Z1}
		strip = rect{one.X1, box.Z0, one.X1 + gap, box.Z1}
		return one, strip, rect{strip.X1, box.Z0, box.X1, box.Z1}
	}
	one = rect{box.X0, box.Z0, box.X1, box.Z0 + first}
	strip = rect{box.X0, one.Z1, box.X1, one.Z1 + gap}
	return one, strip, rect{box.X0, strip.Z1, box.X1, box.Z1}
}

// sealed are the alleys that do not reach the edge of the parcel
// through other alleys they join by an opening of at least a given
// width, found by spreading from those that stand on it.
func sealed(alleys []rect, parcel rect, opening float64) []rect {
	reached := make([]bool, len(alleys))
	var queue []int
	for index, alley := range alleys {
		if alley.reachesEdge(parcel) {
			reached[index] = true
			queue = append(queue, index)
		}
	}
	for ; len(queue) > 0; queue = queue[1:] {
		for index, alley := range alleys {
			if !reached[index] && alley.joins(alleys[queue[0]], opening) {
				reached[index] = true
				queue = append(queue, index)
			}
		}
	}
	var closed []rect
	for index, alley := range alleys {
		if !reached[index] {
			closed = append(closed, alley)
		}
	}
	return closed
}

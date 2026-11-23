package cities

// topAndBase is what a building carries on its roof and at its foot: the
// plant room, tanks, antennas and gantry on the roof, a billboard there
// and another on a side wall, and the row of shops along the front of its
// ground storey. In the building's frame. The roof is the top one, over
// the highest of its storeys, and the low is how high the lowest tier's
// wall reaches.
func (shared court) topAndBase(drawing *drawer, at place, shape massShape,
	built stack) []placedElement {
	parts := drawing.crown(at, built.top)
	parts = append(parts, drawing.roofBoard(at, built.top)...)
	parts = append(parts, drawing.wallBoard(at, shape, built.low,
		drawing.chance("board-left", at))...)
	return append(parts, drawing.podium(at, shape.depth, shape.from,
		shape.width)...)
}

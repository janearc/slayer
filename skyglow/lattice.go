package skyglow

import "github.com/janearc/slayer/maths"

// Cells is the box the field's grid fills, in the scene's space, and how
// many cells it has east, up and south. Its values are at the middles of
// the cells, and Mean reads between them. A drawer that keeps the field
// in its own memory, as the GPU does, takes it by these and Cell.
func (field *Field) Cells() (low, high maths.Vec, count [3]int) {
	made := field.grid
	span := maths.V(made.step[0]*float64(made.count[0]),
		made.step[1]*float64(made.count[1]),
		made.step[2]*float64(made.count[2]))
	return made.low, made.low.Add(span), made.count
}

// Cell is the mean radiance at the middle of one cell, by its place
// east, up and south, each held to the grid as Mean holds a place
// beyond it.
func (field *Field) Cell(east, up, south int) maths.Light {
	return field.values[field.at(max(east, 0), max(up, 0), max(south, 0))]
}

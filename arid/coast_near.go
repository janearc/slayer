// This file is the coast's steepness by place: what a ray's search over a
// coast steps by, so that one cliff does not shorten every step taken
// anywhere over the sea.

package arid

import "math"

// nearCells is how many of the ground's cells a side one tile of the
// coast's steepness covers. Picked: eight, so a tile is wider than any
// step a search takes near the surface and there are few of them.
const nearCells = 8

// tile fills near: for each tile of nearCells cells a side, the ground's
// steepest rise where the share of the waves changes (cellSteepest), in
// the tile and in the eight tiles around it.
func (shore *coast) tile() {
	shore.nearAcross = (shore.across + nearCells - 2) / nearCells
	shore.nearDown = (shore.down + nearCells - 2) / nearCells
	own := make([]float64, shore.nearAcross*shore.nearDown)
	for row := range shore.down - 1 {
		for column := range shore.across - 1 {
			at := row/nearCells*shore.nearAcross + column/nearCells
			own[at] = math.Max(own[at], shore.cellSteepest(column, row))
		}
	}
	shore.near = make([]float64, len(own))
	for row := range shore.nearDown {
		for column := range shore.nearAcross {
			shore.near[row*shore.nearAcross+column] = shore.around(own,
				column, row)
		}
	}
}

// around is the greatest of own in the tile at a column and a row and in
// the tiles beside it, those the grid has.
func (shore *coast) around(own []float64, column, row int) float64 {
	var most float64
	for down := max(row-1, 0); down <= min(row+1, shore.nearDown-1); down++ {
		for across := max(column-1, 0); across <= min(column+1,
			shore.nearAcross-1); across++ {
			most = math.Max(most, own[down*shore.nearAcross+across])
		}
	}
	return most
}

// tileSide is the side of one tile, in metres.
func (shore *coast) tileSide() float64 { return nearCells * shore.step }

// steepestNear is the ground's steepest rise, where the share of the
// waves changes, anywhere within a tile's side of a place east and south
// in the sea's frame. A place off the grid is held to its edge, as locate
// holds it: beyond the grid the ground is the nearest edge sample's. It
// is never more than steepest, the whole coast's, and over open water far
// from any shallows it is nought.
func (shore *coast) steepestNear(east, south float64) float64 {
	column := int(math.Floor((east - shore.east) / shore.step))
	row := int(math.Floor((south - shore.south) / shore.step))
	column = min(max(column, 0), shore.across-2) / nearCells
	row = min(max(row, 0), shore.down-2) / nearCells
	return shore.near[row*shore.nearAcross+column]
}

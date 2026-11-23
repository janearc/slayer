package cities

import (
	"fmt"
	"math"
)

// A grid is a town's habit: blocks bounded by streets, and on every long
// side of a block a row of houses fronting the street. The streets are
// laid on a lattice pinned to the map's own origin and not to any stand,
// so two stands over neighbouring boxes agree on where every street runs,
// and a block belongs to the box that holds its middle.

// block is the room a row of houses takes in a grid, and how the grid is
// laid. All are in the item's data.
type block struct {
	// Length is how long a block is, along its long streets, in metres.
	Length float64
	// Depth is how deep a block is, from one long street to the next
	// row of blocks, in metres, the streets not included.
	Depth float64
	// Street is how wide every street is, in metres.
	Street float64
	// RowLength and RowDepth are the room the item itself takes, the
	// length of its front and how far back it goes.
	RowLength, RowDepth float64
	// Turn is how far the grid is turned clockwise off true north, in
	// degrees. The long streets run at a right angle to the way the
	// north fronts look, which is the turn.
	Turn float64
}

// Quarter is a rectangle of the map, in metres east and north, half open: it
// holds what stands at its west and south edges and not at its east and
// north.
type Quarter struct {
	West, South, East, North float64
}

// holds is whether a place is in the quarter.
func (box Quarter) holds(east, north float64) bool {
	return east >= box.West && east < box.East &&
		north >= box.South && north < box.North
}

// grid lays a row of houses on each long side of every block whose middle
// is in the ground's box, each facing its street. A block's two rows stand
// back to back, the one with its front to the north and the other to the
// south. It lays as many as there are, and if the stand asks for fewer
// takes that many by seed.
type grid struct{}

// Sites is the rows of houses of the blocks in the ground's box, a site
// at the back left corner of each, which is where a drawn row stands from,
// or why the item cannot be laid in a grid.
func (grid) Sites(ground Ground, item Item, seed Seed, most int) (Laid, error) {
	if !item.Blocked {
		return Laid{}, fmt.Errorf("cities: a grid lays out blocks, and the "+
			"data gives %s none", item.Kind)
	}
	block := item.Block
	if !(block.Length > 0 && block.Depth > 0 && block.Street >= 0 &&
		block.RowLength > 0 && block.RowDepth > 0 &&
		block.RowDepth*2 <= block.Depth) {
		return Laid{}, fmt.Errorf("cities: a block needs a length and depth, "+
			"and two rows that fit in its depth, not %+v", block)
	}
	if ground.Box == nil {
		return Laid{}, fmt.Errorf("cities: a grid fills a box of the map, " +
			"and the stand gives none")
	}
	sites := block.rows(*ground.Box)
	chosen := taken(sites, most, seed)
	return Laid{Sites: chosen, Said: fmt.Sprintf("a grid of blocks %g by %g "+
		"m in streets %g m wide laid %d rows, turned %g degrees",
		block.Length, block.Depth, block.Street, len(chosen), block.Turn)}, nil
}

// rows is every row of houses of the blocks whose middles are in the box,
// block by block, the north row and then the south.
func (block block) rows(box Quarter) []Site {
	alongEast, alongNorth := toward(block.Turn + quarterDegrees)
	acrossEast, acrossNorth := toward(block.Turn)
	stepAlong := block.Length + block.Street
	stepAcross := block.Depth + block.Street
	lowest, highest := math.Inf(1), math.Inf(-1)
	least, most := math.Inf(1), math.Inf(-1)
	for _, east := range []float64{box.West, box.East} {
		for _, north := range []float64{box.South, box.North} {
			along := (east*alongEast + north*alongNorth) / stepAlong
			across := (east*acrossEast + north*acrossNorth) / stepAcross
			lowest, highest = math.Min(lowest, along), math.Max(highest, along)
			least, most = math.Min(least, across), math.Max(most, across)
		}
	}
	var sites []Site
	for row := int(math.Floor(least)); row <= int(math.Ceil(most)); row++ {
		for slot := int(math.Floor(lowest)); slot <= int(math.Ceil(highest)); slot++ {
			east := float64(slot)*stepAlong*alongEast +
				float64(row)*stepAcross*acrossEast
			north := float64(slot)*stepAlong*alongNorth +
				float64(row)*stepAcross*acrossNorth
			if box.holds(east, north) {
				sites = append(sites, block.pair(east, north)...)
			}
		}
	}
	return sites
}

// pair is the two rows of one block, from the middle of the block.
func (block block) pair(east, north float64) []Site {
	back := block.Depth/2 - block.RowDepth/2
	return []Site{
		block.row(east, north, back, block.Turn),
		block.row(east, north, -back, block.Turn+halfTurn),
	}
}

// row is the site of one row of houses, whose middle stands off the
// middle of its block by an offset along the turn, north for a positive
// one. The site is the row's back left corner: half its length to the
// left of the middle, as it faces, and half its depth behind it.
func (block block) row(east, north, offset, front float64) Site {
	aheadEast, aheadNorth := toward(block.Turn)
	middleEast, middleNorth := east+offset*aheadEast, north+offset*aheadNorth
	frontEast, frontNorth := toward(front)
	rightEast, rightNorth := toward(front - quarterDegrees)
	return Site{Faced: true, Heading: math.Mod(front+fullDegrees, fullDegrees),
		East: middleEast - block.RowLength/2*rightEast -
			block.RowDepth/2*frontEast,
		North: middleNorth - block.RowLength/2*rightNorth -
			block.RowDepth/2*frontNorth}
}

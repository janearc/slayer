// This file is the ground a sea with no edge meets: which places the open
// sea reaches, how deep the water is over each, and how much of the open
// sea's waves that depth can carry.

package arid

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/slayer/oceantex"
)

// Ground is heights on a square grid, in metres: so many points across,
// eastward, and down, southward, so many metres apart. terrain.Window is
// one.
type Ground interface {
	At(across, down int) float64
	Size() (across, down int, step float64)
}

// joined are the steps, in columns and rows, from a sample to the samples
// the ground's triangles join it to by an edge: the four beside it, and
// the two along the diagonal from a cell's first corner to its last, the
// one edge terrain.Window.GroundOn draws across each cell.
var joined = [...][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {-1, -1}}

// coast is the ground under and beside a sea, in the sea's own frame:
// its first sample east and south of the sea's middle by the metres
// given, its heights above the sea's mean level. Between samples the
// ground is the triangles terrain.Window.GroundOn draws, two a cell.
type coast struct {
	east, south, step float64
	across, down      int
	// height is the ground above the sea's mean level, in metres.
	height []float64
	// cut is whether a sample is below the sea's level and not joined to
	// the open sea through samples below it.
	cut []bool
	// full is the depth, in metres, from which the open sea's waves
	// stand whole: twice their greatest reach over the breaking ratio.
	full float64
	// near is, tile by tile, the ground's steepest rise where the share
	// of the waves changes, in the tile and the eight around it; the
	// tiles are nearAcross a row and nearDown rows (tile).
	near                 []float64
	nearAcross, nearDown int
}

// spot is where a place falls on the coast: the triangle under it, the
// share of each corner in it, the ground's rise a metre east and south
// in the triangle, and whether the place is on the grid at all.
type spot struct {
	corners    [3]int
	weights    [3]float64
	rise, fall float64
	on         bool
}

// sides is how many of a wave's reach make its height: the reach above
// the mean to the crest, and the reach below it to the trough.
const sides = 2

// newCoast reads a ground for a sea whose waves reach at most reach above
// and below its mean: where the ground's first sample is in the sea's
// frame, east and south, and the sea's mean level in the ground's own
// heights. It refuses a ground of fewer than two samples a side, a step
// that is not a length, and anything not finite.
func newCoast(ground Ground, east, south, level, reach float64) (*coast,
	error) {
	if ground == nil {
		return nil, errors.New("arid: a coast needs ground")
	}
	across, down, step := ground.Size()
	if across < 2 || down < 2 || !finite(step) || !(step > 0) ||
		!finite(east) || !finite(south) || !finite(level) || !finite(reach) {
		return nil, errors.New("arid: a coast needs two samples a side, " +
			"a step above nought and a finite place and level")
	}
	made := &coast{east: east, south: south, step: step, across: across,
		down: down, height: make([]float64, across*down),
		full: sides * reach / oceantex.BreakingRatio}
	for row := range down {
		for column := range across {
			height := ground.At(column, row) - level
			if !finite(height) {
				return nil, fmt.Errorf("arid: the ground at %d, %d is %g",
					column, row, height)
			}
			made.height[row*across+column] = height
		}
	}
	made.flood()
	made.tile()
	return made, nil
}

// flood marks the samples the open sea does not reach. The open sea is
// taken to lie beyond every edge of the grid, so it comes in at each edge
// sample below its level, and from there over every sample below its
// level that an edge of the ground's triangles joins to one it has
// reached. A sample below the level it never comes to is cut off.
func (shore *coast) flood() {
	reached := make([]bool, len(shore.height))
	queue := shore.edges(reached)
	for len(queue) > 0 {
		at := queue[len(queue)-1]
		queue = shore.spread(at, reached, queue[:len(queue)-1])
	}
	shore.cut = make([]bool, len(shore.height))
	for at, height := range shore.height {
		shore.cut[at] = height < 0 && !reached[at]
	}
}

// edges are the samples on the grid's edge that the open sea comes in at,
// marked as reached.
func (shore *coast) edges(reached []bool) []int {
	var queue []int
	last := (shore.down - 1) * shore.across
	for column := range shore.across {
		queue = shore.reach(column, reached, queue)
		queue = shore.reach(last+column, reached, queue)
	}
	for row := range shore.down {
		first := row * shore.across
		queue = shore.reach(first, reached, queue)
		queue = shore.reach(first+shore.across-1, reached, queue)
	}
	return queue
}

// spread reaches the samples joined to one already reached.
func (shore *coast) spread(at int, reached []bool, queue []int) []int {
	column, row := at%shore.across, at/shore.across
	for _, step := range joined {
		next, down := column+step[0], row+step[1]
		if next < 0 || down < 0 || next >= shore.across || down >= shore.down {
			continue
		}
		queue = shore.reach(down*shore.across+next, reached, queue)
	}
	return queue
}

// reach marks a sample reached and queues it, if it is below the sea's
// level and not reached already.
func (shore *coast) reach(at int, reached []bool, queue []int) []int {
	if shore.height[at] >= 0 || reached[at] {
		return queue
	}
	reached[at] = true
	return append(queue, at)
}

// locate is where a place, east and south in the sea's frame, falls on
// the coast. A place off the grid is held to its edge, so beyond the grid
// the ground is the nearest edge sample's, and does not rise or fall
// along an axis it was held on.
func (shore *coast) locate(east, south float64) spot {
	across := (east - shore.east) / shore.step
	down := (south - shore.south) / shore.step
	lastColumn, lastRow := float64(shore.across-1), float64(shore.down-1)
	held := spot{on: across >= 0 && down >= 0 && across <= lastColumn &&
		down <= lastRow}
	heldAcross := math.Min(math.Max(across, 0), lastColumn)
	heldDown := math.Min(math.Max(down, 0), lastRow)
	column := min(int(heldAcross), int(lastColumn)-1)
	row := min(int(heldDown), int(lastRow)-1)
	into, deep := heldAcross-float64(column), heldDown-float64(row)
	shore.triangle(&held, column, row, into >= deep)
	if into >= deep {
		held.weights = [3]float64{1 - into, into - deep, deep}
	} else {
		held.weights = [3]float64{1 - deep, deep - into, into}
	}
	if heldAcross != across {
		held.rise = 0
	}
	if heldDown != down {
		held.fall = 0
	}
	return held
}

// triangle fills a spot's corners and the ground's rise from one of the
// two triangles of the cell at a column and a row, which meet on the
// diagonal from its first corner to its last: the one above that
// diagonal, toward the cell's next column, or the one below it, toward
// its next row. The corners run first corner, the other, last corner.
func (shore *coast) triangle(place *spot, column, row int, upper bool) {
	near := row*shore.across + column
	far := near + shore.across
	other := far
	if upper {
		other = near + 1
	}
	place.corners = [3]int{near, other, far + 1}
	first, middle, last := shore.height[near], shore.height[other],
		shore.height[far+1]
	place.rise, place.fall = last-middle, middle-first
	if upper {
		place.rise, place.fall = middle-first, last-middle
	}
	place.rise /= shore.step
	place.fall /= shore.step
}

// ground is the ground's height above the sea's mean level at a spot.
func (shore *coast) ground(place spot) float64 {
	var height float64
	for index, corner := range place.corners {
		height += place.weights[index] * shore.height[corner]
	}
	return height
}

// dry is whether a spot is in a triangle with a corner the open sea does
// not reach. Such a triangle's corners below the sea's level are all cut
// off, for an edge joins each pair, so no water stands over any of it.
func (shore *coast) dry(place spot) bool {
	if !place.on {
		return false
	}
	for _, corner := range place.corners {
		if shore.cut[corner] {
			return true
		}
	}
	return false
}

// wet is whether water may stand at a place, east and south in the sea's
// frame: everywhere for a sea with no coast, and off the grid; on it,
// wherever the triangle under the place is not dry. Over land in a triangle
// that is not dry the surface is the mean plane, under the ground and
// hidden by it from above, but a ray that reaches it some other way meets
// it.
func (shore *coast) wet(east, south float64) bool {
	if shore == nil || !finite(east) || !finite(south) {
		return true
	}
	return !shore.dry(shore.locate(east, south))
}

// meet is how much of the open sea's waves stand at a place, east and
// south in the sea's frame, from none to all, and how fast that share
// changes a metre east and a metre south. A sea with no coast is open
// everywhere, as is a place that is not a number.
//
// The share is the depth over full, held between none and all. The open
// surface is never further than its reach from its mean, so the share
// times it is never further than half the breaking ratio of the depth
// [Mc94]; over ground at or above the sea's level the share is none.
func (shore *coast) meet(east, south float64) (share, eastward,
	southward float64) {
	if shore == nil || !finite(east) || !finite(south) {
		return 1, 0, 0
	}
	place := shore.locate(east, south)
	depth := -shore.ground(place)
	switch {
	case depth <= 0:
		return 0, 0, 0
	case depth >= shore.full:
		return 1, 0, 0
	}
	return depth / shore.full, -place.rise / shore.full,
		-place.fall / shore.full
}

// steepest is the most the ground rises a metre, in any direction, in a
// triangle where the share of the waves changes: one whose corners are
// not all at or above the sea's level, nor all at least full deep.
func (shore *coast) steepest() float64 {
	var most float64
	for row := range shore.down - 1 {
		for column := range shore.across - 1 {
			most = math.Max(most, shore.cellSteepest(column, row))
		}
	}
	return most
}

// cellSteepest is steepest over the two triangles of one cell.
func (shore *coast) cellSteepest(column, row int) float64 {
	var most float64
	for _, upper := range [...]bool{true, false} {
		var place spot
		shore.triangle(&place, column, row, upper)
		if shore.changes(place) {
			most = math.Max(most, math.Hypot(place.rise, place.fall))
		}
	}
	return most
}

// changes is whether the share of the waves changes over a triangle.
func (shore *coast) changes(place spot) bool {
	shallowest, deepest := math.Inf(1), math.Inf(-1)
	for _, corner := range place.corners {
		depth := -shore.height[corner]
		shallowest, deepest = math.Min(shallowest, depth),
			math.Max(deepest, depth)
	}
	return deepest > 0 && shallowest < shore.full
}

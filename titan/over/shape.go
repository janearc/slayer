package over

import (
	"cmp"
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
)

// neighbours are the eight places beside a sample, as steps across and
// down.
var neighbours = [8][2]int{
	{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1},
}

// shape is what a map shows of itself at each of its samples: how high,
// which way it faces, which way is downhill, how much ground drains
// through it and how broken the map is around it. The made ground reads all
// of it and the map itself is never changed.
type shape struct {
	across, down int
	step         float64
	height       []float64   // metres over the floor
	normal       []maths.Vec // unit, smoothed across the neighbours
	fall         []maths.Vec // unit, horizontal, downhill; x and z only
	drained      []float64   // square metres draining through the sample
	rough        []float64   // metres of spread over the sample's neighbourhood
	fit          fitting     // for reading rough
}

// Drainage is how much ground drains through each sample of a map, in
// square metres, row after row. Each sample passes all it has gathered, its
// own cell and what came to it, to its steepest lower neighbour, among the
// eight beside it. A sample with no lower neighbour keeps it.
func Drainage(from Map) []float64 {
	across, down, _ := from.Size()
	return gathered(heightsOf(from), across, down, stepOf(from))
}

// stepOf is how far apart a map's samples are.
func stepOf(from Map) float64 {
	_, _, step := from.Size()
	return step
}

// heightsOf is a map's heights, row after row.
func heightsOf(from Map) []float64 {
	across, down, _ := from.Size()
	heights := make([]float64, 0, across*down)
	for row := range down {
		for column := range across {
			heights = append(heights, from.At(column, row))
		}
	}
	return heights
}

// gathered is the area each sample of a grid of heights drains, working
// from the highest sample to the lowest so that every sample has all it will
// get before it passes it on.
func gathered(heights []float64, across, down int, step float64) []float64 {
	order := make([]int, len(heights))
	for i := range order {
		order[i] = i
	}
	slices.SortFunc(order, func(first, second int) int {
		return cmp.Or(cmp.Compare(heights[second], heights[first]),
			cmp.Compare(first, second))
	})
	cells := make([]float64, len(heights))
	for i := range cells {
		cells[i] = 1
	}
	for _, sample := range order {
		if lower, found := steepest(heights, across, down, sample); found {
			cells[lower] += cells[sample]
		}
	}
	for i := range cells {
		cells[i] *= step * step
	}
	return cells
}

// steepest is the lower neighbour a sample sends its water to: the one it
// falls to fastest, a diagonal counted for the further it is. It says
// whether there is one.
func steepest(heights []float64, across, down, sample int) (int, bool) {
	column, row := sample%across, sample/across
	best, found, fastest := 0, false, 0.0
	for _, move := range neighbours {
		toward, below := column+move[0], row+move[1]
		if toward < 0 || toward >= across || below < 0 || below >= down {
			continue
		}
		other := below*across + toward
		drop := (heights[sample] - heights[other]) /
			math.Hypot(float64(move[0]), float64(move[1]))
		if drop > fastest {
			best, found, fastest = other, true, drop
		}
	}
	return best, found
}

// survey reads a map into its shape, every height taken from the floor.
func survey(from Map, floor float64) *shape {
	across, down, step := from.Size()
	read := &shape{across: across, down: down, step: step,
		height: heightsOf(from)}
	for i := range read.height {
		read.height[i] -= floor
	}
	read.drained = gathered(read.height, across, down, step)
	read.normal = make([]maths.Vec, len(read.height))
	read.fall = make([]maths.Vec, len(read.height))
	read.rough = make([]float64, len(read.height))
	read.fit = quadricFit()
	for row := range down {
		for column := range across {
			read.reads(column, row)
		}
	}
	return read
}

// reads fills in what is read at one sample: its normal, its fall line
// and its roughness.
func (read *shape) reads(column, row int) {
	i := row*read.across + column
	east, south := read.rise(column, row)
	read.normal[i] = maths.V(-east, 1, -south).Unit()
	read.fall[i] = maths.V(1, 0, 0)
	if run := math.Hypot(east, south); run > 0 {
		read.fall[i] = maths.V(-east/run, 0, -south/run)
	}
	read.rough[i] = read.spread(column, row)
}

// rise is how fast the ground rises east and south at a sample, metres a
// metre, read across the samples either side, or the one beside it at the
// map's edge.
func (read *shape) rise(column, row int) (east, south float64) {
	west, eastward := max(0, column-1), min(read.across-1, column+1)
	north, southward := max(0, row-1), min(read.down-1, row+1)
	east = (read.at(eastward, row) - read.at(west, row)) /
		(float64(eastward-west) * read.step)
	south = (read.at(column, southward) - read.at(column, north)) /
		(float64(southward-north) * read.step)
	return east, south
}

// at is the height of a sample, metres over the floor.
func (read *shape) at(column, row int) float64 {
	return read.height[row*read.across+column]
}

// place is where a point stands among the samples: the three corners of
// the triangle it is in, and how much of each it is. The triangles are the
// map's own, two to a cell, cut from the corner with the lowest column and
// row to the corner with the highest.
type place struct {
	corners [3]int
	weights [3]float64
	across  float64 // how far across its cell, nought to one
	down    float64 // how far down its cell, nought to one
}

// locate is the place of a point, east and south in metres from the map's
// first sample.
func (read *shape) locate(east, south float64) place {
	column := min(max(int(math.Floor(east/read.step)), 0), read.across-2)
	row := min(max(int(math.Floor(south/read.step)), 0), read.down-2)
	across := east/read.step - float64(column)
	down := south/read.step - float64(row)
	first := row*read.across + column
	second, third := first+read.across, first+read.across+1
	if across >= down {
		return place{[3]int{first, first + 1, third},
			[3]float64{1 - across, across - down, down}, across, down}
	}
	return place{[3]int{first, second, third},
		[3]float64{1 - down, down - across, across}, across, down}
}

// surface is the shape's surface at a place, metres over the floor.
func (read *shape) surface(at place) float64 {
	var sum float64
	for i, corner := range at.corners {
		sum += at.weights[i] * read.height[corner]
	}
	return sum
}

// smoothed is the shape's normal at a place: the corners' normals blended
// and made one long.
func (read *shape) smoothed(at place) maths.Vec {
	var sum maths.Vec
	for i, corner := range at.corners {
		sum = sum.Add(read.normal[corner].Scale(at.weights[i]))
	}
	return sum.Unit()
}

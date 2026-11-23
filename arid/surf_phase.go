// This file bends a band of waves over a coast: where the water is
// shallower its crests come closer and turn toward the shallows, and it
// works out by how much at every sample of the ground.

package arid

import (
	"math"

	"github.com/janearc/slayer/oceantex"
)

// sweep is the bending of one band of waves over a coast, solved on the
// ground's grid. Its crests are the level lines of time, which grows a
// metre for every metre the waves travel in open water, and faster where
// the water is shallow: |grad time| = k(h) / k(open), the eikonal.
//
// It is solved by fast sweeping (Zhao 2005, Math. Comp. 74, 603-627):
// Godunov's upwind update at every sample, the grid swept in each of its
// four orders, round after round, until a round changes nothing.
type sweep struct {
	across, down int
	step         float64
	// cost is how much time grows over one grid step at each sample, or
	// not a number where there is no water for the waves to cross.
	cost []float64
	// time is the solution so far, infinite where nothing has reached.
	time []float64
	// fixed is whether a sample's time is given, where the waves come in.
	fixed []bool
}

// sweepOrders are the four ways the grid is swept: east or west along each
// row, and the rows north to south or south to north.
var sweepOrders = [...][2]bool{{true, true}, {false, true}, {true, false},
	{false, false}}

// bandNumbers is, for each band in the order oceantex makes them, the
// longest first, the wave number in radians a metre that stands for the
// band in its bending: the spectrum's peak, peak, held to the wave
// numbers the band carries. A band carries those from the last band's
// shortest to the shortest its own grid holds, pi N / side.
func bandNumbers(bands []*oceantex.Field, peak float64) []float64 {
	numbers := make([]float64, len(bands))
	var longest float64
	for index, band := range bands {
		shortest := math.Pi * float64(band.N) / band.Extent
		numbers[index] = math.Min(math.Max(peak, longest), shortest)
		longest = shortest
	}
	return numbers
}

// lag is how far, in metres along the waves' way, a band of waves over the
// coast is ahead of the open sea's at each sample: time less the distance
// along toward, the waves' way as an east and a south of one length.
// number is the band's wave number, open the open water's depth, and
// floor the depth under which the bending is held. It also says how many
// rounds the sweeps took.
func (shore *coast) lag(number, open, floor float64, toward [2]float64) (
	[]float64, int) {
	field := shore.sweepFor(number, open, floor)
	shore.seed(field, toward)
	rounds := field.solve()
	lags := make([]float64, len(field.time))
	known := make([]bool, len(field.time))
	for at, time := range field.time {
		if !math.IsInf(time, 1) {
			east, south := shore.place(at)
			lags[at] = time - toward[0]*east - toward[1]*south
			known[at] = true
		}
	}
	shore.fill(lags, known)
	return lags, rounds
}

// sweepFor is a sweep over the coast for a band of a wave number: the cost
// of a step at a sample is the step times the wave number in its depth
// over the open water's, the depth held to at least floor. Land and water
// the open sea does not reach have no cost.
func (shore *coast) sweepFor(number, open, floor float64) *sweep {
	field := &sweep{across: shore.across, down: shore.down, step: shore.step,
		cost:  make([]float64, len(shore.height)),
		time:  make([]float64, len(shore.height)),
		fixed: make([]bool, len(shore.height))}
	omega := frequency(number, open)
	reference := waveNumber(omega, open)
	for at, height := range shore.height {
		field.time[at] = math.Inf(1)
		field.cost[at] = math.NaN()
		if height < 0 && !shore.cut[at] {
			depth := math.Max(-height, floor)
			field.cost[at] = shore.step * waveNumber(omega, depth) / reference
		}
	}
	return field
}

// place is where a sample is in the sea's frame, east and south.
func (shore *coast) place(at int) (east, south float64) {
	return shore.east + float64(at%shore.across)*shore.step,
		shore.south + float64(at/shore.across)*shore.step
}

// seed gives the time at the water samples of each edge the waves come in
// over, the distance along their way, as the open sea's waves have it.
func (shore *coast) seed(field *sweep, toward [2]float64) {
	last := (shore.down - 1) * shore.across
	for column := range shore.across {
		shore.give(field, column, toward, toward[1] > 0)
		shore.give(field, last+column, toward, toward[1] < 0)
	}
	for row := range shore.down {
		first := row * shore.across
		shore.give(field, first, toward, toward[0] > 0)
		shore.give(field, first+shore.across-1, toward, toward[0] < 0)
	}
}

// give fixes one edge sample's time, if the waves come in over its edge
// and it is water.
func (shore *coast) give(field *sweep, at int, toward [2]float64,
	coming bool) {
	if !coming || math.IsNaN(field.cost[at]) {
		return
	}
	east, south := shore.place(at)
	field.time[at] = toward[0]*east + toward[1]*south
	field.fixed[at] = true
}

// solve sweeps the field round after round until a round changes no time
// by more than sweepSettled of a step, or sweepRounds have been swept, and
// says how many rounds it took.
func (field *sweep) solve() int {
	for round := range sweepRounds {
		if field.round() <= sweepSettled*field.step {
			return round + 1
		}
	}
	return sweepRounds
}

// round sweeps the grid in each of its four orders, and is the most any
// time changed.
func (field *sweep) round() float64 {
	var most float64
	for _, order := range sweepOrders {
		for row := range field.down {
			if !order[1] {
				row = field.down - 1 - row
			}
			for column := range field.across {
				if !order[0] {
					column = field.across - 1 - column
				}
				most = math.Max(most, field.relax(column, row))
			}
		}
	}
	return most
}

// relax is Godunov's upwind update at a sample: the least time that its
// least neighbours along each axis, a and b, and its cost allow. A time
// infinite on one axis leaves the other's alone. It is how much the time
// fell, and nothing for a fixed sample or one with no water.
func (field *sweep) relax(column, row int) float64 {
	at := row*field.across + column
	cost := field.cost[at]
	if field.fixed[at] || math.IsNaN(cost) {
		return 0
	}
	a := math.Min(field.at(column-1, row), field.at(column+1, row))
	b := math.Min(field.at(column, row-1), field.at(column, row+1))
	if math.IsInf(a, 1) && math.IsInf(b, 1) {
		return 0
	}
	next := math.Min(a, b) + cost
	if math.Abs(a-b) < cost {
		next = (a + b + math.Sqrt(twice*cost*cost-(a-b)*(a-b))) / twice
	}
	if next >= field.time[at] {
		return 0
	}
	fell := field.time[at] - next
	field.time[at] = next
	return fell
}

// at is the time at a column and a row, infinite off the grid.
func (field *sweep) at(column, row int) float64 {
	if column < 0 || row < 0 || column >= field.across || row >= field.down {
		return math.Inf(1)
	}
	return field.time[row*field.across+column]
}

// fill gives every sample whose lag is not known the mean of its known
// neighbours', front by front from those known, so the lag is whole over
// land and over water the waves did not reach. A grid with nothing known
// is left at nought.
func (shore *coast) fill(lags []float64, known []bool) {
	front := shore.nextFront(known, nil)
	for len(front) > 0 {
		for _, at := range front {
			lags[at] = shore.meanKnown(lags, known, at)
		}
		for _, at := range front {
			known[at] = true
		}
		front = shore.nextFront(known, front)
	}
}

// nextFront is the unknown samples beside a known one: beside those of the
// last front, or, with none, beside any.
func (shore *coast) nextFront(known []bool, last []int) []int {
	var front []int
	seen := make(map[int]bool)
	look := func(at int) {
		for _, next := range shore.besides(at) {
			if !known[next] && !seen[next] {
				seen[next] = true
				front = append(front, next)
			}
		}
	}
	if last != nil {
		for _, at := range last {
			look(at)
		}
		return front
	}
	for at := range known {
		if known[at] {
			look(at)
		}
	}
	return front
}

// besides are the samples the ground's triangles join to one by an edge.
func (shore *coast) besides(at int) []int {
	column, row := at%shore.across, at/shore.across
	var near []int
	for _, step := range joined {
		next, down := column+step[0], row+step[1]
		if next >= 0 && down >= 0 && next < shore.across && down < shore.down {
			near = append(near, down*shore.across+next)
		}
	}
	return near
}

// meanKnown is the mean lag of a sample's known neighbours.
func (shore *coast) meanKnown(lags []float64, known []bool, at int) float64 {
	var sum float64
	var count int
	for _, next := range shore.besides(at) {
		if known[next] {
			sum += lags[next]
			count++
		}
	}
	return sum / float64(count)
}

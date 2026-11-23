// This file reads values held at a coast's samples between them, on the
// ground's triangles, and finds the swash's reach up the beach.

package arid

import "math"

// blend is a value held at the coast's samples, read at a spot: its
// corners' values, weighted as the spot's place among them is.
func (shore *coast) blend(place spot, values []float64) float64 {
	var total float64
	for index, corner := range place.corners {
		total += place.weights[index] * values[corner]
	}
	return total
}

// faceRise is how much a value held at the samples rises a metre east and
// a metre south over a spot's triangle, the way triangle finds the
// ground's rise: upper triangles run first corner, the one east, the last.
func (shore *coast) faceRise(place spot, values []float64) (east,
	south float64) {
	first, middle, last := values[place.corners[0]],
		values[place.corners[1]], values[place.corners[2]]
	if place.corners[1] == place.corners[0]+1 {
		return (middle - first) / shore.step, (last - middle) / shore.step
	}
	return (last - middle) / shore.step, (middle - first) / shore.step
}

// slant is a value held at the samples, read at a spot, and its rise a
// metre east and south there. Off the grid the value is held to the edge,
// as locate holds the ground, so it does not rise along a held axis.
func (shore *coast) slant(place spot, values []float64, east,
	south float64) (value, eastward, southward float64) {
	eastward, southward = shore.faceRise(place, values)
	heldEast, heldSouth := shore.held(east, south)
	if heldEast {
		eastward = 0
	}
	if heldSouth {
		southward = 0
	}
	return shore.blend(place, values), eastward, southward
}

// held is whether a place is off the grid east or west, and north or
// south, so that locate holds it to the edge along that axis.
func (shore *coast) held(east, south float64) (alongEast, alongSouth bool) {
	across := (east - shore.east) / shore.step
	down := (south - shore.south) / shore.step
	return across < 0 || across > float64(shore.across-1),
		down < 0 || down > float64(shore.down-1)
}

// gradient is the ground's rise a metre east and south at a sample, by
// the difference across its neighbours, or to its one neighbour at an
// edge.
func (shore *coast) gradient(at int) (east, south float64) {
	column, row := at%shore.across, at/shore.across
	west, eastward := max(column-1, 0), min(column+1, shore.across-1)
	north, southward := max(row-1, 0), min(row+1, shore.down-1)
	east = (shore.height[row*shore.across+eastward] -
		shore.height[row*shore.across+west]) /
		(float64(eastward-west) * shore.step)
	south = (shore.height[southward*shore.across+column] -
		shore.height[north*shore.across+column]) /
		(float64(southward-north) * shore.step)
	return east, south
}

// waterline is whether a sample is water the open sea reaches with land
// beside it, joined by an edge of the ground's triangles.
func (shore *coast) waterline(at int) bool {
	if shore.height[at] >= 0 || shore.cut[at] {
		return false
	}
	for _, next := range shore.besides(at) {
		if shore.height[next] >= 0 {
			return true
		}
	}
	return false
}

// reaching is the run-up a beach face carries: its mean and deviation.
type reaching struct{ mean, deviation float64 }

// swash works out the wet band. At each sample of the waterline the waves
// break by the beach face's slope, and are counted by kind; at each sample
// of land beside it the run-up is Stockdon's for that land's slope.
//
// From there the run-up spreads up the land joined to it, while the land
// is under the run-up two in a hundred pass, and each sample's wetness is
// the share of run-ups that reach it, a normal of the run-up's mean and
// deviation.
//
// Water the sea reaches is wet throughout.
func (waves *surf) swash(given arrival) {
	shore := waves.shore
	waves.wetness = make([]float64, len(shore.height))
	carried := make([]reaching, len(shore.height))
	visited := make([]bool, len(shore.height))
	var queue []int
	for at := range shore.height {
		if shore.height[at] < 0 && !shore.cut[at] {
			waves.wetness[at] = 1
		}
		if shore.waterline(at) {
			waves.count(at, given)
			queue = waves.beach(at, given, carried, visited, queue)
		}
	}
	for len(queue) > 0 {
		at := queue[len(queue)-1]
		queue = waves.climb(at, carried, visited, queue[:len(queue)-1])
	}
}

// count adds a waterline sample's breaker to the line, by the slope there.
func (waves *surf) count(at int, given arrival) {
	east, south := waves.shore.gradient(at)
	switch breakerKind(math.Hypot(east, south), given.significant,
		given.length) {
	case spilling:
		waves.counted.Spilling++
	case plunging:
		waves.counted.Plunging++
	default:
		waves.counted.Surging++
	}
}

// beach starts the run-up on the land beside a waterline sample, each by
// its own slope, and queues it.
func (waves *surf) beach(at int, given arrival, carried []reaching,
	visited []bool, queue []int) []int {
	shore := waves.shore
	for _, next := range shore.besides(at) {
		if shore.height[next] < 0 || visited[next] {
			continue
		}
		east, south := shore.gradient(next)
		mean, deviation := runup(math.Hypot(east, south), given.significant,
			given.length)
		reach := mean + exceedanceDeviations*deviation
		waves.counted.RunupMost = math.Max(waves.counted.RunupMost, reach)
		if waves.counted.RunupLeast == 0 || reach < waves.counted.RunupLeast {
			waves.counted.RunupLeast = reach
		}
		queue = waves.wet(next, reaching{mean, deviation}, carried, visited,
			queue)
	}
	return queue
}

// climb carries a land sample's run-up to the land joined to it.
func (waves *surf) climb(at int, carried []reaching, visited []bool,
	queue []int) []int {
	shore := waves.shore
	for _, next := range shore.besides(at) {
		if shore.height[next] >= 0 && !visited[next] {
			queue = waves.wet(next, carried[at], carried, visited, queue)
		}
	}
	return queue
}

// wet gives a land sample its wetness from a run-up, if the land is under
// the run-up two in a hundred pass, and queues it to carry it on.
func (waves *surf) wet(at int, given reaching, carried []reaching,
	visited []bool, queue []int) []int {
	height := waves.shore.height[at]
	if !(given.deviation > 0) ||
		height >= given.mean+exceedanceDeviations*given.deviation {
		return queue
	}
	visited[at] = true
	carried[at] = given
	waves.wetness[at] = math.Erfc((height-given.mean)/
		(given.deviation*math.Sqrt2)) / twice
	waves.counted.Swashed++
	return append(queue, at)
}

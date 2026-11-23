package skyglow

import (
	"fmt"
	"math"
	"sync"

	"github.com/janearc/slayer/maths"
)

// fourPi is the solid angle of the whole sphere, in steradians: a point
// source's intensity over the sphere is its power, and the mean radiance
// it makes at a distance r is its intensity over four pi r squared.
const fourPi = 4 * math.Pi

// mostNodes is the most nodes a grid may have. The scattering sum is
// every node to every node, so the work grows as the square of this: at
// forty thousand a pass is sixteen hundred million pairs.
const mostNodes = 40000

// leastNodes is the fewest nodes along an axis: two, since the grid's
// values are read between them.
const leastNodes = 2

// Region is the box the glow is solved over, in the scene's space: x
// east, y up, z south. Outside it the field holds its edge value.
type Region struct {
	Low, High maths.Vec
}

// grid is the air cut into cells, each with its value at the middle:
// cells across east, up and south, and what each level of height
// holds.
type grid struct {
	low     maths.Vec
	step    [3]float64
	count   [3]int
	volume  float64
	nodes   []maths.Vec
	heights []float64
	density []float64
	column  []float64
}

// newGrid cuts the region into the counts of cells, or says why it
// cannot be done.
func newGrid(air Air, region Region, count [3]int) (*grid, error) {
	total := 1
	for _, each := range count {
		if each < leastNodes {
			return nil, fmt.Errorf("skyglow: %d nodes along an axis, "+
				"fewer than %d", each, leastNodes)
		}
		total *= each
	}
	if total > mostNodes {
		return nil, fmt.Errorf("skyglow: %d nodes, more than %d", total,
			mostNodes)
	}
	span := region.High.Sub(region.Low)
	if !(span.X > 0 && span.Y > 0 && span.Z > 0) {
		return nil, fmt.Errorf("skyglow: a region with no volume")
	}
	made := &grid{low: region.Low, count: count}
	made.step = [3]float64{span.X / float64(count[0]),
		span.Y / float64(count[1]), span.Z / float64(count[2])}
	made.volume = made.step[0] * made.step[1] * made.step[2]
	made.lay(air)
	return made, nil
}

// lay fills in the places of the nodes and what the air is at each
// level: level by level up, then south, then east.
func (made *grid) lay(air Air) {
	for level := range made.count[1] {
		height := made.low.Y + (float64(level)+half)*made.step[1]
		made.heights = append(made.heights, height)
		made.density = append(made.density, air.density(height))
		made.column = append(made.column, air.column(height))
		for south := range made.count[2] {
			for east := range made.count[0] {
				made.nodes = append(made.nodes, maths.V(
					made.low.X+(float64(east)+half)*made.step[0], height,
					made.low.Z+(float64(south)+half)*made.step[2]))
			}
		}
	}
}

// half is the middle of a cell, as a share of its step.
const half = 0.5

// perLevel is how many nodes are in one level of height.
func (made *grid) perLevel() int { return made.count[0] * made.count[2] }

// leastRange is the nearest a source is taken to be to a node, in
// metres: half the smallest step. A source nearer than that is smeared
// to this distance, which keeps its light finite.
func (made *grid) leastRange() float64 {
	return half * math.Min(made.step[0], math.Min(made.step[1],
		made.step[2]))
}

// between is the optical depth of a straight path of a length between
// two levels of height: the change in the column over the change in
// height, or the density of the level for a path along one.
func (made *grid) between(from, to int, length float64) float64 {
	if from == to {
		return length * made.density[from]
	}
	return length * (made.column[to] - made.column[from]) /
		(made.heights[to] - made.heights[from])
}

// parallel runs body over the ranges of workers' share of n items, each
// to its own slice of the output, so no two workers write the same
// place and the sums come out the same however many there are.
func parallel(workers, n int, body func(from, to int)) {
	share := (n + workers - 1) / workers
	var group sync.WaitGroup
	for from := 0; from < n; from += share {
		to := min(from+share, n)
		group.Add(1)
		go func() {
			defer group.Done()
			body(from, to)
		}()
	}
	group.Wait()
}

// reached is how far a path is, taken to be no nearer than least, and
// its unit way: up for a path of no length.
func reached(path maths.Vec, least float64) (float64, maths.Vec) {
	length := path.Len()
	if length == 0 {
		return least, maths.Up
	}
	return math.Max(length, least), path.Scale(1 / length)
}

// first is the first order of scattering at every node: the light
// that reaches it straight from the sources. lit holds all of it, which
// the orders past the first scatter on; own holds all but the lamps the
// tracer scatters itself, which is what the glow adds.
func (made *grid) first(air Air, sources []Source, workers int) (
	own, lit []maths.Light) {
	own = make([]maths.Light, len(made.nodes))
	lit = make([]maths.Light, len(made.nodes))
	least := made.leastRange()
	parallel(workers, len(made.nodes), func(from, to int) {
		for index := from; index < to; index++ {
			place := made.nodes[index]
			for _, source := range sources {
				reach, out := reached(place.Sub(source.At), least)
				tau := air.optical(source.At.Y, place.Y, reach)
				got := source.toward(out).Scale(
					math.Exp(-tau) / (fourPi * reach * reach))
				lit[index] = lit[index].Add(got)
				if !source.Traced {
					own[index] = own[index].Add(got)
				}
			}
		}
	})
	return own, lit
}

// scatter is the next order of scattering from the last: every node
// scatters an intensity of the albedo times its extinction times its
// volume times the mean radiance there, alike every way, and every
// other node takes the light of that as of a point source. A node takes
// from itself the light of its own cell, a ball of the cell's volume
// glowing evenly, whose mean over itself is its source times its radius.
func (made *grid) scatter(air Air, carried []maths.Light,
	workers int) []maths.Light {
	powers := make([]maths.Light, len(carried))
	for index, value := range carried {
		level := index / made.perLevel()
		powers[index] = value.Scale(air.Albedo * made.density[level] *
			made.volume)
	}
	radius := math.Cbrt(3 * made.volume / fourPi)
	next := make([]maths.Light, len(carried))
	least := made.leastRange()
	parallel(workers, len(carried), func(from, to int) {
		for index := from; index < to; index++ {
			level := index / made.perLevel()
			next[index] = made.take(index, level, powers, least).Add(
				carried[index].Scale(air.Albedo * made.density[level] *
					radius))
		}
	})
	return next
}

// take is the light the powers of every node but one send to it, each a
// point source, through the air between.
func (made *grid) take(index, level int, powers []maths.Light,
	least float64) maths.Light {
	var red, green, blue float64
	place := made.nodes[index]
	for other, power := range powers {
		if other == index || power == (maths.Light{}) {
			continue
		}
		reach, _ := reached(place.Sub(made.nodes[other]), least)
		tau := made.between(other/made.perLevel(), level, reach)
		weight := math.Exp(-tau) / (fourPi * reach * reach)
		red += power.R * weight
		green += power.G * weight
		blue += power.B * weight
	}
	return maths.RGB(red, green, blue)
}

// scattered is how much light a field of mean radiances has scattered:
// the sum over the nodes of the luminance times the extinction times the
// volume. It is what the camera is to see of the order, to within the
// albedo, and so what the orders are compared by.
func (made *grid) scattered(values []maths.Light) float64 {
	total := 0.0
	for index, value := range values {
		total += value.Luminance() * made.density[index/made.perLevel()]
	}
	return total * made.volume
}

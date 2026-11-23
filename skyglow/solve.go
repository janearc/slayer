package skyglow

import (
	"fmt"
	"math"
	"runtime"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
)

// Default grid and series choices. The nodes are cells across east, up
// and south: a thousand to a level, ten levels, ten thousand in all,
// which is a hundred million pairs to an order. Six orders is far more
// than a haze of any visibility asks, since each is about the albedo
// times the optical depth of the one before; a hundredth is the share of
// the whole an order may add and be left as the last.
const (
	defaultAcross    = 32
	defaultLevels    = 10
	defaultOrders    = 6
	defaultTolerance = 0.01
)

// padCap is the most, in metres, the region reaches past the emitters on
// each side. Past a unit of optical depth most of a source's light is
// gone, so the pad is that distance, held to this where the air is thin
// enough for it to be many kilometres. Chosen.
const padCap = 2000.0

// Options says how fine the grid is and how many orders of scattering to
// work.
type Options struct {
	// Nodes is the cells across east, up and south.
	Nodes [3]int
	// Orders is the most orders of scattering to work, one or more.
	Orders int
	// Tolerance is the share of the whole an order adds, under which the
	// series stops after it.
	Tolerance float64
	// Workers is how many goroutines share the sums.
	Workers int
}

// DefaultOptions are the grid and series above, on every processor the
// program may use.
func DefaultOptions() Options {
	return Options{Nodes: [3]int{defaultAcross, defaultLevels,
		defaultAcross}, Orders: defaultOrders, Tolerance: defaultTolerance,
		Workers: runtime.GOMAXPROCS(0)}
}

// Report says what a solve did and how far it can be trusted.
type Report struct {
	// Nodes and Spacing are the grid's counts and its cell sizes in
	// metres, east, up and south.
	Nodes   [3]int
	Spacing [3]float64
	// Sources is how many sources were solved, and Traced how many of
	// them are lamps the tracer scatters itself.
	Sources, Traced int
	// Shares is the share of all the scattered light each order is, the
	// first order first.
	Shares []float64
	// Tail is the share of the whole the orders past the last are
	// likely to add, taking them to fall off by the ratio of the last two
	// each; nought for a series of one order.
	Tail float64
	// Peak is the greatest mean radiance in the field, per channel.
	Peak maths.Light
}

// Field is the glow the air holds, as mean radiance at the middle of
// each cell of a grid, read between them.
type Field struct {
	// Medium is the air the field is of: the one volume whose scattering
	// takes it.
	Medium medium.Medium
	grid   *grid
	values []maths.Light
	report Report
}

// Report is what the solve did.
func (field *Field) Report() Report { return field.report }

// Lights says whether the field is the light of the given air: the
// tracer asks it of every volume it scatters in, and only the air the
// field was solved for takes it.
func (field *Field) Lights(made medium.Medium) bool {
	return made == field.Medium
}

// Mean is the mean radiance of the light the air has gathered, at a
// place: the eight nearest nodes' values, weighted by nearness. Beyond
// the grid it holds the value at the edge.
func (field *Field) Mean(at maths.Vec) maths.Light {
	made := field.grid
	east, eastShare := between(at.X-made.low.X, made.step[0], made.count[0])
	up, upShare := between(at.Y-made.low.Y, made.step[1], made.count[1])
	south, southShare := between(at.Z-made.low.Z, made.step[2],
		made.count[2])
	var mean maths.Light
	for corner := range 8 {
		weight, index := 1.0, 0
		for axis, share := range [3]float64{eastShare, upShare,
			southShare} {
			if corner>>axis&1 == 1 {
				weight *= share
			} else {
				weight *= 1 - share
			}
		}
		index = field.at(east+corner&1, up+corner>>1&1, south+corner>>2&1)
		mean = mean.Add(field.values[index].Scale(weight))
	}
	return mean
}

// at is the place of a node in the values, its coordinates held to the
// grid.
func (field *Field) at(east, up, south int) int {
	made := field.grid
	east = min(east, made.count[0]-1)
	up = min(up, made.count[1]-1)
	south = min(south, made.count[2]-1)
	return (up*made.count[2]+south)*made.count[0] + east
}

// between is where an offset from the grid's edge falls among cell
// middles: the lower cell and how far toward the next, with offsets
// outside the middles held to the first and the last.
func between(offset, step float64, count int) (int, float64) {
	position := math.Min(math.Max(offset/step-half, 0), float64(count-1))
	lower := min(int(position), count-leastNodes)
	return lower, position - float64(lower)
}

// RegionAround is the region the glow of some sources is solved over:
// their extent east and south, padded by a unit of optical depth at the
// base of the air, held to padCap, and the air's own height up.
func RegionAround(air Air, sources []Source) Region {
	pad := math.Min(1/air.Extinction, padCap)
	low := maths.V(math.Inf(1), air.Base, math.Inf(1))
	high := maths.V(math.Inf(-1), air.Base+air.Depth, math.Inf(-1))
	for _, source := range sources {
		low.X, low.Z = math.Min(low.X, source.At.X),
			math.Min(low.Z, source.At.Z)
		high.X, high.Z = math.Max(high.X, source.At.X),
			math.Max(high.Z, source.At.Z)
	}
	return Region{Low: low.Sub(maths.V(pad, 0, pad)),
		High: high.Add(maths.V(pad, 0, pad))}
}

// Solve is the glow of the sources in the air, over the region, as the
// package says: the first order from the sources, then the orders of
// scattering one after another until one adds less than the tolerance
// or the orders run out. Nothing shining gives a dark field, which is
// what air with no light in it is. It refuses air that is not air, a
// region with no volume, and options that cannot be worked.
func Solve(air Air, region Region, sources []Source, options Options) (
	*Field, error) {
	if !air.Valid() {
		return nil, fmt.Errorf("skyglow: the air %+v cannot be solved", air)
	}
	if options.Orders < 1 || options.Workers < 1 ||
		!(options.Tolerance > 0 && options.Tolerance < 1) {
		return nil, fmt.Errorf("skyglow: options %+v cannot be worked",
			options)
	}
	made, err := newGrid(air, region, options.Nodes)
	if err != nil {
		return nil, err
	}
	own, lit := made.first(air, sources, options.Workers)
	field := &Field{grid: made, values: own}
	field.series(air, lit, options)
	field.report.Nodes, field.report.Spacing = made.count, made.step
	field.report.Sources = len(sources)
	for _, each := range sources {
		if each.Traced {
			field.report.Traced++
		}
	}
	return field, nil
}

// series works the orders past the first, adding each into the field,
// and writes the shares and the tail into the report.
func (field *Field) series(air Air, lit []maths.Light, options Options) {
	made := field.grid
	powers := []float64{made.scattered(lit)}
	carried := lit
	for order := 2; order <= options.Orders; order++ {
		carried = made.scatter(air, carried, options.Workers)
		for index, value := range carried {
			field.values[index] = field.values[index].Add(value)
		}
		powers = append(powers, made.scattered(carried))
		if powers[len(powers)-1] <= options.Tolerance*sum(powers) {
			break
		}
	}
	field.report.Shares, field.report.Tail = shares(powers)
	for _, value := range field.values {
		field.report.Peak = maths.RGB(math.Max(field.report.Peak.R, value.R),
			math.Max(field.report.Peak.G, value.G),
			math.Max(field.report.Peak.B, value.B))
	}
}

// sum is the total of some numbers.
func sum(numbers []float64) float64 {
	total := 0.0
	for _, each := range numbers {
		total += each
	}
	return total
}

// shares is each order's share of the whole, and the share the orders
// past the last would add if each fell off by the ratio of the last two
// from the last: its own over one less the ratio, of the whole with it.
func shares(powers []float64) ([]float64, float64) {
	total := sum(powers)
	each := make([]float64, len(powers))
	if total == 0 {
		return each, 0
	}
	for index, power := range powers {
		each[index] = power / total
	}
	if len(powers) < leastNodes || powers[len(powers)-2] == 0 {
		return each, 0
	}
	ratio := powers[len(powers)-1] / powers[len(powers)-2]
	if ratio >= 1 {
		return each, 1
	}
	rest := powers[len(powers)-1] * ratio / (1 - ratio)
	return each, rest / (total + rest)
}

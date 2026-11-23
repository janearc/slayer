package planet

import (
	"fmt"
	"math"
	"slices"

	"github.com/janearc/slayer/maths/noise"
)

// grains are how a ring's matter clumps across its plane: by a pattern, its
// clumps from smallest to largest in metres, so many more of the small ones as
// a power law of this slope says, sheared along the orbit so many times longer
// than across.
//
// they leave so much of the ring empty, from zero, none, to one, nearly all.
// modelled: a pattern in layers about the ring, a layer for each doubling of
// size, each weighed by the slope; not particles.
type grains struct {
	name                        string
	seed                        uint64
	pattern                     pattern
	smallest, largest, shear    float64
	slope, clumping             float64
	threshold, softness, spread float64
}

// a pattern is a noise through space at a scale of one, about zero and mostly
// within one of it either way.
type pattern func(seed uint64, x, y, z float64) float64

// patterns is every pattern a ring's grains may take, by name:
//
//   - smooth: gradient noise, soft billows of thicker and thinner.
//   - cells: worley cells, clumps with gaps between, as moonlets clear.
//   - ridges: ridged noise, thin strands, as self-gravity's wakes.
//   - speckle: a fresh value in every cell, grit.
var patterns = map[string]pattern{
	"smooth":  noise.Gradient,
	"cells":   cells,
	"ridges":  ridges,
	"speckle": speckle,
}

// Patterns is the names of the patterns, in alphabetical order.
func Patterns() []string {
	var names []string
	for name := range patterns {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// cells is worley's: the distance to the nearest of a seeded point in each cell
// round about, near zero in a clump and up to one between them, turned so a
// clump stands thick and the gap thin.
func cells(seed uint64, x, y, z float64) float64 {
	nearest := math.Inf(whole)
	at := [axes]float64{x, y, z}
	for i := -adjacent; i <= adjacent; i++ {
		for k := -adjacent; k <= adjacent; k++ {
			for depth := -adjacent; depth <= adjacent; depth++ {
				nearest = math.Min(nearest, seeded(seed, at, i, k, depth))
			}
		}
	}
	return whole - double*math.Min(whole, nearest)
}

// The places of the three axes of space in a list of three numbers.
const (
	// xAxis is defined as the place of the x axis: the first.
	xAxis = 0
	// yAxis is defined as the place of the y axis: the second.
	yAxis = 1
	// zAxis is defined as the place of the z axis: the third.
	zAxis = 2
)

// seeded is the distance from a point to the seeded point of the cell
// so many cells off the one it is in, along each axis.
func seeded(seed uint64, at [axes]float64, i, k, depth int) float64 {
	cell := [axes]int64{int64(math.Floor(at[xAxis])) + int64(i),
		int64(math.Floor(at[yAxis])) + int64(k),
		int64(math.Floor(at[zAxis])) + int64(depth)}
	var sum float64
	for axis := range cell {
		point := float64(cell[axis]) + noise.Hash3(seed+uint64(axis),
			cell[xAxis], cell[yAxis], cell[zAxis])
		sum += (at[axis] - point) * (at[axis] - point)
	}
	return math.Sqrt(sum)
}

// ridges is ridged noise: high in thin strands where gradient noise crosses
// zero, low between.
func ridges(seed uint64, x, y, z float64) float64 {
	return whole - double*math.Abs(noise.Gradient(seed, x, y, z))
}

// speckled is a ring's speckle at a point: its seed, its cell along x, its
// cell round the circle, and how many cells go round it.
type speckled struct {
	seed                 uint64
	reach, round, around float64
}

// at is the fresh value of the cell so many cells along x and round the
// circle from the point's own, from zero to one.
func (grid speckled) at(along, roundBy float64) float64 {
	cell := math.Mod(grid.round+roundBy, grid.around)
	return noise.Hash3(grid.seed, int64(grid.reach+along), int64(cell),
		int64(grid.around))
}

// speckle is a fresh value at each cell, from minus one to one, eased
// to its neighbours' across the cell so no cell shows a hard edge.
//
// its cells are counted along x, and round the circle y and z lie on, a
// whole number of them to its circumference: square cells in y and z
// would cut the ring along straight lines across its plane.
func speckle(seed uint64, x, y, z float64) float64 {
	circumference := circle * math.Hypot(y, z)
	around := math.Max(whole, math.Round(circumference))
	turn := (math.Atan2(z, y) + math.Pi) / circle * around
	reach, round := math.Floor(x), math.Floor(turn)
	grid := speckled{seed: seed, reach: reach, round: round, around: around}
	inward, outward := eased(x-reach), eased(turn-round)
	near := grid.at(nothing, nothing) +
		(grid.at(nothing, whole)-grid.at(nothing, nothing))*outward
	far := grid.at(whole, nothing) +
		(grid.at(whole, whole)-grid.at(whole, nothing))*outward
	return double*(near+(far-near)*inward) - whole
}

// Smoothstep's cubic is three t squared less two t cubed.
const (
	// smoothstepCubic is defined as the three of smoothstep's cubic.
	smoothstepCubic = 3
	// smoothstepSquare is defined as the two of smoothstep's cubic.
	smoothstepSquare = 2
)

// eased is smoothstep's ease over a cell, from zero at its one edge to one at
// the other, its slope zero at both.
func eased(position float64) float64 {
	return position * position * (smoothstepCubic - smoothstepSquare*position)
}

// mostLayers is the most doublings of size grains are drawn with, so a
// wide range costs no more than this.
var mostLayers = pickCount("grain-most-layers")

// Grained is the ring with its matter clumped by a pattern of a name.
//
// from a seed, the clumps run from smallest to largest in metres; a slope says
// how many more small ones there are, a shear how many times longer along the
// orbit than across each is drawn, and a clumping, from zero to one, how much
// of the ring they leave empty. Sizes, slope and shear may not be endless.
func (ring Ring) Grained(name string, seed uint64, smallest, largest,
	slope, shear, clumping float64) (Ring, error) {
	drawn, known := patterns[name]
	if !known {
		return ring, fmt.Errorf("grains in a pattern %q; there are %v",
			name, Patterns())
	}
	if !(smallest > nothing) || !(largest >= smallest) ||
		!(slope > nothing) || !(shear >= whole) ||
		!(clumping >= nothing && clumping <= whole) ||
		endless(largest) || endless(slope) || endless(shear) {
		return ring, fmt.Errorf("grains from %g to %g m, slope %g, "+
			"shear %g, clumping %g", smallest, largest, slope, shear,
			clumping)
	}
	ring.grains = grains{name: name, seed: seed, pattern: drawn,
		smallest: smallest, largest: largest, shear: shear, slope: slope,
		clumping: clumping}
	ring.grains.settle()
	return ring, nil
}

// endless is whether an amount is plus or minus infinity.
func endless(amount float64) bool { return math.IsInf(amount, nothing) }

// emptiest is the share of a ring's plane its grains leave empty at the
// greatest clumping, from picks.yaml.
var emptiest = pick("grain-emptiest")

// filled is the share of a ring's plane its grains leave holding matter at a
// clumping: all of it at zero, what is left of the plane by emptiest at one.
func filled(clumping float64) float64 { return whole - emptiest*clumping }

// samples is how many points of the plane a ring's grains are read at
// to find where their fill's edge falls.
var samples = pickCount("grain-samples")

// tailShare is the tails of the sampled values the edge's softness is read
// between: the sample at one over this from either end.
var tailShare = pickCount("grain-tail-share")

// shapeSeedOffset and spreadSeedOffset are what a ring's seed is mixed with
// to draw the points its grains are read at, and the pattern that finds the
// spread of one layer, so the two do not repeat the pattern itself.
var (
	shapeSeedOffset  = uint64(pickCount("grain-point-seed-offset"))
	spreadSeedOffset = uint64(pickCount("grain-spread-seed-offset"))
)

// settleSpan is how many of the largest clumps across the points its grains
// are read at lie, and settleDepth where on the pattern's third axis the
// spread is read; both are in picks.yaml.
var (
	settleSpan  = pick("grain-settle-span")
	settleDepth = pick("grain-settle-depth")
)

// softShare is how much of the spread between the sampled tails a clump's
// edge is soft by, from picks.yaml.
var softShare = pick("grain-soft-share")

// settle finds the grains' threshold, the value of their pattern above which a
// filled share of the plane lies, by reading it at points across many of its
// largest clumps; the softness of a clump's edge, a share of the layers'
// spread; and the spread of one layer, its pattern's standard deviation.
func (made *grains) settle() {
	values := make([]float64, samples)
	one := made.seed ^ spreadSeedOffset
	var sum, squares float64
	for i := range values {
		x := noise.Hash(made.seed^shapeSeedOffset,
			int64(double*i)) * settleSpan * made.largest
		z := noise.Hash(made.seed^shapeSeedOffset,
			int64(double*i+adjacent)) * settleSpan * made.largest
		values[i], _ = made.layered(x, z, nothing)
		read := made.pattern(one, x/made.largest, z/made.largest,
			settleDepth)
		sum, squares = sum+read, squares+read*read
	}
	slices.Sort(values)
	at := int((whole - filled(made.clumping)) * float64(samples-adjacent))
	made.threshold = values[at]
	made.softness = softShare * (values[samples-adjacent-samples/tailShare] -
		values[samples/tailShare])
	mean := sum / float64(samples)
	variance := math.Max(nothing, squares/float64(samples)-mean*mean)
	made.spread = math.Sqrt(variance)
}

// layered is the grains' pattern in layers at a point of the plane, a layer for
// each doubling of size from largest to smallest, each weighed by the slope;
// and how much of it is hidden, finer than the finest a ray there sees, in
// metres.
//
// a layer of clumps twice the finest or more is seen whole, one no larger than
// it is taken at its mean, zero, as a pixel takes what it cannot tell apart,
// and one between in part; what is hidden is the standard deviation the unseen
// layers would have added.
//
// the pattern is read on a cylinder about the ring: its reach one way, and
// round the orbit shrunk by the shear, so a clump is drawn out along its orbit
// and no seam runs where the angle comes round.
func (made grains) layered(x, z, finest float64) (sum, hidden float64) {
	reach := math.Hypot(x, z)
	round := math.Atan2(z, x)
	along := reach / made.shear
	size := made.largest
	var unseen float64
	for i := nothing; i < mostLayers && size >= made.smallest; i++ {
		strength := made.weight(size)
		var seen float64 = whole
		if finest > nothing {
			seen = math.Max(nothing, math.Min(whole, size/finest-whole))
		}
		unseen += (whole - seen*seen) * strength * strength
		if seen > nothing {
			sum += seen * strength * made.pattern(made.seed+uint64(i),
				reach/size, along*math.Cos(round)/size,
				along*math.Sin(round)/size)
		}
		size /= double
	}
	return sum, math.Sqrt(unseen) * made.spread
}

// hiddenWidths is how many standard deviations of the hidden layers wide
// smoothstep's edge is taken over, so its slope at the middle is a normal
// spread's; its value is in picks.yaml.
var hiddenWidths = pick("grain-hidden-widths")

// weight is how strongly a layer of clumps of a size shows against the
// largest, by the power law of their sizes.
func (made grains) weight(size float64) float64 {
	return math.Pow(size/made.largest, (made.slope-double)/double)
}

// share is how much of the ring's mean depth stands at a point of its
// plane, for its grains, where a ray sees nothing finer than so many
// metres: one where there are none.
//
// where the pattern stands above its threshold there is matter, packed so
// the ring keeps its mean depth, and below it vacuum, with a soft edge
// between.
//
// where the ray's footprint holds clumps it cannot tell apart, the edge
// widens by what they hide, so the share is near the mean of the matter
// under the footprint, as the light of a pixel is.
//
// modelled: the hidden layers are taken as a normal spread, its slope at
// the middle matched by smoothstep's over hiddenWidths standard deviations.
func (made grains) share(x, z, finest float64) float64 {
	if made.clumping == nothing || made.pattern == nil {
		return whole
	}
	sum, hidden := made.layered(x, z, finest)
	width := made.softness + hiddenWidths*hidden
	edge := math.Max(nothing, math.Min(whole,
		(sum-made.threshold)/width+half))
	return eased(edge) / filled(made.clumping)
}

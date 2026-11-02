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
	nearest := math.Inf(1)
	at := [3]float64{x, y, z}
	for i := -1; i <= 1; i++ {
		for k := -1; k <= 1; k++ {
			for depth := -1; depth <= 1; depth++ {
				nearest = math.Min(nearest, seeded(seed, at, i, k, depth))
			}
		}
	}
	return 1 - 2*math.Min(1, nearest)
}

// seeded is the distance from a point to the seeded point of the cell
// so many cells off the one it is in, along each axis.
func seeded(seed uint64, at [3]float64, i, k, depth int) float64 {
	cell := [3]int64{int64(math.Floor(at[0])) + int64(i),
		int64(math.Floor(at[1])) + int64(k),
		int64(math.Floor(at[2])) + int64(depth)}
	sum := 0.0
	for axis := range cell {
		point := float64(cell[axis]) + noise.Hash3(seed+uint64(axis),
			cell[0], cell[1], cell[2])
		sum += (at[axis] - point) * (at[axis] - point)
	}
	return math.Sqrt(sum)
}

// ridges is ridged noise: high in thin strands where gradient noise crosses
// zero, low between.
func ridges(seed uint64, x, y, z float64) float64 {
	return 1 - 2*math.Abs(noise.Gradient(seed, x, y, z))
}

// speckle is a fresh value at each cell, from minus one to one, eased
// to its neighbours' across the cell so no cell shows a hard edge.
//
// its cells are counted along x, and round the circle y and z lie on, a
// whole number of them to its circumference: square cells in y and z
// would cut the ring along straight lines across its plane.
func speckle(seed uint64, x, y, z float64) float64 {
	around := math.Max(1, math.Round(2*math.Pi*math.Hypot(y, z)))
	turn := (math.Atan2(z, y) + math.Pi) / (2 * math.Pi) * around
	reach, round := math.Floor(x), math.Floor(turn)
	valued := func(i, k float64) float64 {
		cell := math.Mod(round+k, around)
		return noise.Hash3(seed, int64(reach+i), int64(cell), int64(around))
	}
	inward, outward := eased(x-reach), eased(turn-round)
	near := valued(0, 0) + (valued(0, 1)-valued(0, 0))*outward
	far := valued(1, 0) + (valued(1, 1)-valued(1, 0))*outward
	return 2*(near+(far-near)*inward) - 1
}

// eased is smoothstep's ease over a cell, from zero at its one edge to one at
// the other, its slope zero at both.
func eased(t float64) float64 { return t * t * (3 - 2*t) }

// mostLayers is the most doublings of size grains are drawn with, so a
// wide range costs no more than this: picked.
const mostLayers = 14

// Grained is the ring with its matter clumped by a pattern of a name.
//
// from a seed, the clumps run from smallest to largest in metres; a slope says
// how many more small ones there are, a shear how many times longer along the
// orbit than across each is drawn, and a clumping, from zero to one, how much
// of the ring they leave empty.
func (ring Ring) Grained(name string, seed uint64, smallest, largest,
	slope, shear, clumping float64) (Ring, error) {
	drawn, known := patterns[name]
	if !known {
		return ring, fmt.Errorf("grains in a pattern %q; there are %v",
			name, Patterns())
	}
	if !(smallest > 0) || !(largest >= smallest) || !(slope > 0) ||
		!(shear >= 1) || !(clumping >= 0 && clumping <= 1) {
		return ring, fmt.Errorf("grains from %g to %g m, slope %g, "+
			"shear %g, clumping %g", smallest, largest, slope, shear,
			clumping)
	}
	ring.grains = grains{seed: seed, pattern: drawn, smallest: smallest,
		largest: largest, shear: shear, slope: slope, clumping: clumping}
	ring.grains.settle()
	return ring, nil
}

// filled is the share of a ring's plane its grains leave holding matter at a
// clumping: all of it at zero, a twentieth at one.
func filled(clumping float64) float64 { return 1 - 0.95*clumping }

// samples is how many points of the plane a ring's grains are read at
// to find where their fill's edge falls: picked.
const samples = 4096

// settle finds the grains' threshold, the value of their pattern above which a
// filled share of the plane lies, by reading it at points across many of its
// largest clumps; the softness of a clump's edge, a tenth of the layers'
// spread; and the spread of one layer, its pattern's standard deviation.
func (made *grains) settle() {
	values := make([]float64, samples)
	one, sum, squares := made.seed^0x0e, 0.0, 0.0
	for i := range values {
		x := noise.Hash(made.seed^0x5eed, int64(2*i)) * 64 * made.largest
		z := noise.Hash(made.seed^0x5eed, int64(2*i+1)) * 64 * made.largest
		values[i], _ = made.layered(x, z, 0)
		read := made.pattern(one, x/made.largest, z/made.largest, 0.5)
		sum, squares = sum+read, squares+read*read
	}
	slices.Sort(values)
	at := int((1 - filled(made.clumping)) * float64(samples-1))
	made.threshold = values[at]
	made.softness = 0.1 * (values[samples-1-samples/20] - values[samples/20])
	mean := sum / samples
	made.spread = math.Sqrt(math.Max(0, squares/samples-mean*mean))
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
	size, unseen := made.largest, 0.0
	for i := 0; i < mostLayers && size >= made.smallest; i++ {
		strength := math.Pow(size/made.largest, (made.slope-2)/2)
		seen := 1.0
		if finest > 0 {
			seen = math.Max(0, math.Min(1, size/finest-1))
		}
		unseen += (1 - seen*seen) * strength * strength
		if seen > 0 {
			sum += seen * strength * made.pattern(made.seed+uint64(i),
				reach/size, along*math.Cos(round)/size,
				along*math.Sin(round)/size)
		}
		size /= 2
	}
	return sum, math.Sqrt(unseen) * made.spread
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
// the middle matched by smoothstep's over 3.75 standard deviations.
func (made grains) share(x, z, finest float64) float64 {
	if made.clumping == 0 || made.pattern == nil {
		return 1
	}
	sum, hidden := made.layered(x, z, finest)
	width := made.softness + 3.75*hidden
	edge := math.Max(0, math.Min(1, (sum-made.threshold)/width+0.5))
	return eased(edge) / filled(made.clumping)
}

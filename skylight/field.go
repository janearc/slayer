package skylight

import (
	"math"
	"slices"
)

// Field is the light Solve kept: at each kept height, for each local
// solar zenith, the radiance arriving at an observer looking each way,
// per unit solar irradiance perpendicular to the sun's rays at the top,
// per steradian; and the crossings behind each bin, to show its noise.
//
// A bin's radiance is the power of the crossings in it, each over its
// cosine from the vertical, over the area of its band of zenith on the
// kept sphere and its solid angle. Unscattered sunlight is not in it:
// Transmittance gives that.
type Field struct {
	shape
	air      air
	heights  []float64
	radiance []float64
	counts   []int64
}

// gather adds the walkers' tallies and budgets in their order, turns
// the tallies into radiance, and the budget's weights into power: the
// area of the band of the top sphere's disc they started on, over the
// photons.
func gather(made *air, grid Grid, walkers []*walker,
	photons int) (*Field, Budget, census) {
	disc := made.band.area()
	power := disc / float64(photons)
	field := &Field{shape: shape{grid.Zeniths, grid.Views, grid.Azimuths},
		air: *made, heights: slices.Clone(grid.Heights)}
	sums := make([]float64, len(walkers[none].tally.sums))
	field.counts = make([]int64, len(sums))
	spent := Budget{Launched: disc}
	var counted census
	for _, each := range walkers {
		for bin := range sums {
			sums[bin] += each.tally.sums[bin]
			field.counts[bin] += each.tally.counts[bin]
		}
		spent.add(each.spent, power)
		counted.events += each.counted.events
		counted.grounded += each.counted.grounded
	}
	field.radiance = field.radiant(sums, power)
	return field, spent, counted
}

// add adds a walker's budget, in weights, as power.
func (spent *Budget) add(weights Budget, power float64) {
	spent.Escaped += weights.Escaped * power
	spent.Air += weights.Air * power
	spent.Ground += weights.Ground * power
	spent.Capped += weights.Capped * power
}

// radiant turns the summed tallies into radiance: each bin's sum, times
// a photon's power, over its band's area and its solid angle. That is
// the view cosine's width times twice the azimuth's, since the azimuths
// either side of the sun are folded into one.
func (f *Field) radiant(sums []float64, power float64) []float64 {
	solid := two / float64(f.views) * fullTurn / float64(f.azimuths)
	radiance := make([]float64, len(sums))
	for keep, height := range f.heights {
		radius := f.air.radius + height
		for zenith := range f.zeniths {
			area := bandArea(radius, zenith, f.zeniths)
			first := f.index(keep, zenith, none, none)
			for bin := first; bin < first+f.views*f.azimuths; bin++ {
				radiance[bin] = sums[bin] * power / (area * solid)
			}
		}
	}
	return radiance
}

// bandArea is the area of the band of zenith on a sphere of radius, of
// zeniths equal bands from nought to pi: 2 pi r squared, times the
// cosine of its upper edge's zenith less that of its lower.
func bandArea(radius float64, zenith, zeniths int) float64 {
	width := math.Pi / float64(zeniths)
	return fullTurn * radius * radius * (math.Cos(float64(zenith)*width) -
		math.Cos(float64(zenith+one)*width))
}

// Radiance is the light arriving at an observer at height, in metres,
// where the sun's zenith is sunZenith, in radians, who looks at a
// cosine viewCosine from local up and azimuth radians from the sun, per
// unit solar irradiance at the top, per steradian.
//
// It is read at the nearest kept height, trilinearly between the
// centres of the bins about the look; past the first or last centre on
// an axis it holds that bin's value. The azimuth is folded: either side
// of the sun is the same.
func (f *Field) Radiance(height, sunZenith, viewCosine,
	azimuth float64) float64 {
	keep := f.nearest(height)
	zenith := axisOf(sunZenith/math.Pi, f.zeniths)
	view := axisOf((viewCosine+one)/two, f.views)
	round := axisOf(folded(azimuth)/math.Pi, f.azimuths)
	sum := 0.0
	for corner := range corners {
		z, zenithWeight := zenith.at(corner & one)
		v, viewWeight := view.at(corner >> one & one)
		a, roundWeight := round.at(corner >> two & one)
		sum += zenithWeight * viewWeight * roundWeight *
			f.radiance[f.index(keep, z, v, a)]
	}
	return sum
}

// Count is how many crossings are behind the bin the look falls in, at
// the nearest kept height, with the same arguments as Radiance: its
// noise is about one over the root of that.
func (f *Field) Count(height, sunZenith, viewCosine,
	azimuth float64) int64 {
	return f.counts[f.index(f.nearest(height),
		binOf(sunZenith/math.Pi, f.zeniths),
		binOf((viewCosine+one)/two, f.views),
		binOf(folded(azimuth)/math.Pi, f.azimuths))]
}

// Transmittance is the share of the direct sun's light, at the top,
// that reaches height, in metres, where the sun's zenith is sunZenith,
// in radians: exp of minus the optical depth along the straight line to
// the sun, through each shell, and nought where the planet is in the way.
func (f *Field) Transmittance(height, sunZenith float64) float64 {
	radius := f.air.radius + height
	along := radius * math.Cos(sunZenith)
	ground := height * (two*f.air.radius + height)
	if along < none && along*along > ground {
		return none
	}
	depth := 0.0
	for _, through := range f.air.layers {
		depth += through.extinction * (chord(radius, along, through.outer) -
			chord(radius, along, through.inner))
	}
	return math.Exp(-depth)
}

// nearest is the index of the kept height nearest height.
func (f *Field) nearest(height float64) int {
	best := none
	for index, kept := range f.heights {
		if math.Abs(kept-height) < math.Abs(f.heights[best]-height) {
			best = index
		}
	}
	return best
}

// an axis is where a look falls between the centres of one axis's
// bins: the bin at or below it, the one above, and how far toward that.
type axis struct {
	low, high int
	share     float64
}

// axisOf is where a share from nought to one falls between the centres
// of bins equal bins. Past the first or last centre it is at that one.
func axisOf(share float64, bins int) axis {
	place := min(max(share*float64(bins)-half, none), float64(bins-one))
	low := int(place)
	return axis{low: low, high: min(low+one, bins-one),
		share: place - float64(low)}
}

// at is the bin of one side of the axis, nought for the low one, and
// its weight.
func (a axis) at(side int) (int, float64) {
	if side == none {
		return a.low, one - a.share
	}
	return a.high, a.share
}

// folded is an azimuth taken to nought to pi: the sky either side of
// the sun's vertical is the same.
func folded(azimuth float64) float64 {
	return math.Abs(math.Remainder(azimuth, fullTurn))
}

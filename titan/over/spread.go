package over

import (
	"math"
	"slices"
)

// spreadReach is how many samples either side of a sample its
// neighbourhood reaches, across and down: the neighbourhood is seven
// samples square, 28 m at a spacing of 4 m. It is picked. It is the least
// that leaves many samples, 49, for the six numbers of the surface fitted
// to them, and it spans tens of metres: far more than one facet or one
// crease between two triangles of the survey.
const spreadReach = 3

// scaleReach is how far the wider neighbourhood reaches, thirteen samples
// square, across which the survey's fall of roughness with scale is read
// against the narrower one: nearly twice its width. It is picked.
const scaleReach = 6

// fitTerms is the numbers in the surface fitted to a neighbourhood: its
// height, its slope each way, and its bend across, down and between.
const fitTerms = 6

// The bounds the survey's fall of roughness with scale is kept within, as
// the power of width that roughness grows by.
const (
	// leastScaling is the square root: roughness that falls slower toward
	// small scales is noise of the survey, not of the ground. Picked.
	leastScaling = 0.5
	// mostScaling is the same slope at every scale. Picked: a residual
	// after a quadric fit may grow faster with width than this on smooth
	// ground, but made relief is never let fall faster toward small waves
	// than the same-slope law.
	mostScaling = 1.0
	// residualFloor is the share of a neighbourhood's sum of squares under
	// which what a fit leaves is taken as the arithmetic's rounding, not
	// roughness: far over a float's rounding, a millionth, and far under
	// any relief a survey holds. Picked.
	residualFloor = 1e-9
	// scaleStride is how many samples apart, across and down, the fall is
	// read: one in sixteen samples is enough for a median. Picked.
	scaleStride = 4
	// medianShare is the middle of a sorted list. Defined.
	medianShare = 0.5
)

// fitting is the inverse of a neighbourhood's normal matrix: what turns the
// sums of heights over a neighbourhood into the surface that fits it best.
type fitting [fitTerms][fitTerms]float64

// terms are the six things a fitted surface is made of at a place
// dx samples across and dz samples down from a neighbourhood's middle: one,
// each distance, each distance squared and the two together.
func terms(dx, dz float64) [fitTerms]float64 {
	return [fitTerms]float64{1, dx, dz, dx * dx, dx * dz, dz * dz}
}

// quadricFit is the fitting for a neighbourhood. It is the same for every
// neighbourhood, being laid on the same samples, so it is made once.
func quadricFit() fitting {
	return quadricFitOver(spreadReach)
}

// quadricFitOver is the fitting for a neighbourhood that reaches so many
// samples either side of its middle.
func quadricFitOver(reach int) fitting {
	var normal fitting
	for dz := -reach; dz <= reach; dz++ {
		for dx := -reach; dx <= reach; dx++ {
			each := terms(float64(dx), float64(dz))
			for i := range each {
				for j := range each {
					normal[i][j] += each[i] * each[j]
				}
			}
		}
	}
	return inverse(normal)
}

// inverse is a matrix turned over, by Gauss and Jordan's elimination. The
// matrix is positive definite, a sum of outer products, so no row is
// swapped.
func inverse(matrix fitting) fitting {
	var turned fitting
	for i := range turned {
		turned[i][i] = 1
	}
	for pivot := range matrix {
		scale := 1 / matrix[pivot][pivot]
		for j := range matrix {
			matrix[pivot][j] *= scale
			turned[pivot][j] *= scale
		}
		for row := range matrix {
			if row == pivot {
				continue
			}
			factor := matrix[row][pivot]
			for j := range matrix {
				matrix[row][j] -= factor * matrix[pivot][j]
				turned[row][j] -= factor * turned[pivot][j]
			}
		}
	}
	return turned
}

// spread is how broken the map is round a sample, in metres: the root mean
// square of what is left of the heights of its neighbourhood once the
// surface that best fits them, its slope and its bend, is taken away. A
// smooth rise of any slope, and a smooth rim, leave nothing. Round a sample
// whose neighbourhood leaves the map it is nought.
func (read *shape) spread(column, row int) float64 {
	return read.spreadOver(column, row, spreadReach, read.fit)
}

// spreadOver is spread over a neighbourhood that reaches so many samples
// either side, with that neighbourhood's fitting.
func (read *shape) spreadOver(column, row, reach int, fit fitting) float64 {
	if column < reach || row < reach || column+reach >= read.across ||
		row+reach >= read.down {
		return 0
	}
	centre := read.at(column, row)
	var moment [fitTerms]float64
	var squares float64
	for dz := -reach; dz <= reach; dz++ {
		for dx := -reach; dx <= reach; dx++ {
			rise := read.at(column+dx, row+dz) - centre
			for i, each := range terms(float64(dx), float64(dz)) {
				moment[i] += each * rise
			}
			squares += rise * rise
		}
	}
	samples := float64((2*reach + 1) * (2*reach + 1))
	left := squares - fit.explained(moment)
	if left <= residualFloor*squares {
		return 0
	}
	return math.Sqrt(left / samples)
}

// scaling is how the survey's roughness grows with the width it is read
// over, as a power of the width, read over the stretch: the median, over
// samples a stride apart, of how much rougher the wider neighbourhood is
// than the narrower, kept between the square root and the same slope at
// every scale. Where the stretch has no rough sample it is the latter.
func (read *shape) scaling(stretch Stretch) float64 {
	wide := quadricFitOver(scaleReach)
	ratio := math.Log(float64(2*scaleReach+1) / float64(2*spreadReach+1))
	var powers []float64
	last := [2]int{stretch.Column + stretch.Across, stretch.Row + stretch.Down}
	for row := stretch.Row; row <= last[1]; row += scaleStride {
		for column := stretch.Column; column <= last[0]; column += scaleStride {
			narrow := read.spread(column, row)
			broad := read.spreadOver(column, row, scaleReach, wide)
			if narrow > 0 && broad > 0 {
				powers = append(powers, math.Log(broad/narrow)/ratio)
			}
		}
	}
	if len(powers) == 0 {
		return mostScaling
	}
	slices.Sort(powers)
	middle := powers[int(medianShare*float64(len(powers)))]
	return min(mostScaling, max(leastScaling, middle))
}

// explained is the sum of squares of the heights that the fitted surface
// accounts for, given the moments of the heights over the terms.
func (fit fitting) explained(moment [fitTerms]float64) float64 {
	var sum float64
	for i := range moment {
		for j := range moment {
			sum += moment[i] * fit[i][j] * moment[j]
		}
	}
	return sum
}

// width is how wide a neighbourhood is, metres, from its first sample to
// its last and one more.
func (read *shape) width() float64 {
	return float64(2*spreadReach+1) * read.step
}

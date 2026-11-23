// This file is where the open sea breaks: the share of the surface at a
// place that is white water, from the waves' own downward acceleration.

package arid

import (
	"math"

	"github.com/janearc/slayer/oceantex"
)

// The two multipliers and three shifts of the SplitMix64 mixing step
// (Steele, Lea and Flood 2014, Fast splittable pseudorandom number
// generators, OOPSLA; the constants as in Vigna's splitmix64.c), and how
// many of the mixed bits a float64 holds exactly.
const (
	stirFirst    = 0xbf58476d1ce4e5b9
	stirSecond   = 0x94d049bb133111eb
	stirStep     = 0x9e3779b97f4a7c15
	stirHigh     = 30
	stirMiddle   = 27
	stirLow      = 31
	stirMantissa = 53
)

// whitecap is whether the surface at an east and a south is breaking: one
// where it is and nought where it is not, so that over any patch of sea
// its mean is the share of the patch that is white.
//
// A wave breaks where the surface's downward acceleration passes
// oceantex.BreakingAcceleration of gravity, the threshold of Snyder and
// Kennedy's mechanism for whitecaps.
//
// The acceleration is the bands' own
// (falling), cut by the share of the waves the coast lets stand, and the
// unseen waves': those are not held anywhere, so their part is a smooth
// noise of the variance they have (grain), which for free waves in deep
// water is their slope variance, both being the spectrum's fourth moment.
//
// It is the water breaking at this instant. The foam a breaker leaves
// behind it, which is most of the white a photograph of the sea shows, is
// not here: nothing remembers where a wave broke.
func (sea *Sea) whitecap(east, south float64) float64 {
	share, _, _ := sea.coast.meet(east, south)
	if !(share > 0) {
		return 0
	}
	falling := sea.falling(east, south) +
		math.Sqrt(sea.unseen)*sea.grain(east, south)
	if share*falling >= oceantex.BreakingAcceleration {
		return 1
	}
	return 0
}

// falling is the open surface's downward acceleration at an east and a
// south, as a share of gravity, from the waves the bands hold: each
// band's own (squeezed), the shorter bands read where the longer ones'
// water came from, as waves reads them.
func (sea *Sea) falling(east, south float64) float64 {
	var total float64
	last := len(sea.bands) - 1
	for index, band := range sea.bands {
		half := band.Extent / 2
		total += squeezed(band, half+east, half-south)
		if index < last {
			_, pushEast, pushNorth := carriedSample(band, half+east,
				half-south)
			east, south = east-pushEast, south+pushNorth
		}
	}
	return total
}

// squeezed is one crested band's downward acceleration at an east and a
// north on its tile, as a share of gravity.
//
// For free waves in deep water
// the surface's acceleration upward, over gravity, is the divergence of
// the water's sideways carrying: both are the sum of each wave's height
// times its wave number, signed.
//
// A crested band holds one less the
// inverse of its carrying's Jacobian, so the divergence is that inverse's
// trace over its determinant, less two; where the inverse is not a
// carrying at all the water has folded, and is breaking whatever the
// threshold.
func squeezed(band *oceantex.Field, east, north float64) float64 {
	xx := 1 - periodicSample(band, band.StretchXX, east, north)
	xy := -periodicSample(band, band.StretchXY, east, north)
	yy := 1 - periodicSample(band, band.StretchYY, east, north)
	det := xx*yy - xy*xy
	if !(det > 0) {
		return math.Inf(1)
	}
	return 2 - (xx+yy)/det
}

// grain is a smooth noise over the plane with a mean of nought and a
// variance of one everywhere: unit Gaussians at the corners of cells half
// the finest band's shortest wave a side, the longest half-wave the
// unseen waves have, blended on straight lines and scaled back to a
// variance of one.
//
// It stands for the unseen waves' part of the
// acceleration, whose own pattern no band holds: the pattern is noise
// and no sea's.
func (sea *Sea) grain(east, south float64) float64 {
	side := sea.finest / oceantex.CascadeCut()
	gx, gy := east/side, south/side
	xFloor, yFloor := math.Floor(gx), math.Floor(gy)
	tx, ty := gx-xFloor, gy-yFloor
	weights := [...]float64{(1 - tx) * (1 - ty), tx * (1 - ty),
		(1 - tx) * ty, tx * ty}
	var sum, squares float64
	for index, weight := range weights {
		sum += weight * gaussianAt(int64(xFloor)+int64(index%2),
			int64(yFloor)+int64(index/2))
		squares += weight * weight
	}
	return sum / math.Sqrt(squares)
}

// gaussianAt is a number drawn from the unit Gaussian for a corner of
// the grain's cells, the same every time for the same corner: Box and
// Muller's transform of two numbers stirred from the corner's place.
func gaussianAt(column, row int64) float64 {
	columnBits := stirred(uint64(column))
	bits := stirred(columnBits ^ uint64(row))
	first := unitOf(bits)
	second := unitOf(stirred(bits + stirStep))
	return math.Sqrt(-2*math.Log(1-first)) * math.Cos(2*math.Pi*second)
}

// stirred is bits mixed by SplitMix64's mixing step, so that neighbouring
// numbers give unlike ones.
func stirred(bits uint64) uint64 {
	bits ^= bits >> stirHigh
	bits *= stirFirst
	bits ^= bits >> stirMiddle
	bits *= stirSecond
	return bits ^ bits>>stirLow
}

// unitOf is a number from nought up to one made of the top bits of bits,
// as many as a float64 holds exactly.
func unitOf(bits uint64) float64 {
	return float64(bits>>(64-stirMantissa)) / (1 << stirMantissa)
}

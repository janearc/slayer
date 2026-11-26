package snakes

import "math"

// coatMarks holds the strength of a border, dark interior and pale island.
// The marks are illustrative; the species profile supplies colours and pitch.
type coatMarks struct{ border, dark, pale float64 }

// spot is one softened elliptical mark in the longitudinal and angular plane.
func spot(x, y, width, height float64) float64 {
	return math.Hypot(x/width, y/height)
}

func disk(distance, radius, feather float64) float64 {
	return 1 - smooth(radius-feather, radius+feather, distance)
}

// bodyMarks shares the coat rules across profiles in straight local space.
// turn is zero at the back, +/- one at the underside, and wraps at that seam.
func bodyMarks(kind string, phase, turn, feather float64) coatMarks {
	cell, across := math.Floor(phase), periodic(phase)
	side := math.Abs(turn)
	distance := math.Inf(1)
	var pale, ring float64
	switch kind {
	case "saddles":
		// Blotches are rounded and vary between adjacent cells. Their width
		// and centres are chosen, not species measurements.
		width := .35 + .08*math.Sin(cell*1.71)
		shift := .035 * math.Sin(cell*2.17)
		reach := .40 + .06*math.Cos(cell*1.29)
		warp := .10 * turn * math.Sin(turn*8+cell*1.43)
		jag := .10*math.Sin(turn*13+cell*1.31) + .045*math.Sin(turn*29+cell*.73)
		distance = spot(across-shift-warp, turn, width, reach) - jag
		flank := spot(periodic(phase+.32), side-.62, .18, .10)
		ring = disk(flank, 1, feather)
	case "spots":
		// Paired larger dorsal ovals and a smaller, hollow flank ocellus.
		shift := .02 * math.Sin(cell*1.37)
		width := .25 + .035*math.Sin(cell*1.61)
		left := spot(across-shift, turn-.18, width, .115)
		right := spot(across-shift, turn+.18, width, .115)
		distance = math.Min(left, right)
		flank := spot(across, side-.54, .18, .13)
		ring = disk(flank, 1, feather) * (1 - disk(flank, .42, feather))
		pale = disk(flank, .32, feather)
	case "reticulated":
		// Outlined rhombi lie along the back; light, ringed side islands
		// interrupt the regular net. Both are a simplified reconstruction.
		shift := .035 * math.Sin(cell*1.53)
		width := .43 + .04*math.Cos(cell*1.17)
		reach := .50 + .04*math.Sin(cell*1.89)
		diamond := math.Abs(across-shift)/width + side/reach
		ring = disk(math.Abs(diamond-1), .10, feather)
		flank := spot(across, side-.58, .18, .13)
		ring = math.Max(ring, disk(flank, 1, feather)*(1-disk(flank, .42, feather)))
		pale = disk(flank, .32, feather)
	case "triangles":
		distance = math.Abs(across)/.40 + math.Abs(side-.40)/.40
	case "bands":
		distance = math.Abs(across) / .30
	}
	border := disk(distance, 1, feather)
	dark := disk(distance, .72, feather)
	if kind == "spots" {
		// The large anaconda dorsal marks are dark rather than ringed.
		dark = border
	}
	return coatMarks{
		border: math.Max(border, ring),
		dark:   math.Max(dark, ring),
		pale:   pale,
	}
}

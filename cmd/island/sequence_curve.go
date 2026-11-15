package main

import "math"

// curveLocation shares a coordinate velocity across each interior waypoint.
// Position and look target are separate tracks; a fixed look target stays fixed.
func (sequence sequencePlan) curveLocation(leg int, share float64, target bool) location {
	point := func(key sequenceKey) location {
		if target {
			return key.Look
		}
		return key.At
	}
	return location{
		Longitude: sequence.coordinate(leg, share, func(key sequenceKey) float64 { return point(key).Longitude }),
		Latitude:  sequence.coordinate(leg, share, func(key sequenceKey) float64 { return point(key).Latitude }),
		Height:    sequence.coordinate(leg, share, func(key sequenceKey) float64 { return point(key).Height }),
	}
}

// coordinate uses cubic Hermite interpolation with shared endpoint slopes.
// Each slope has the segment's sign and no greater magnitude than its secant.
// The derivative's Bernstein coefficients are m0, 3*d-m0-m1 and m1, so these
// bounds preserve monotonicity within each leg. They avoid coordinate overshoot,
// not terrain collisions or changes in geographic speed or viewing angle.
func (sequence sequencePlan) coordinate(leg int, share float64, value func(sequenceKey) float64) float64 {
	from, to := sequence.Keys[leg], sequence.Keys[leg+1]
	a, b := value(from), value(to)
	if share <= 0 {
		return a
	}
	if share >= 1 {
		return b
	}
	span := to.Seconds - from.Seconds
	m0, m1 := sequence.coordinateSlope(leg, value), sequence.coordinateSlope(leg+1, value)
	u2, u3 := share*share, share*share*share
	return a + (b-a)*(3*u2-2*u3) + span*(m0*(u3-2*u2+share)+m1*(u3-u2))
}

// coordinateSlope chooses the slower neighbouring rate when direction agrees.
// Opposing or stationary legs meet at zero velocity; endpoints use their one
// adjacent secant. These are authored interpolation choices, not flight physics.
func (sequence sequencePlan) coordinateSlope(index int, value func(sequenceKey) float64) float64 {
	secant := func(leg int) float64 {
		a, b := sequence.Keys[leg], sequence.Keys[leg+1]
		return (value(b) - value(a)) / (b.Seconds - a.Seconds)
	}
	if index == 0 {
		return secant(0)
	}
	if index == len(sequence.Keys)-1 {
		return secant(index - 1)
	}
	before, after := secant(index-1), secant(index)
	if before == 0 || after == 0 || math.Signbit(before) != math.Signbit(after) {
		return 0
	}
	return math.Copysign(math.Min(math.Abs(before), math.Abs(after)), after)
}

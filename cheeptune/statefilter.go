package cheeptune

import "math"

// stateFilter is a state-variable filter in the trapezoidal form (the one
// Andrew Simper and Vadim Zavalishin describe): one step gives the low,
// band and high outputs together, and it stays stable with the cutoff
// anywhere under half the rate and moving every sample.
type stateFilter struct {
	g, k   float64 // the tuned coefficients
	a1, a2 float64
	s1, s2 float64 // the two integrators' states
}

// newStateFilter is a filter at rest, not yet tuned.
func newStateFilter() *stateFilter { return &stateFilter{} }

// filterCeiling is the highest cutoff, as a share of the rate. Picked:
// just under half, where the tangent is still finite.
const filterCeiling = 0.49

// filterSharpest is the damping left at resonance one. Picked: a little
// above nought, so the filter rings hard and does not sing on its own.
const filterSharpest = 0.05

// tune sets the cutoff in hertz and the resonance, nought (flat) to one.
func (f *stateFilter) tune(hertz, resonance float64) {
	hertz = math.Min(hertz, filterCeiling*studioRate)
	f.g = math.Tan(math.Pi * hertz / studioRate)
	f.k = 2 - (2-filterSharpest)*resonance
	f.a1 = 1 / (1 + f.g*(f.g+f.k))
	f.a2 = f.g * f.a1
}

// step takes one sample in and gives the low, band and high outputs.
func (f *stateFilter) step(x float64) (low, band, high float64) {
	v3 := x - f.s2
	v1 := f.a1*f.s1 + f.a2*v3
	v2 := f.s2 + f.g*v1
	f.s1 = 2*v1 - f.s1
	f.s2 = 2*v2 - f.s2
	return v2, v1, x - f.k*v1 - v2
}

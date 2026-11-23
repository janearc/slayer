package oceantex

import "math"

// Ringed is the variance of a periodic n by n grid of values, row by row,
// split by whole wave number: ring m holds the mean square of the waves
// that fit m times across the grid's side, in any direction, m from
// nought to under n/2. Waves in the grid's corners, past n/2, are in no
// ring. n is a power of two, as fft takes it.
//
// It is how a sampled sea is set beside its spectrum: the ring with the
// most in it is the sea's peak wave, and the rings summed are its height's
// variance.
func Ringed(values []float64, n int) []float64 {
	grid := make([]complex128, n*n)
	for index, value := range values {
		grid[index] = complex(value, 0)
	}
	fft2(grid, n, false)
	rings := make([]float64, n/2)
	cells := float64(n) * float64(n)
	for iy := range n {
		for ix := range n {
			ring := int(math.Round(math.Hypot(float64(wave(ix, n)),
				float64(wave(iy, n)))))
			if ring >= len(rings) {
				continue
			}
			at := grid[iy*n+ix]
			rings[ring] += (real(at)*real(at) + imag(at)*imag(at)) /
				(cells * cells)
		}
	}
	return rings
}

// PeakWavelength is the length in metres of the wind sea's peak wave by
// wave number: where the height's variance per unit wave number, all
// directions together, is greatest. That is the frequency spectrum
// carried across by the dispersion relation, S(omega) d(omega)/dk, so it
// peaks a little longer than the wave of the peak frequency.
//
// It is found by trying integrate.steps wave numbers, evenly in the
// logarithm, between the waves of integrate.low and integrate.high peak
// frequencies. A calm sea has no peak and gives nought.
func (s Sea) PeakWavelength() float64 {
	if s.Wind <= 0 || s.Fetch <= 0 {
		return 0
	}
	peak := s.Peak()
	low := math.Pow(sumLow*peak, 2) / gravity
	high := math.Pow(sumHigh*peak, 2) / gravity
	step := math.Log(high/low) / sumSteps
	var most, at float64
	for each := range int(sumSteps) {
		k := low * math.Exp((float64(each)+0.5)*step)
		omega, slope := s.turning(k)
		if density := s.frequency(omega) * slope; density > most {
			most, at = density, k
		}
	}
	return 2 * math.Pi / at
}

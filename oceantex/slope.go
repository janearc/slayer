package oceantex

import (
	"errors"
	"math"
)

// The sum that gives the slope a grid resolves: its steps in direction
// over a full turn and in the logarithm of wave number.
var (
	slopeTurns = num("slope.integrate.turns")
	slopeSteps = num("slope.integrate.steps")

	// totalOffset and totalSlope are Cox and Munk's fit to the total
	// mean-square slope of a clean surface, apart from its two parts.
	totalOffset = num("cox-munk.total.offset")
	totalSlope  = num("cox-munk.total.slope")
)

// SlopeBudget is a sea's mean-square slope, both ways across the surface
// together, split at a square grid's spacing into the part the grid
// carries in its heights and the part below one cell, which a glint
// lobe must carry instead.
//
// Total is Cox and Munk's, CoxMunkSlope. Resolved is the spectrum's up
// to the grid's Nyquist wave number, ResolvedSlope. Unresolved is Total
// less Resolved, or the floor where that is less than the floor.
//
// Floored says Unresolved is the floor and not the difference; Exceeds
// says the spectrum resolves at least Cox and Munk's whole slope, so the
// two do not agree on this sea and the floor stands in for nothing.
type SlopeBudget struct {
	Total      float64 `json:"total"`
	Resolved   float64 `json:"resolved"`
	Unresolved float64 `json:"unresolved"`
	// Nyquist is the grid's Nyquist wave number, pi over its spacing,
	// radians a metre.
	Nyquist float64 `json:"nyquist_radians_per_metre"`
	Floored bool    `json:"floored"`
	Exceeds bool    `json:"exceeds"`
}

// CoxMunkSlope is the mean-square slope of a clean sea surface, along
// the wind and across it together, for a wind in metres a second:
// Cox and Munk's fit to the total, 0.003 + 0.00512 U, 0.03884 at 7 m/s.
// A calm or backward wind leaves the offset.
//
// It is not Roughness's two variances summed: those are fit apart and
// sum to 0.003 + 0.00508 U, 0.03856 at 7 m/s. Within the fits' own
// 0.004 either way, the two agree.
//
// Cox, C. and Munk, W. (1954), J. Opt. Soc. Am. 44, 838-850. It is not
// on the shelf that SLAYER_SHELF names. The numbers are as
// sources.yaml records them from it, and as it records them seen again
// in Ross and Dion (2004), SPIE 5572, an open copy at sleet.larc.nasa.gov.
//
// Cox and Munk measured U at 12.5 m; a Sea's Wind is at 10 m, and is
// taken as their U unchanged. A neutral log profile over a roughness
// length of 0.1 mm (assumed) puts U at 12.5 m about 2% above U at 10 m.
func CoxMunkSlope(wind float64) float64 {
	return totalOffset + totalSlope*math.Max(wind, 0)
}

// ResolvedSlope is the mean-square slope of the waves a square grid of
// this spacing in metres can carry: the integral of k squared times the
// spectrum, Wavenumber, over the wave vectors whose components both lie
// within the grid's Nyquist wave number, pi over the spacing.
//
// It is summed in polar form, even steps in direction and in the
// logarithm of wave number, from integrate.low peak frequencies, read as
// deep water, out to the edge of the grid's square. A calm sea, or one
// with no fetch, resolves nothing.
//
// Its spectrum falls as k to the fourth beyond the peak, so the slope
// it holds grows as the log of the grid's Nyquist wave number without
// bound: a fine enough grid resolves more than Cox and Munk measured.
func (s Sea) ResolvedSlope(spacing float64) float64 {
	if s.Wind <= 0 || s.Fetch <= 0 {
		return 0
	}
	edge := math.Pi / spacing
	low := math.Pow(sumLow*s.Peak(), 2) / Gravity
	turn := 2 * math.Pi / slopeTurns
	var sum float64
	for step := 0; step < int(slopeTurns); step++ {
		theta := (float64(step) + 0.5) * turn
		sin, cos := math.Sincos(theta)
		reach := edge / math.Max(math.Abs(cos), math.Abs(sin))
		sum += s.slopeAlong(sin, cos, low, reach)
	}
	return sum * turn
}

// slopeAlong is the integral, along one direction with this sine and
// cosine, of k squared times the spectrum times k, the polar area's,
// over wave numbers from low to high: even steps in log k, where dk is
// k d(log k).
func (s Sea) slopeAlong(sin, cos, low, high float64) float64 {
	if !(high > low) {
		return 0
	}
	step := math.Log(high/low) / slopeSteps
	var sum float64
	for each := 0; each < int(slopeSteps); each++ {
		k := low * math.Exp((float64(each)+0.5)*step)
		sum += k * k * k * k * s.Wavenumber(k*cos, k*sin)
	}
	return sum * step
}

// SlopeBudget splits the sea's mean-square slope at a square grid's
// spacing in metres: Cox and Munk's total, the spectrum's resolved part,
// and the unresolved rest, floored at slope.floor. It refuses a spacing
// that is not finite and above nought, and a slope that is not finite.
func (s Sea) SlopeBudget(spacing float64) (SlopeBudget, error) {
	if !finite(spacing) || !(spacing > 0) {
		return SlopeBudget{}, errors.New(
			"oceantex: a grid spacing is finite and above nought")
	}
	total, resolved := CoxMunkSlope(s.Wind), s.ResolvedSlope(spacing)
	if !finite(total) || !finite(resolved) {
		return SlopeBudget{}, errors.New(
			"oceantex: the sea's slope is not finite")
	}
	budget := SlopeBudget{Total: total, Resolved: resolved,
		Unresolved: total - resolved, Nyquist: math.Pi / spacing,
		Exceeds: resolved >= total}
	if budget.Unresolved < slopeFloor {
		budget.Unresolved, budget.Floored = slopeFloor, true
	}
	return budget, nil
}

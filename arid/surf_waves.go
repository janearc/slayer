// This file is what linear wave theory and the surf's measurements say of
// one wave train as the water shoals: its length, its speed, its height,
// whether it breaks and as what, and how far up the beach it runs.

package arid

import "math"

// waveNumber is the wave number, in radians a metre, of a wave of an
// angular frequency in water of a depth in metres: the dispersion
// relation, omega^2 = g k tanh(k h) [Airy], solved by Newton's method. A
// depth of nought or less is deep water, where omega^2 = g k.
func waveNumber(omega, depth float64) float64 {
	deep := omega * omega / gravity
	if !(depth > 0) {
		return deep
	}
	number := deep / math.Sqrt(math.Tanh(deep*depth))
	for range dispersionRounds {
		tanh := math.Tanh(number * depth)
		sech := 1 / math.Cosh(number*depth)
		miss := gravity*number*tanh - omega*omega
		rate := gravity * (tanh + number*depth*sech*sech)
		next := number - miss/rate
		if math.Abs(next-number) <= dispersionSettled*number {
			return next
		}
		number = next
	}
	return number
}

// frequency is the angular frequency, in radians a second, of a wave of a
// wave number in water of a depth: the dispersion relation [Airy] read the
// other way. A depth of nought or less is deep water.
func frequency(number, depth float64) float64 {
	if !(depth > 0) {
		return math.Sqrt(gravity * number)
	}
	return math.Sqrt(gravity * number * math.Tanh(number*depth))
}

// groupSpeed is the speed, in metres a second, at which a wave of an
// angular frequency carries its energy in water of a depth: c_g = c / 2
// (1 + 2 k h / sinh 2 k h) [CW-SW eq. 3.23], half the phase speed in deep
// water and all of it in shallow.
func groupSpeed(omega, depth float64) float64 {
	number := waveNumber(omega, depth)
	phase := omega / number
	if !(depth > 0) {
		return phase / twice
	}
	doubled := twice * number * depth
	return phase / twice * (1 + doubled/math.Sinh(doubled))
}

// shoaling is how much taller a wave of an angular frequency stands in
// water of a depth than in the open water's, open, when no energy is lost
// on the way: K_S = (c_g0 / c_g)^(1/2) [CW-SW eq. 4.6]. It falls a little
// under one in water of middling depth and grows without end at the shore.
func shoaling(omega, depth, open float64) float64 {
	return math.Sqrt(groupSpeed(omega, open) / groupSpeed(omega, depth))
}

// breakingFraction is the share of the waves at a place that are breaking,
// from the ratio of their root-mean-square height to the greatest the
// depth carries, Hmax = gamma d: Battjes and Janssen's (1 - Q) / ln Q =
// -(Hrms / Hmax)^2 [SWAN eq. 2.66, 8 E = Hrms^2], solved for Q.
//
// Q = 1 is always a root. For a ratio under one the other lies between
// nought and the ratio squared, where the left side less the right is
// greatest, and halving finds it. From a ratio of one, all are breaking.
func breakingFraction(ratio float64) float64 {
	switch {
	case !(ratio > 0):
		return 0
	case ratio >= 1:
		return 1
	}
	squared := ratio * ratio
	low, high := 0.0, squared
	for range breakingRounds {
		middle := (low + high) / twice
		if 1-middle+squared*math.Log(middle) < 0 {
			low = middle
		} else {
			high = middle
		}
	}
	return (low + high) / twice
}

// breaker is a kind of breaking wave, by the surf similarity number.
type breaker int

// The kinds of breaker, by Battjes's surf similarity number offshore
// [IRI]: spilling on a gentle beach, plunging on a steeper one, surging
// or collapsing on the steepest.
const (
	spilling breaker = iota
	plunging
	surging
)

// breakerKind is the kind of breaker a beach face of a slope, rise over
// run, makes of waves of a significant height and a deep-water length:
// the surf similarity number xi0 = tan(alpha) / sqrt(H0 / L0), Battjes
// 1974 [IRI], against his bounds.
func breakerKind(slope, height, length float64) breaker {
	similar := slope / math.Sqrt(height/length)
	switch {
	case similar < spillingBelow:
		return spilling
	case similar > surgingAbove:
		return surging
	}
	return plunging
}

// runup is the mean and the standard deviation of the highest the swash
// reaches, in metres over the sea's mean level, on a beach face of a
// slope, for waves of a significant height and a deep-water length.
//
// Stockdon's R2 [CW-RU], the reach two run-ups in a hundred pass, is
// 1.1 (setup + S / 2). Taken as a mean, 1.1 setup, and a deviation of 1.1
// S / 4, the two-in-a-hundred reach is two deviations over the mean. On a
// beach under 0.3, R2 is 0.043 sqrt(H L) and is all swash about nought.
func runup(slope, height, length float64) (mean, deviation float64) {
	scale := math.Sqrt(height * length)
	similar := slope / math.Sqrt(height/length)
	if similar < dissipativeBelow {
		return 0, dissipativeRunup * scale / exceedanceDeviations
	}
	setup := setupScale * slope * scale
	swash := math.Hypot(swashIncident*slope*scale, swashInfragravity*scale)
	return runupExceedance * setup,
		runupExceedance * swash / swashDeviations
}

// whitecapCover is the share of the open sea that whitecaps cover in a
// wind of a speed at 10 m, in metres a second: W = 3.84e-6 U^3.41,
// Monahan and O'Muircheartaigh 1980, held to at most all of it.
func whitecapCover(wind float64) float64 {
	if !(wind > 0) {
		return 0
	}
	return math.Min(whitecapScale*math.Pow(wind, whitecapPower), 1)
}

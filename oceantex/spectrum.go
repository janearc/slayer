package oceantex

import "math"

// Every number below is read from sources.yaml by the name in quotes.
var (
	// gravity is standard gravity, metres a second squared.
	gravity = num("gravity.standard")

	alphaScale  = num("jonswap.alpha.scale")
	alphaPower  = num("jonswap.alpha.power")
	peakScale   = num("jonswap.peak.scale")
	peakPower   = num("jonswap.peak.power")
	peakGamma   = num("jonswap.gamma")
	widthBelow  = num("jonswap.width.below")
	widthAbove  = num("jonswap.width.above")
	peakDecay   = num("jonswap.decay")
	tailPower   = num("jonswap.tail")
	rolloff     = num("jonswap.rolloff")
	fullPeak    = num("pierson-moskowitz.peak")
	lowScale    = num("spread.low.scale")
	lowPower    = num("spread.low.power")
	lowTop      = num("spread.low.top")
	midScale    = num("spread.mid.scale")
	midPower    = num("spread.mid.power")
	midTop      = num("spread.mid.top")
	highOffset  = num("spread.high.offset")
	highScale   = num("spread.high.scale")
	highRate    = num("spread.high.rate")
	tmaKnee     = num("depth.tma.knee")
	tmaTop      = num("depth.tma.top")
	tmaHalf     = num("depth.tma.half")
	heightTimes = num("waveheight.factor")
	sumSteps    = num("integrate.steps")
	sumLow      = num("integrate.low")
	sumHigh     = num("integrate.high")
)

// Sea is the wind that raised the waves, and the water under them.
type Sea struct {
	// Wind is the wind speed ten metres above the water, in metres a
	// second.
	Wind float64
	// Fetch is the distance the wind has blown over open water, in metres.
	// Near a coast with the wind blowing off the land, it is the distance
	// to that land.
	Fetch float64
	// Toward is the direction the wind blows toward, in radians from east,
	// counterclockwise: 0 is toward the east, pi/2 toward the north.
	Toward float64
	// Depth is the depth of the water in metres, the tide included. Zero
	// is deep water. The depth is the same everywhere in a sea: there is
	// no sloping beach.
	Depth float64
	// Swell is a train of long waves from a distant wind, added to the
	// wind sea, or nil for none. A sea with no swell is the wind sea
	// alone, exactly.
	Swell *Swell
}

// grown is the fetch that counts: the sea's own, or the fetch at which
// this wind has raised all the sea it can, if that is shorter.
//
// GLUE: [H73]'s formulas describe a sea still growing and never stop. A
// real one stops when it is fully developed, which [PM64] found is when
// its peak has fallen to 0.877 g / U. grown is the fetch at which the
// JONSWAP peak reaches that.
func (s Sea) grown() float64 {
	full := math.Pow(peakScale/fullPeak, 1/peakPower) *
		s.Wind * s.Wind / gravity
	return math.Min(s.Fetch, full)
}

// Peak is the angular frequency, radians a second, where the spectrum
// peaks in the prototype's rounded fit: 22 (g^2 / (U F))^(1/3), with F
// the fetch that counts. [H73] Eq. 2.4.3 instead prints a dimensionless
// exponent -0.33. This is an approximation, not its exact transcription;
// see doc/phone-source-checks.md for the retained original and conventions.
func (s Sea) Peak() float64 {
	return peakScale * math.Pow(gravity*gravity/(s.Wind*s.grown()), peakPower)
}

// shallow is how much of a deep-water spectrum survives at a frequency in
// water of this sea's depth: the depth factor of the TMA spectrum, in the
// two-parabola form [Ho15] gives. In deep water it is 1.
func (s Sea) shallow(omega float64) float64 {
	if s.Depth <= 0 {
		return 1
	}
	w := omega * math.Sqrt(s.Depth/gravity)
	switch {
	case w <= tmaKnee:
		return tmaHalf * w * w
	case w < tmaTop:
		return 1 - tmaHalf*(tmaTop-w)*(tmaTop-w)
	}
	return 1
}

// frequency is the variance of surface height per unit angular frequency,
// in square metres seconds a radian: the JONSWAP spectrum [H73], times
// the TMA depth factor where the water is shallow.
func (s Sea) frequency(omega float64) float64 {
	if omega <= 0 || s.Wind <= 0 || s.Fetch <= 0 {
		return 0
	}
	peak := s.Peak()
	alpha := alphaScale * math.Pow(s.Wind*s.Wind/(s.grown()*gravity), alphaPower)
	width := widthBelow
	if omega > peak {
		width = widthAbove
	}
	r := math.Exp(-(omega - peak) * (omega - peak) /
		(2 * width * width * peak * peak))
	return alpha * gravity * gravity / math.Pow(omega, tailPower) *
		math.Exp(-peakDecay*math.Pow(peak/omega, rolloff)) *
		math.Pow(peakGamma, r) * s.shallow(omega)
}

// Height is the significant wave height in metres: four times the square
// root of the spectrum's sum, as wave buoys report it. The swell's
// variance, where the sea has one, is in the sum.
func (s Sea) Height() float64 {
	var sum float64
	if !(s.Wind <= 0 || s.Fetch <= 0) {
		peak := s.Peak()
		sum = logSum(sumLow*peak, sumHigh*peak, sumSteps, s.frequency)
	}
	if s.Swell != nil {
		sum += s.swellVariance()
	}
	return heightTimes * math.Sqrt(sum)
}

// spread is how the energy at one frequency is shared among directions:
// the share per radian at angle theta from the wind. It is the
// sech-squared form of [D85], extended above 1.6 peak frequencies by
// Banner, as [Ho15] gives it. Over a full turn it sums to one.
func spread(omega, peak, theta float64) float64 {
	ratio := omega / peak
	var beta float64
	switch {
	case ratio < lowTop:
		beta = lowScale * math.Pow(ratio, lowPower)
	case ratio < midTop:
		beta = midScale * math.Pow(ratio, midPower)
	default:
		beta = math.Pow(10,
			highOffset+highScale*math.Exp(highRate*math.Log(ratio*ratio)))
	}
	sech := 1 / math.Cosh(beta*theta)
	return beta / (2 * math.Tanh(beta*math.Pi)) * sech * sech
}

// turning is a wave's angular frequency and how fast that changes with
// wave number, for a wave number k: the dispersion relation
// omega^2 = g k tanh(k h), which is omega^2 = g k where the water is deep.
func (s Sea) turning(k float64) (omega, slope float64) {
	if s.Depth <= 0 || k*s.Depth > 20 {
		omega = math.Sqrt(gravity * k)
		return omega, gravity / (2 * omega)
	}
	kh := k * s.Depth
	tanh := math.Tanh(kh)
	sech := 1 / math.Cosh(kh)
	omega = math.Sqrt(gravity * k * tanh)
	return omega, gravity * (tanh + kh*sech*sech) / (2 * omega)
}

// wavenumber is the spectrum by wave vector: the variance of surface
// height per unit area of wavenumber space, in metres to the fourth, for
// waves travelling along (kx, ky) radians a metre. It is the wind sea's
// and, where the sea has one, the swell's.
func (s Sea) wavenumber(kx, ky float64) float64 {
	wind := s.windWavenumber(kx, ky)
	if s.Swell == nil {
		return wind
	}
	return wind + s.swellWavenumber(kx, ky)
}

// windWavenumber is the wind sea's spectrum by wave vector.
//
// It is frequency times spread, carried across by the dispersion relation
// and divided by k for the polar area element [Ho15, eq. 61].
func (s Sea) windWavenumber(kx, ky float64) float64 {
	k := math.Hypot(kx, ky)
	if k == 0 || s.Wind <= 0 || s.Fetch <= 0 {
		return 0
	}
	omega, slope := s.turning(k)
	theta := math.Remainder(math.Atan2(ky, kx)-s.Toward, 2*math.Pi)
	return s.frequency(omega) * spread(omega, s.Peak(), theta) * slope / k
}

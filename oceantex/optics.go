package oceantex

import (
	"math"

	"github.com/janearc/slayer/maths"
)

var (
	quanFry       = nums("quan-fry.coefficients")
	indexAt       = num("index.wavelength")
	alongSlope    = num("cox-munk.along.slope")
	acrossOffset  = num("cox-munk.across.offset")
	acrossSlope   = num("cox-munk.across.slope")
	glintGeometry = num("glint.geometry")
	glintTilt     = num("glint.tilt.power")
	slopeFloor    = num("slope.floor")
	viewFloor     = num("view.floor")
	foamScale     = num("monahan.scale")
	foamPower     = num("monahan.power")

	// foamReflectance is the effective reflectance of whitecaps in
	// visible light [K84], lower than fresh foam because a whitecap thins
	// as it ages.
	foamReflectance = num("koepke.foam.reflectance")
)

// Index is the refractive index of seawater against air, by the empirical
// equation of [QF95]: salinity in parts per thousand, temperature in
// degrees Celsius, wavelength in nanometres. It is a fit to measurements
// in visible light, 400 to 700 nm, at salinities up to 35.
func Index(salinity, celsius, nm float64) float64 {
	n := quanFry
	t2 := celsius * celsius
	return n[0] + (n[1]+n[2]*celsius+n[3]*t2)*salinity + n[4]*t2 +
		(n[5]+n[6]*salinity+n[7]*celsius)/nm + n[8]/(nm*nm) + n[9]/(nm*nm*nm)
}

// Fresnel is the fraction of unpolarised light a flat air-water surface
// reflects, given the cosine of the angle between the light and the
// surface's normal, and the water's refractive index. Fresnel's equations
// with Snell's law, as maths.Fresnel has them.
func Fresnel(cosine, index float64) float64 {
	return maths.Fresnel(cosine, index)
}

// roughness is the variance of the sea's slope along the wind and across
// it, as [CM54] measured from photographs of sun glitter over a clean
// surface. They measured the wind at 12.5 metres.
func roughness(wind float64) (along, across float64) {
	wind = math.Max(wind, 0)
	return alongSlope * wind, acrossOffset + acrossSlope*wind
}

// SlopeChance is the chance of a slope: a Gaussian, the first-order
// form of [CM54], per unit slope along and across the wind.
func SlopeChance(along, across, varAlong, varAcross float64) float64 {
	return math.Exp(-(along*along/varAlong+across*across/varAcross)/2) /
		(2 * math.Pi * math.Sqrt(varAlong*varAcross))
}

// Hidden is Smith's function: how much of a rough surface is hidden from
// a direction, from the cosine of the direction's angle to the vertical
// and the slope variance along it. Lambda = (exp(-v^2)/(v sqrt(pi)) -
// erfc(v))/2 with v = cot(angle)/(sqrt(2) sigma), as given for the sea in
// Ross & Dion (SPIE 5572, 2004) after Smith (1967). The share of the
// surface in view is 1/(1 + Lambda).
func Hidden(cosine, variance float64) float64 {
	if cosine <= 0 {
		return math.Inf(1)
	}
	if cosine >= 1 || variance <= 0 {
		return 0
	}
	cot := cosine / math.Sqrt(1-cosine*cosine)
	v := cot / math.Sqrt(2*variance)
	return (math.Exp(-v*v)/(v*math.Sqrt(math.Pi)) - math.Erfc(v)) / 2
}

// whitecaps is the fraction of the sea covered by whitecaps at a wind
// speed ten metres up, by the robust fit of [MO80].
func whitecaps(wind float64) float64 {
	if wind <= 0 {
		return 0
	}
	return math.Min(1, foamScale*math.Pow(wind, foamPower))
}

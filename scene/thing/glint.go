package thing

import "math"

// Microfacet is how much of a lamp's light a rough surface sends toward
// the eye as a highlight, before Fresnel's share: the reflection
// function times the cosine to the lamp, so that times the light a lamp
// sends face on is the highlight's radiance.
//
// It is the GGX distribution of the surface's tiny faces times Smith's
// share of them neither shadowed from the lamp nor hidden from the eye,
// over four times the cosine to the eye, the cosine to the lamp having
// cancelled.
//
// facing is the cosine between the normal and the way half between the
// lamp and the eye; lit and viewed are the cosines to the lamp and to
// the eye; alpha is the GGX width, the square of the surface's Satin.
//
// Walter and others, "Microfacet models for refraction through rough
// surfaces" (EGSR 2007), equations 33 and 34, with Smith's separable
// shadowing.
func Microfacet(facing, lit, viewed, alpha float64) float64 {
	if !(facing > 0 && lit > 0 && viewed > 0 && alpha > 0) {
		return 0
	}
	a2 := alpha * alpha
	spread := facing*facing*(a2-1) + 1
	faces := a2 / (math.Pi * spread * spread)
	return faces * smith(lit, a2) * smith(viewed, a2) / (4 * viewed)
}

// smith is the share of a rough surface's tiny faces a way of cosine
// cos to the normal sees unshadowed, for a GGX width squared a2.
func smith(cos, a2 float64) float64 {
	return 2 * cos / (cos + math.Sqrt(a2+(1-a2)*cos*cos))
}

// Beckmann is Microfacet's highlight for a surface whose tiny faces
// have Gaussian slopes, as the sea's have to first order: Beckmann's
// distribution of faces times Smith's share neither shadowed nor
// hidden, over four times the cosine to the eye.
//
// facing, lit and viewed are as Microfacet's. slopes is the faces'
// mean-square slope, both ways together. Beckmann's slopes are Gaussian,
// variance slopes/2 each way, so slopes is Cox and Munk's sigma squared,
// matched exactly (see oceantex.CoxMunkSlope).
//
// No Satin matches it: GGX's slopes have tails too heavy for a
// variance. At equal peak density (alpha = sqrt(slopes), Satin = slopes
// to the quarter) GGX is 0.68 of Beckmann where tan^2 is slopes, 2.2
// times it at twice that tangent and 81 times at three times.
//
// Walter and others, "Microfacet models for refraction through rough
// surfaces" (EGSR 2007): Beckmann's distribution, Smith's shadowing.
func Beckmann(facing, lit, viewed, slopes float64) float64 {
	if !(facing > 0 && lit > 0 && viewed > 0 && slopes > 0) {
		return 0
	}
	return beckmannFaces(facing, slopes) * beckmannSmith(lit, slopes) *
		beckmannSmith(viewed, slopes) / (4 * viewed)
}

// beckmannFaces is the Beckmann density of tiny faces whose normal has
// cosine facing to the surface's, for mean-square slope slopes:
// exp(-tan^2 / slopes) / (pi slopes cos^4). Times facing, over the
// hemisphere, it sums to one.
func beckmannFaces(facing, slopes float64) float64 {
	square := facing * facing
	tangent := (1 - square) / square
	return math.Exp(-tangent/slopes) /
		(math.Pi * slopes * square * square)
}

// beckmannSmith is the share of a Beckmann surface's tiny faces a way of
// cosine cos sees unshadowed: 1 / (1 + Lambda), Lambda = (exp(-a^2) /
// (a sqrt(pi)) - erfc(a)) / 2, a = cot / sqrt(slopes), as
// oceantex.Hidden has it for the sea. Straight down, all of them.
func beckmannSmith(cos, slopes float64) float64 {
	if cos >= 1 {
		return 1
	}
	a := cos / math.Sqrt((1-cos*cos)*slopes)
	hidden := (math.Exp(-a*a)/(a*math.SqrtPi) - math.Erfc(a)) / 2
	return 1 / (1 + hidden)
}

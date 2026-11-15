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

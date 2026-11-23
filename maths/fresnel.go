package maths

import "math"

// polarisations is how many polarisations unpolarised light is the mean of,
// defined: the light that vibrates in the plane of incidence and the light
// that vibrates across it.
const polarisations = 2

// Fresnel is the fraction of unpolarised light a flat boundary reflects,
// given the cosine of the angle between the light and the boundary's
// normal, and the index of the medium the light is going into over the
// index of the one it comes from: Fresnel's equations with Snell's law.
//
// An index above one is light going into the denser medium. An index
// below one is light going into the less dense one, and past the
// critical angle, where nothing can be transmitted, Fresnel returns
// exactly one. The cosine is clamped to between zero and one.
//
// An index of zero or less, an index that is no number, and an infinite
// index are no medium, and the share is no number.
func Fresnel(cosine, index float64) float64 {
	cosine = math.Min(unity, math.Max(nought, cosine))
	if !(index > nought) || math.IsInf(index, unity) {
		return math.NaN()
	}
	sine2 := (unity - cosine*cosine) / (index * index)
	if sine2 >= unity {
		return unity
	}
	// the cosine of the refracted angle
	inside := math.Sqrt(unity - sine2)
	across := (cosine - index*inside) / (cosine + index*inside)
	along := (index*cosine - inside) / (index*cosine + inside)
	return (across*across + along*along) / polarisations
}

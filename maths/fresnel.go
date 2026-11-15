package maths

import "math"

// Fresnel is the fraction of unpolarised light a flat boundary reflects,
// given the cosine of the angle between the light and the boundary's
// normal, and the index of the medium the light is going into over the
// index of the one it comes from: Fresnel's equations with Snell's law.
//
// An index above one is light going into the denser medium. An index
// below one is light going into the less dense one, and past the
// critical angle, where nothing can be transmitted, Fresnel returns
// exactly one. The cosine is clamped to between zero and one.
func Fresnel(cosine, index float64) float64 {
	cosine = math.Min(1, math.Max(0, cosine))
	sine2 := (1 - cosine*cosine) / (index * index)
	if sine2 >= 1 {
		return 1
	}
	inside := math.Sqrt(1 - sine2) // cosine of the refracted angle
	across := (cosine - index*inside) / (cosine + index*inside)
	along := (index*cosine - inside) / (index*cosine + inside)
	return (across*across + along*along) / 2
}

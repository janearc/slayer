package thing

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The counts and places a metal's reading is made of.
const (
	// metalNone is defined: no amount.
	metalNone = 0
	// metalWhole is defined: one amount, the whole of a share.
	metalWhole = 1
	// metalPair is defined: two, the parts of a pair, and what halves.
	metalPair = 2
	// metalBands is defined: the three bands of red, green and blue.
	metalBands = 3
	// metalFour is defined: the factor of four in Fresnel's equations for
	// a conductor.
	metalFour = 4
	// metalRed is defined: the first place in a reading, red's.
	metalRed = 0
	// metalGreen is defined: the second place in a reading, green's.
	metalGreen = 1
	// metalBlue is defined: the third place in a reading, blue's.
	metalBlue = 2
)

// Metal is a conductor's complex refractive index, n + ik, at the three
// wavelengths the tracer reads light at: red, green and blue, about 610,
// 550 and 465 nm. Its reflectance follows from them by Fresnel's
// equations for a conductor, and differs by colour and by angle, which
// is what makes gold gold: it mirrors most of red and under half of blue
// face on, and all of every colour at grazing.
//
// A Metal with no index at all, n and k zero in a colour, reflects all of
// that colour at every look: see Conductor.
type Metal struct {
	N, K [metalBands]float64
}

// Film is a clear layer over a metal, as Kapton is over the aluminium of
// a spacecraft's blankets, silica over the gold of a telescope's mirror,
// or glass over the silver of a radiator's tiles: its refractive index,
// its thickness in metres, and how much of each of red, green and blue
// it absorbs, per metre.
//
// Its front reflects by Fresnel's equations for a dielectric; what
// passes is absorbed going in and coming out, the metal below reflects
// what reaches it, and the light that bounces between the two is
// summed. The layers' interference is not drawn: the sum is of
// intensities, right for a layer many wavelengths thick and only near
// for one a fraction of a wavelength thin.
//
// An Index at or below zero is read as glass's 1.5, as Material.Index is:
// see Material.Dense.
type Film struct {
	Index, Thickness float64
	Absorbs          [metalBands]float64
}

// dense is the refractive index the film reflects and bends by: Index, or
// glass's 1.5 where none is given or it is at or below zero.
func (film *Film) dense() float64 {
	if film.Index <= metalNone {
		return glassIndex
	}
	return film.Index
}

// Reflects is how much of each of red, green and blue the surface
// mirrors at a look whose cosine to its normal is cos: a metal's by its
// complex index under its film, when it has one, or otherwise the same
// share of all three by Schlick's approximation, as mirrors gives.
// Either is scaled by Gloss; with no gloss nothing is mirrored.
func (material *Material) Reflects(cos float64) maths.Light {
	if material.Metal == nil {
		return maths.Grey(material.Mirrors(cos))
	}
	if material.Gloss <= metalNone {
		return maths.Grey(metalNone)
	}
	gloss := math.Min(material.Gloss, metalWhole)
	cos = math.Max(metalNone, math.Min(metalWhole, cos))
	var share [metalBands]float64
	for band := range share {
		share[band] = gloss * material.metalReflects(cos, band)
	}
	return maths.RGB(share[metalRed], share[metalGreen], share[metalBlue])
}

// metalReflects is the share of colour band the metal, under its film when
// it has one, mirrors at a look of cosine cos.
func (material *Material) metalReflects(cos float64, band int) float64 {
	refractive, k := material.Metal.N[band], material.Metal.K[band]
	film := material.Film
	if film == nil {
		return Conductor(cos, refractive, k)
	}
	within := film.dense()
	sinIn := metalWhole - cos*cos
	inside := math.Sqrt(math.Max(metalNone,
		metalWhole-sinIn/(within*within)))
	front := dielectric(cos, within)
	passes := math.Exp(-film.Absorbs[band] * film.Thickness / inside)
	twice := passes * passes
	below := Conductor(inside, refractive/within, k/within)
	bounce := front * below * twice
	if bounce >= metalWhole {
		// Nothing leaves the film but what it reflects: the sum of the
		// bounces would divide nothing by nothing, and is all of it.
		return metalWhole
	}
	return front + (metalWhole-front)*(metalWhole-front)*below*twice/
		(metalWhole-bounce)
}

// Conductor is the share of unpolarised light a Conductor of complex
// index n + ik, relative to what the light arrives through, reflects at
// a look of cosine cos: Fresnel's equations, the s and p shares
// averaged. Face on it is ((n-1)^2 + k^2) / ((n+1)^2 + k^2); at grazing
// it is all.
//
// A Conductor of no index at all, n and k both zero, is what the equations
// tend to as both fall to nothing: it reflects all, at every look. Face on
// they divide nothing by nothing there, so that value is given instead.
func Conductor(cos, refractive, k float64) float64 {
	if refractive == metalNone && k == metalNone {
		return metalWhole
	}
	cos2 := cos * cos
	sin2 := metalWhole - cos2
	n2, k2 := refractive*refractive, k*k
	t0 := n2 - k2 - sin2
	ab := math.Sqrt(t0*t0 + metalFour*n2*k2)
	t1 := ab + cos2
	reach := math.Sqrt(math.Max(metalNone, (ab+t0)/metalPair))
	t2 := metalPair * cos * reach
	perpendicular := (t1 - t2) / (t1 + t2)
	t3 := cos2*ab + sin2*sin2
	t4 := t2 * sin2
	parallel := perpendicular * (t3 - t4) / (t3 + t4)
	return (perpendicular + parallel) / metalPair
}

// dielectric is the share of unpolarised light a clear surface of
// refractive index n, relative to what the light arrives through,
// reflects at a look of cosine cos: Fresnel's equations, exactly, where
// mirrors takes Schlick's approximation of them. Past the critical
// angle, from inside a denser medium, it is all.
func dielectric(cos, refractive float64) float64 {
	return maths.Fresnel(cos, refractive)
}

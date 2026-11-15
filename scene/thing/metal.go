package thing

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// Metal is a conductor's complex refractive index, n + ik, at the three
// wavelengths the tracer reads light at: red, green and blue, about 610,
// 550 and 465 nm. Its reflectance follows from them by Fresnel's
// equations for a conductor, and differs by colour and by angle, which
// is what makes gold gold: it mirrors most of red and under half of blue
// face on, and all of every colour at grazing.
type Metal struct {
	N, K [3]float64
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
type Film struct {
	Index, Thickness float64
	Absorbs          [3]float64
}

// Reflects is how much of each of red, green and blue the surface
// mirrors at a look whose cosine to its normal is cos: a metal's by its
// complex index under its film, when it has one, or otherwise the same
// share of all three by Schlick's approximation, as Mirrors gives.
// Either is scaled by Gloss; with no gloss nothing is mirrored.
func (m *Material) Reflects(cos float64) maths.Light {
	if m.Metal == nil {
		return maths.Grey(m.Mirrors(cos))
	}
	if m.Gloss <= 0 {
		return maths.Light{}
	}
	gloss := math.Min(m.Gloss, 1)
	cos = math.Max(0, math.Min(1, cos))
	var share [3]float64
	for c := range share {
		share[c] = gloss * m.metalReflects(cos, c)
	}
	return maths.RGB(share[0], share[1], share[2])
}

// metalReflects is the share of colour c the metal, under its film when
// it has one, mirrors at a look of cosine cos.
func (m *Material) metalReflects(cos float64, c int) float64 {
	n, k := m.Metal.N[c], m.Metal.K[c]
	film := m.Film
	if film == nil {
		return Conductor(cos, n, k)
	}
	within := film.Index
	sinIn := 1 - cos*cos
	inside := math.Sqrt(math.Max(0, 1-sinIn/(within*within)))
	front := Dielectric(cos, within)
	passes := math.Exp(-film.Absorbs[c] * film.Thickness / inside)
	twice := passes * passes
	below := Conductor(inside, n/within, k/within)
	return front + (1-front)*(1-front)*below*twice/(1-front*below*twice)
}

// Conductor is the share of unpolarised light a conductor of complex
// index n + ik, relative to what the light arrives through, reflects at
// a look of cosine cos: Fresnel's equations, the s and p shares
// averaged. Face on it is ((n-1)^2 + k^2) / ((n+1)^2 + k^2); at grazing
// it is all.
func Conductor(cos, n, k float64) float64 {
	cos2 := cos * cos
	sin2 := 1 - cos2
	n2, k2 := n*n, k*k
	t0 := n2 - k2 - sin2
	ab := math.Sqrt(t0*t0 + 4*n2*k2)
	t1 := ab + cos2
	a := math.Sqrt(math.Max(0, (ab+t0)/2))
	t2 := 2 * cos * a
	s := (t1 - t2) / (t1 + t2)
	t3 := cos2*ab + sin2*sin2
	t4 := t2 * sin2
	p := s * (t3 - t4) / (t3 + t4)
	return (s + p) / 2
}

// Dielectric is the share of unpolarised light a clear surface of
// refractive index n, relative to what the light arrives through,
// reflects at a look of cosine cos: Fresnel's equations, exactly, where
// Mirrors takes Schlick's approximation of them. Past the critical
// angle, from inside a denser medium, it is all.
func Dielectric(cos, n float64) float64 {
	return maths.Fresnel(cos, n)
}

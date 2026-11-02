// Package planet computes reflected starlight from layered atmospheres. Each
// layer describes gas, haze, and cloud through its composition and optical
// properties rather than a painted colour.
//
// The transfer model solves single scattering through homogeneous planar layers
// above a Lambertian deck. Gas coefficients come from refractive indices and
// absorption data; haze uses droplet scattering and absorption. Multiple
// scattering is omitted, so thick white haze renders too dark.
package planet

import "math"

// optics is a layer of air at one wavelength, by optical depth: how
// much its gas scatters and absorbs, and how much its haze turns light
// aside, with the share of that the haze scatters rather than absorbs
// and how far forward it leans.
type optics struct {
	scatter, absorb  float64
	haze, hazeAlbedo float64
	hazeLean         float64
}

// depth is the layer's whole optical depth.
func (layer optics) depth() float64 {
	return layer.scatter + layer.absorb + layer.haze
}

// scattered is the layer's scattering, gas and haze, by optical depth.
func (layer optics) scattered() float64 {
	return layer.scatter + layer.haze*layer.hazeAlbedo
}

// phase is how the layer throws light turned through an angle whose
// cosine is turn, normalised so that over every way out it sums to four
// pi: the gas's and the haze's, each weighed by how much it scatters.
func (layer optics) phase(turn float64) float64 {
	gas, haze := layer.scatter, layer.haze*layer.hazeAlbedo
	if gas+haze == 0 {
		return 0
	}
	return (gas*rayleigh(turn) + haze*henyey(layer.hazeLean, turn)) /
		(gas + haze)
}

// rayleigh is a molecule's phase: three quarters of one and the turn's
// cosine squared.
func rayleigh(turn float64) float64 {
	return 0.75 * (1 + turn*turn)
}

// henyey is the henyey-greenstein phase of a haze leaning forward by lean: zero
// throws light alike every way, near one throws it on.
func henyey(lean, turn float64) float64 {
	square := lean * lean
	return (1 - square) / math.Pow(1+square-2*lean*turn, 1.5)
}

// reflected is the light a column sends back toward an eye, as radiance
// over the star's light on a surface square to it. the stack runs from
// the top down; the deck below it sends back its albedo alike every way.
//
// star and eye are the cosines of their heights above the horizon, and
// turn the cosine of the angle the light is turned through.
//
// each layer scatters once: its share of the light, from its depth on
// the way in and the way out, times the depth above it on both ways.
// the deck sees what reaches it and sends back what all the depth
// leaves.
func reflected(stack []optics, deck, star, eye, turn float64) float64 {
	slant := 1/star + 1/eye
	above, light := 0.0, 0.0
	for _, layer := range stack {
		tau := layer.depth()
		if tau > 0 {
			albedo := layer.scattered() / tau
			light += albedo * layer.phase(turn) / (4 * math.Pi) *
				star / (star + eye) * math.Exp(-above*slant) *
				-math.Expm1(-tau*slant)
		}
		above += tau
	}
	return light + deck*star/math.Pi*math.Exp(-above*slant)
}

// scatterTurn is the cosine of the angle light is turned through, from
// a star at a height whose cosine is star to an eye at one whose cosine
// is eye, round from it by round radians: the light travels away from
// the star and leaves toward the eye.
func scatterTurn(star, eye, round float64) float64 {
	return -(star*eye + math.Sqrt(1-star*star)*math.Sqrt(1-eye*eye)*
		math.Cos(round))
}

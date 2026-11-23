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
	made := layer.leaf()
	return made.phase(turn, made.lobe.at(turn))
}

// a leaf is a layer with all that does not depend on the angles worked
// out: its optics as written, its whole depth and whether it has any, the
// share of what it takes that it scatters, the gas's and haze's
// scattering and their sum, and the haze's lobe.
type leaf struct {
	raw                 optics
	tau, albedo         float64
	open                bool
	gas, haze, together float64
	lobe                lobe
	slot                int
}

// leaf is the layer worked out once.
func (layer optics) leaf() leaf {
	gas, haze := layer.scatter, layer.haze*layer.hazeAlbedo
	tau := layer.depth()
	made := leaf{raw: layer, tau: tau, open: tau > nothing, gas: gas,
		haze: haze, together: gas + haze, lobe: lobeOf(layer.hazeLean)}
	if made.open {
		made.albedo = layer.scattered() / tau
	}
	return made
}

// phase is how the leaf throws light turned through an angle whose cosine
// is turn, given its haze's henyey phase at that turn, which every layer
// with the same lobe shares: its optics' phase.
func (layer *leaf) phase(turn, tilt float64) float64 {
	if layer.together == nothing {
		return nothing
	}
	return (layer.gas*rayleigh(turn) + layer.haze*tilt) / layer.together
}

// rayleighScale is three quarters, which a molecule's phase is normalised by:
// defined by rayleigh's phase function.
const rayleighScale = 0.75

// rayleigh is a molecule's phase: three quarters of one and the turn's
// cosine squared.
func rayleigh(turn float64) float64 {
	return rayleighScale * (whole + turn*turn)
}

// henyey is the henyey-greenstein phase of a haze leaning forward by lean: zero
// throws light alike every way, near one throws it on.
func henyey(lean, turn float64) float64 {
	return lobeOf(lean).at(turn)
}

// a lobe is henyey's phase of one lean, less the turn: the terms that
// stay the same at every angle.
type lobe struct{ rest, near, twice float64 }

// lobeOf is the lobe of a haze leaning forward by lean.
func lobeOf(lean float64) lobe {
	square := lean * lean
	return lobe{rest: whole - square, near: whole + square,
		twice: double * lean}
}

// henyeyPower is three halves, the power henyey's phase divides by: defined
// by the henyey-greenstein function.
const henyeyPower = 1.5

// at is henyey's phase through an angle whose cosine is turn.
func (shape lobe) at(turn float64) float64 {
	return shape.rest / math.Pow(shape.near-shape.twice*turn, henyeyPower)
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
	leaves := make([]leaf, len(stack))
	tilts := make([]float64, len(stack))
	for k, layer := range stack {
		leaves[k] = layer.leaf()
		leaves[k].slot = k
		tilts[k] = leaves[k].lobe.at(turn)
	}
	return shone(leaves, tilts, deck, star, eye, turn)
}

// shone is reflected for a stack already worked out, given each lobe's
// henyey phase at the turn, by the leaves' slots.
func shone(stack []leaf, tilts []float64, deck, star, eye,
	turn float64) float64 {
	slant := whole/star + whole/eye
	var above, light float64
	for k := range stack {
		layer := &stack[k]
		if layer.open {
			phase := layer.phase(turn, tilts[layer.slot])
			light += layer.albedo * phase / sphere *
				star / (star + eye) * math.Exp(-above*slant) *
				-math.Expm1(-layer.tau*slant)
		}
		above += layer.tau
	}
	return light + deck*star/math.Pi*math.Exp(-above*slant)
}

// scatterTurn is the cosine of the angle light is turned through, from
// a star at a height whose cosine is star to an eye at one whose cosine
// is eye, round from it by round radians: the light travels away from
// the star and leaves toward the eye.
func scatterTurn(star, eye, round float64) float64 {
	return -(star*eye + math.Sqrt(whole-star*star)*math.Sqrt(whole-eye*eye)*
		math.Cos(round))
}

// Package tissue models diffuse flux crossing finite layers of an optical
// material. It works independently of any surface reflectance texture.
package tissue

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// Layer describes a homogeneous, plane-parallel layer. Thickness is in
// metres. Absorption and Backscatter are the effective two-flux coefficients
// per metre for each RGB band. Backscatter couples the two diffuse fluxes.
type Layer struct {
	Thickness               float64
	Absorption, Backscatter maths.Light
}

// SlabResponse records fractions of incident diffuse flux. Transmission is
// reciprocal for the matched-index layer interfaces considered here.
type SlabResponse struct {
	FrontReflectance, BackReflectance maths.Light
	Transmittance                     maths.Light
}

// SolveSlab combines the layers from front to back. It includes repeated
// diffuse reflections between index-matched layers. An empty stack
// transmits all diffuse light.
func SolveSlab(layers []Layer) (SlabResponse, error) {
	return SolveScaledSlab(layers, 1)
}

// SolveScaledSlab scales every layer thickness by the same finite,
// nonnegative factor before solving the stack.
func SolveScaledSlab(layers []Layer, scale float64) (SlabResponse, error) {
	if !nonnegative(scale) {
		return SlabResponse{}, fmt.Errorf("tissue: scale must be finite and nonnegative")
	}
	stack := [3]twoFlux{{through: 1}, {through: 1}, {through: 1}}
	for index, layer := range layers {
		if !nonnegative(layer.Thickness) {
			return SlabResponse{}, fmt.Errorf("tissue: layer %d needs finite nonnegative thickness", index)
		}
		absorption := [3]float64{layer.Absorption.R, layer.Absorption.G, layer.Absorption.B}
		backscatter := [3]float64{layer.Backscatter.R, layer.Backscatter.G, layer.Backscatter.B}
		thickness := layer.Thickness * scale
		if !nonnegative(thickness) {
			return SlabResponse{}, fmt.Errorf("tissue: layer %d needs finite scaled thickness", index)
		}
		for band := range stack {
			if !nonnegative(absorption[band]) || !nonnegative(backscatter[band]) {
				return SlabResponse{}, fmt.Errorf("tissue: layer %d band %d needs finite nonnegative coefficients", index, band)
			}
			k := absorption[band] * thickness
			s := backscatter[band] * thickness
			if !nonnegative(k + 2*s) {
				return SlabResponse{}, fmt.Errorf("tissue: layer %d band %d has unrepresentable optical depth", index, band)
			}
			stack[band] = addLayers(stack[band], slabBand(k, s))
		}
	}
	return SlabResponse{
		FrontReflectance: maths.RGB(stack[0].front.reflect, stack[1].front.reflect, stack[2].front.reflect),
		BackReflectance:  maths.RGB(stack[0].back.reflect, stack[1].back.reflect, stack[2].back.reflect),
		Transmittance:    maths.RGB(stack[0].through, stack[1].through, stack[2].through),
	}, nil
}

// Keep absorption explicitly. Subtracting a reflectance near one would
// erase small escape probabilities when adding very opaque layers.
type faceFlux struct{ reflect, absorb float64 }

type twoFlux struct {
	front, back faceFlux
	through     float64
}

// slabBand solves the continuous Kubelka–Munk equations for one layer.
// Absorption and backscatter are optical depths. Dividing the solution
// by cosh(width) avoids overflow in an optically thick layer.
func slabBand(absorption, backscatter float64) twoFlux {
	width := math.Sqrt(absorption) * math.Sqrt(absorption+2*backscatter)
	shape := 1.0
	if width > 0 {
		shape = math.Tanh(width) / width
	}
	denominator := 1 + (absorption+backscatter)*shape
	decay := math.Exp(-width)
	coshFactor := 1 + decay*decay
	sech := 2 * decay / coshFactor
	oneMinusDecay := -math.Expm1(-width)
	oneMinusSech := oneMinusDecay * oneMinusDecay / coshFactor
	side := faceFlux{
		reflect: (backscatter * shape) / denominator,
		absorb:  (absorption*shape + oneMinusSech) / denominator,
	}
	return twoFlux{front: side, back: side, through: sech / denominator}
}

func addLayers(first, second twoFlux) twoFlux {
	// This is 1 - R_back(first) R_front(second), written as a sum of
	// escape and absorption shares to retain precision near one.
	escape := first.through + first.back.absorb +
		first.back.reflect*(second.through+second.front.absorb)
	if escape == 0 {
		return twoFlux{front: first.front, back: second.back}
	}
	fromFront := min(1, first.through/escape)
	fromBack := min(1, second.through/escape)
	through := min(first.through*fromBack, second.through*fromFront)
	return twoFlux{
		front: faceFlux{
			reflect: min(1-through, first.front.reflect+first.through*fromFront*second.front.reflect),
			absorb:  first.front.absorb + fromFront*(second.front.absorb+second.front.reflect*first.back.absorb),
		},
		back: faceFlux{
			reflect: min(1-through, second.back.reflect+second.through*fromBack*first.back.reflect),
			absorb:  second.back.absorb + fromBack*(first.back.absorb+first.back.reflect*second.front.absorb),
		},
		through: through,
	}
}

func nonnegative(value float64) bool { return value >= 0 && value <= math.MaxFloat64 }

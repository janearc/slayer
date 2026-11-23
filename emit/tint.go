package emit

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/observer"
)

// Tinted is a description whose light is a colour and not a black body's:
// amounts of light from the three srgb lamps, in linear light, and its
// luminance at a metre in the kit's relative light, as Relatively gives
// it. A description that is not one says so with false.
//
// It is how a lamp is magenta or cyan: a neon tube's light is that of the
// gas, and no heat of any black body is its colour.
type Tinted interface {
	Tint() (colour maths.Light, luminance float64, tinted bool)
}

// RelativeTint is the light of a colour, scaled so that its luminance at a
// metre is the one given, as Relative is for a black body: the same
// relative light, with the colour chosen and not found from a heat. It
// refuses a colour that is no light, one the three lines a tint is made
// of cannot give, and a luminance that is not above nought.
func RelativeTint(colour maths.Light, luminance float64) (bands.Light, error) {
	spectrum, err := observer.Tinted(colour.R, colour.G, colour.B)
	if err != nil {
		return bands.Light{}, err
	}
	scale := luminance / colour.Luminance()
	if !(luminance > 0) || math.IsNaN(scale) || math.IsInf(scale, 0) {
		return bands.Light{}, fmt.Errorf("emit: a tint of %v at a "+
			"luminance of %v, which the kit can't carry", colour, luminance)
	}
	return spectrum.Scale(scale), nil
}

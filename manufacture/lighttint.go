package manufacture

import (
	"errors"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/observer"
)

var errTint = errors.New("manufacture: a tinted light needs a luminance " +
	"above nought, and a colour that is some light, from amounts of the " +
	"three srgb lamps")

// tinted is a colour a light has that is no black body's, as amounts of
// light from the three srgb lamps, the spectrum made of it, and how many
// watts of that spectrum make a unit of luminance.
type tinted struct {
	colour   maths.Light
	spectrum bands.Light
	watts    float64
}

// newTint is the spectrum of a colour: of three lines, which light of any
// colour of the srgb lamps is. It refuses a colour the observer refuses.
func newTint(colour maths.Light) (spectrum, error) {
	made, err := observer.Tinted(colour.R, colour.G, colour.B)
	if err != nil {
		return spectrum{}, err
	}
	var sum float64
	first, last := made.Span()
	for band := first; band <= last; band++ {
		sum += made.Power(band)
	}
	return spectrum{tint: &tinted{colour: colour, spectrum: made,
		watts: sum / colour.Luminance()}}, nil
}

// tinted is whether the spectrum is a colour's.
func (of spectrum) tinted() bool { return of.tint != nil }

// newTintedLight is a bare light of a colour, its strength given in the
// kit's relative light as a set's lamp is: its luminance at a metre. It
// shines every way. A neon sign is such a light: the colour of its glass,
// which no heat of a black body gives.
//
// Its power, which the roads that read
// watts a steradian use, is as many watts as make that luminance in that
// colour, so that two signs of one luminance are as bright however they are
// coloured, as they are in the relative light. It refuses a colour that is
// no light, and a luminance that is not above nought.
func newTintedLight(colour maths.Light, luminance float64) (light, error) {
	if !positive(luminance) {
		return light{}, errTint
	}
	of, err := newTint(colour)
	if err != nil {
		return light{}, err
	}
	made, err := newLight(of, measure.Watts(luminance*of.tint.watts),
		everyway{})
	made.luminance = luminance
	return made, err
}

// Tint is the colour of the light and its luminance at a metre, if it is a
// light of a colour of its own.
func (made light) Tint() (colour maths.Light, luminance float64,
	tint bool) {
	if !made.colour.tinted() {
		return maths.Light{}, 0, false
	}
	return made.colour.tint.colour, made.luminance, true
}

// lampLight is the light a drawn lamp is: of its colour, if it has one of
// its own, and otherwise of the black body of its heat.
func lampLight(lamp *cities.Lamp) (light, error) {
	if lamp.Tint != (maths.Light{}) {
		return newTintedLight(lamp.Tint, lamp.Luminance)
	}
	return newRelativeLight(measure.Kelvin(lamp.Heat), lamp.Luminance)
}

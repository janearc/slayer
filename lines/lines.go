// Package lines builds emission colours from elemental spectral lines, using
// libtheme's light representation.
//
// Wavelengths follow NIST's values in nanometres in air. Relative strengths are
// approximate and normalized to each element's strongest line: useful for
// illustration, not quantitative spectroscopy.
package lines

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/observer"
)

// Element is one gas and the lines it glows in
type Element struct {
	name  string
	lines map[float64]float64
}

// known is every element this package has lines for, wavelength to
// strength
var known = map[string]map[float64]float64{
	// hydrogen's balmer lines, at a nebula's usual ratios
	"hydrogen": {656.28: 1, 486.13: 0.35, 434.05: 0.16, 410.17: 0.09},
	// neon's orange and red, the colour of a sign
	"neon": {585.25: 1, 640.22: 0.9, 703.24: 0.5, 614.31: 0.45,
		588.19: 0.4, 594.48: 0.4, 609.62: 0.35, 616.36: 0.3,
		621.73: 0.3, 626.65: 0.3, 633.44: 0.35, 638.30: 0.3,
		650.65: 0.35, 659.90: 0.2, 692.95: 0.3},
	// argon once ionized, as a magnetar leaves it: the blue and green of
	// an argon laser
	"argon": {487.99: 1, 514.53: 0.8, 476.49: 0.4, 496.51: 0.35,
		434.81: 0.3, 465.79: 0.25, 457.94: 0.2, 472.69: 0.2,
		501.72: 0.15, 454.50: 0.15},
}

// Of is the element with that name, or an error saying there are no
// lines for it
func Of(name string) (Element, error) {
	found, ok := known[name]
	if !ok {
		return Element{}, fmt.Errorf("lines: none known for %q", name)
	}
	return Element{name: name, lines: found}, nil
}

// Glow is the element's colour as three channels, scaled to the given
// luminance, so a mix of gases changes the colour and not how bright the
// gas is.
//
// A line the three lamps cannot reach would need a negative amount of
// one of them; that part is dropped, since a glow gives light and cannot
// take it away.
func (gas Element) Glow(luminance float64) maths.Light {
	red, green, blue := srgb.Light(observer.Swatch(bands.Lines(gas.lines)))
	colour := maths.RGB(math.Max(0, red), math.Max(0, green),
		math.Max(0, blue))
	if colour.Luminance() <= 0 {
		return maths.Light{}
	}
	return colour.Scale(luminance / colour.Luminance())
}

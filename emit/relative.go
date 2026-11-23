package emit

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/observer"
)

// Relatively is a description that gives its strength in the kit's own
// relative light, as a set's lamp does, and not in watts a steradian: the
// heat of the black body that is its light, in kelvin, and its luminance
// at a metre. A description that is not one says so with false.
type Relatively interface {
	Relative() (kelvin, luminance float64, relative bool)
}

// Relative is the light of a black body at a temperature, scaled so that
// its luminance at a metre is the one given: the same light a lamp of a
// set has, from the same working. It refuses a body so cold its light is
// all nought, or so hot it overflows.
func Relative(kelvin, luminance float64) (bands.Light, error) {
	spectrum := bands.Split(swatch.Blackbody(kelvin))
	red, green, blue := srgb.Light(observer.Swatch(spectrum))
	scale := luminance / maths.RGB(red, green, blue).Luminance()
	if math.IsNaN(scale) || math.IsInf(scale, 0) {
		return bands.Light{}, fmt.Errorf("emit: a black body at %v K, whose "+
			"light the kit can't carry", kelvin)
	}
	return spectrum.Scale(scale), nil
}

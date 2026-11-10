package flame

import (
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// ColourSpec supplies a thermal lookup's temperature range and picture scale.
type ColourSpec struct {
	Coolest, Hottest measure.Temperature
	Exposure         float64
}

// Palette reads a retained thermal colour table without rebuilding it.
// Colours returns an implementation of this contract.
type Palette interface {
	At(measure.Temperature) maths.Light
}

// Colours prepares black-body colour with the same observer as the flame.
func Colours(spec ColourSpec) (glowing, error) {
	cold, hot := spec.Coolest.InKelvin(), spec.Hottest.InKelvin()
	if !sane(cold) || !sane(hot) || hot < cold || !sane(spec.Exposure) {
		return glowing{}, errFlame
	}
	return newGlowing(cold, hot, spec.Exposure), nil
}

// At interpolates the prepared radiance, clamping to its temperature range.
func (table glowing) At(heat measure.Temperature) maths.Light { return table.at(heat) }

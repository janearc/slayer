// Package record models one photosensitive layer: its spectral sensitivity,
// development curve, grain, and the light its developed image absorbs.
package record

import (
	"errors"

	"github.com/janearc/libtheme-css/spaces/ok"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/crystal"
	"github.com/janearc/slayer/ninjafilm/curve"
	"github.com/janearc/slayer/ninjafilm/number"
)

// what a record is refused for: a window no lamp shines in, or an image
// that blocks no lamp
var (
	errBlind  = errors.New("ninjafilm: a record sees some of the light")
	errUnseen = errors.New("ninjafilm: a record's image blocks some " +
		"of the light")
)

// halfway is the lightness of the middle grey, defined as half of libtheme's
// scale. The scale runs from black at zero to white at one.
const halfway = 0.5

// middle is how much light the middle grey gives back of the light that
// white does. Stocks are rated so that this grey lands on the middle of
// their curves.
var middle = middleGrey()

// middleGrey is the luminance of the grey halfway up libtheme's
// lightness.
func middleGrey() float64 {
	_, luminance, _ := ok.Grey(halfway).Swatch().XYZ()
	return luminance
}

// Record is one sensitive layer of a stock. Its window is the lamps
// whose light exposes it. Its image is the lamps its developed self
// blocks. Its grain and tone are the crystal it is made of and the
// curve it develops along.
type Record struct {
	window maths.Light
	image  maths.Light
	grain  crystal.Crystal
	tone   curve.Curve
}

// New is a record from a window and an image with a crystal and a
// curve, or why not.
func New(window, image maths.Light, grain crystal.Crystal,
	tone curve.Curve) (Record, error) {
	if !number.Positive(number.Sum(window)) {
		return Record{}, errBlind
	}
	if !number.Positive(number.Sum(image)) {
		return Record{}, errUnseen
	}
	return Record{window, image, grain, tone}, nil
}

// Window is the lamps whose light exposes the record.
func (layer Record) Window() maths.Light { return layer.window }

// Image is the lamps the developed record blocks.
func (layer Record) Image() maths.Light { return layer.image }

// exposure is how strongly a light exposes the record against the
// middle grey. One is the middle of its curve. White light exposes
// every record alike, whatever its window.
func (layer Record) exposure(light maths.Light) float64 {
	return number.Sum(light.Times(layer.window)) /
		number.Sum(layer.window) / middle
}

// Developed is the density the record comes to over an area where a
// light fell. Its curve says what the density is. That is counted out
// in crystals with a draw of chance.
func (layer Record) Developed(light maths.Light, area,
	chance float64) float64 {
	smooth := layer.tone.Density(layer.exposure(light))
	found := crystal.Counted(layer.grain.Expected(smooth, area), chance)
	return layer.grain.Density(found, area)
}

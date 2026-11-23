package develop

import (
	"fmt"
	"image"
	"math"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/observer"
	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
)

type Options struct {
	Exposure float64
	Knee     float64
	Top      float64
	Bloom    float64
	White    float64
}

// DefaultOptions is Options with the knee at 0.7 and the top at 8, and the
// exposure, bloom, and white balance at zero, which leaves balance off.
func DefaultOptions() Options {
	return Options{Knee: .7, Top: 8}
}

// Develop sums frames into one picture and tone maps it. It refuses a nil
// frame. If White is not zero it first balances the sum to that many
// kelvin. Exposure is in stops, so the sum is scaled by two to its power.
func Develop(frames []*frame.Frame, options Options) (*image.RGBA, error) {
	for index, picture := range frames {
		if picture == nil {
			return nil, fmt.Errorf("develop: frame %d is nil", index)
		}
	}
	total, err := raw.Sum(frames)
	if err != nil {
		return nil, err
	}
	if options.White != 0 {
		if err := balance(total, options.White); err != nil {
			return nil, err
		}
	}
	tone := colour.Tone{Exposure: math.Exp2(options.Exposure), Knee: options.Knee, Top: options.Top, Bloom: options.Bloom}
	return tone.Develop(total), nil
}

// balance scales the red and blue of total, in place, so that a blackbody at
// kelvin comes out with equal red, green, and blue. It refuses a nil frame,
// a temperature that is not positive and finite, and one with no red or blue.
func balance(total *frame.Frame, kelvin float64) error {
	if total == nil {
		return fmt.Errorf("develop: frame is nil")
	}
	if !(kelvin > 0) || math.IsInf(kelvin, 0) {
		return fmt.Errorf("a white balance of %v kelvin", kelvin)
	}
	spectrum := bands.Split(swatch.Blackbody(kelvin))
	red, green, blue := srgb.Light(observer.Swatch(spectrum))
	if !(red > 0 && blue > 0) {
		return fmt.Errorf("no white at %v kelvin", kelvin)
	}
	gains := [3]float32{float32(green / red), 1, float32(green / blue)}
	for i := range total.Pix {
		total.Pix[i].R *= gains[0]
		total.Pix[i].B *= gains[2]
	}
	return nil
}

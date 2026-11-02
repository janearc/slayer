package colour

import (
	"fmt"
	"image"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/frame"
	"github.com/janearc/slayer/observer"
)

// errSizes is what a composite of layers of two sizes panics with.
const errSizes = "colour: layer %d is %d by %d, and layer 0 is %d by " +
	"%d: a composite's layers are one size"

// Layer is one band of a composite. It is a frame rendered in that band
// with how strongly it shows.
//
// Bands of zero are visible light. They add as their own colour. The
// strength of any other band is the red of its frame alone. Its green
// and blue are unread. Gains of zero are one, full strength. So no
// gain turns a layer off. Leave the layer out instead.
//
// Its tint is the visible colour the band stands in, when it is set. It
// replaces the colour of libtheme for the range. Palettes do so. The
// hubble palette paints sulphur red, hydrogen green and oxygen blue.
//
//ninja:vow practice slayer's false colour pictures hold their layers
type Layer struct {
	band maths.Band
	grid *frame.Frame
	gain float64
	tint maths.Light
}

// NewLayer is a layer of a band. It holds a frame rendered in the band
// with a gain and a tint.
//
//ninja:vow practice slayer's false colour pictures make their layers here
func NewLayer(band maths.Band, grid *frame.Frame, gain float64,
	tint maths.Light) Layer {
	return Layer{band, grid, gain, tint}
}

// False is a false colour picture. Every layer's band is run through the
// profile of libtheme for its range. It becomes the visible colour that
// stands for it, times the strength of the band in each pixel. Then the
// layers add.
//
// Visible layers add as themselves. It is the False of a tone with no
// tone. This is what nasa calls false colour, and it says so.
//
//ninja:vow practice slayer's false colour pictures develop here
func False(layers []Layer) *image.RGBA { return Tone{}.False(layers) }

// False is the false colour picture through the tone. Exposure and the
// shoulder are applied to the sum, after the layers add.
//
//ninja:vow practice slayer's false colour pictures develop here
func (tone Tone) False(layers []Layer) *image.RGBA {
	if len(layers) == none {
		return image.NewRGBA(image.Rect(none, none, none, none))
	}
	across, down := layers[first].grid.W, layers[first].grid.H
	for index, layer := range layers {
		if layer.grid.W != across || layer.grid.H != down {
			panic(sizeRefusal(index, layer, across, down))
		}
	}
	lights := make([]maths.Light, across*down)
	for y := range down {
		for x := range across {
			lights[y*across+x] = layered(layers, x, y)
		}
	}
	return tone.picture(lights, across, down)
}

// sizeRefusal is what a layer of another size is refused with.
func sizeRefusal(index int, layer Layer, across, down int) string {
	return fmt.Sprintf(errSizes, index, layer.grid.W, layer.grid.H, across,
		down)
}

// layered is the light at a pixel with every layer added.
func layered(layers []Layer, x, y int) maths.Light {
	var sum maths.Light
	for _, layer := range layers {
		seen := layer.grid.At(x, y).Light()
		gain := layer.gain
		if gain == none {
			gain = whole
		}
		if layer.band.Nanometres == none {
			sum = sum.Add(seen.Scale(gain))
			continue
		}
		stand := Stands(layer.band)
		if layer.tint != (maths.Light{}) {
			stand = layer.tint
		}
		sum = sum.Add(stand.Scale(seen.R * gain))
	}
	return sum
}

// shown is the ranges a band may stand in, in the order libtheme kept
// them. Fermi is left out: the sensitivity of an instrument, not a
// range of the spectrum.
var shown = []radiation.Range{radiation.Gamma(), radiation.XRay(),
	radiation.ExtremeUltraviolet(), radiation.Ultraviolet(),
	radiation.Infrared(), radiation.Radio(), radiation.Microwave()}

// Stands is the visible light that stands for a band. The range of the
// band is laid across the visible. So the short end of gamma is violet
// and its long end is red. So it is for each range.
//
// Bands in no range are white. Visible light is among them.
//
// Where ranges overlap the first in shown wins. Radio comes before
// microwave. Microwave is never reached. Infrared comes over the short
// end of radio. Lamb drew it so, and the order is kept.
//
//ninja:vow practice slayer's false colour pictures ask what a band stands for
func Stands(band maths.Band) maths.Light {
	for _, span := range shown {
		if span.Holds(band.Nanometres) {
			swatch := observer.FalseColour(span, band.Nanometres)
			red, green, blue := srgb.Light(swatch)
			light := maths.RGB(max(red, none), max(green, none),
				max(blue, none))
			// at unit strength the colour is as bright as white
			if luminance := light.Luminance(); luminance > none {
				light = light.Scale(whole / luminance)
			}
			return light
		}
	}
	return maths.Grey(whole)
}

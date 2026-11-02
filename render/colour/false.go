package colour

import (
	"fmt"
	"image"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/observer"
	"github.com/janearc/slayer/render/frame"
)

// Layer is one band of a composite: a frame rendered in that band, and how
// strongly it shows.
//
// a Band of zero is visible light and adds as its own colour; any other band's
// strength is its frame's red alone, its green and blue unread. a Gain of zero
// is one, full strength, so no gain turns a layer off: leave the layer out
// instead.
type Layer struct {
	Band  maths.Band
	Frame *frame.Frame
	Gain  float64
	// Tint, when set, is the visible colour the band stands in instead
	// of libtheme's for its range: a palette's, as the hubble palette
	// paints sulphur red, hydrogen green and oxygen blue.
	Tint maths.Light
}

// False is a false colour picture: every layer's band, through
// libtheme's profile for the range the band is in, becomes the visible
// colour that stands for it, times the band's strength in each pixel,
// and the layers add.
//
// A visible layer adds as itself. It is the tone's False with no tone.
//
// This is what nasa calls false colour, and it says so.
func False(layers []Layer) *image.RGBA { return Tone{}.False(layers) }

// False is the false colour picture through the tone: exposure and
// the shoulder applied to the sum, after the layers add.
func (t Tone) False(layers []Layer) *image.RGBA {
	if len(layers) == 0 {
		return image.NewRGBA(image.Rect(0, 0, 0, 0))
	}
	w, h := layers[0].Frame.W, layers[0].Frame.H
	for i, l := range layers {
		if l.Frame.W != w || l.Frame.H != h {
			panic(fmt.Sprintf("colour: layer %d is %d by %d, and "+
				"layer 0 is %d by %d: a composite's layers "+
				"are one size", i, l.Frame.W, l.Frame.H, w, h))
		}
	}
	lights := make([]maths.Light, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var sum maths.Light
			for _, l := range layers {
				px := l.Frame.At(x, y).Light()
				gain := l.Gain
				if gain == 0 {
					gain = 1
				}
				if l.Band.Nanometres == 0 {
					sum = sum.Add(px.Scale(gain))
					continue
				}
				stand := Stands(l.Band)
				if l.Tint != (maths.Light{}) {
					stand = l.Tint
				}
				sum = sum.Add(stand.Scale(px.R * gain))
			}
			lights[y*w+x] = sum
		}
	}
	return t.picture(lights, w, h)
}

// shown is the ranges a band may stand in, in the order libtheme kept
// them, which Stands keeps: fermi is left out, an instrument's
// sensitivity and not a range of the spectrum.
var shown = []radiation.Range{radiation.Gamma(), radiation.XRay(),
	radiation.ExtremeUltraviolet(), radiation.Ultraviolet(),
	radiation.Infrared(), radiation.Radio(), radiation.Microwave()}

// Stands is the visible light that stands for a band, as the band's range
// laid across the visible gives it, so gamma's short end is violet and
// its long end red, and so for each range. A band in no range is white,
// visible light among them.
//
// where ranges overlap the first in shown wins: radio before microwave,
// which is never reached, and infrared over radio's short end. it was
// drawn so, and the order is kept.
func Stands(band maths.Band) maths.Light {
	for _, span := range shown {
		if span.Holds(band.Nanometres) {
			swatch := observer.FalseColour(span, band.Nanometres)
			r, g, b := srgb.Light(swatch)
			l := maths.RGB(max(r, 0), max(g, 0), max(b, 0))
			// at unit strength the colour is as bright as white
			if y := l.Luminance(); y > 0 {
				l = l.Scale(1 / y)
			}
			return l
		}
	}
	return maths.Grey(1)
}

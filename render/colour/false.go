package colour

import (
	"image"

	"github.com/janearc/slayer/maths"
	ninjacolour "github.com/janearc/slayer/ninjafilm/colour"
	"github.com/janearc/slayer/render/carry"
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
// the shoulder applied to the sum, after the layers add. The layers are
// carried to ninjafilm's frames and composed by ninjafilm's False, which
// refuses layers of two sizes as this package always did.
func (t Tone) False(layers []Layer) *image.RGBA {
	carried := make([]ninjacolour.Layer, len(layers))
	for i, l := range layers {
		carried[i] = ninjacolour.NewLayer(l.Band, carry.Ninja(l.Frame),
			l.Gain, l.Tint)
	}
	return t.Print().False(carried)
}

// Stands is the visible light that stands for a band, as the band's range
// laid across the visible gives it, so gamma's short end is violet and
// its long end red, and so for each range. A band in no range is white,
// visible light among them. It is ninjafilm's.
func Stands(band maths.Band) maths.Light { return ninjacolour.Stands(band) }

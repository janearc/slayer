// This file hands a sea with no edge to a renderer that keeps it on a
// device: the same bands and the same step bound Hit takes, as numbers.

package arid

import "github.com/janearc/slayer/oceantex"

// SeaPacked is a sea with no edge as a renderer that keeps it on a device
// needs it to meet a ray as Hit does and to shade it as Seen does.
//
// It carries no coast: a device given a sea over ground draws the open
// sea's waves everywhere, the ground standing over them where it is higher.
type SeaPacked struct {
	// Bands are the sea's wave bands, each its own side and grid of
	// heights and slopes, and the push of its water. They are the sea's
	// own and must not be written. The CPU sea sums them longest first,
	// each band after the first read where the longer bands' water came
	// from (Sea.waves), by the push. A device that sums them without the
	// push, as the GPU's does not yet read it, searches another surface.
	Bands []*oceantex.Field
	// Slopes are each band's own slope variance, along the wind and
	// across it together, in the order of Bands.
	Slopes []float64
	// Reach is how far above and below its mean the surface stays, in
	// metres: the slab Hit searches.
	Reach float64
	// Steepest bounds the surface's rise a metre in any direction: the
	// rate Hit steps by.
	Steepest float64
	// Touch is how near the surface, in metres, a ray is taken to have
	// met it.
	Touch float64
	// MostSteps is the most steps a ray takes before it ends on the mean.
	MostSteps int
	// ShownWhole and ShownNone are where a band fades from a pixel, in
	// samples the pixel covers: oceantex.PixelShareSpan.
	ShownWhole, ShownNone float64
}

// Pack is the sea as a renderer that keeps it on a device needs it. The
// bands are not copied.
func (sea *Sea) Pack() SeaPacked {
	whole, none := oceantex.PixelShareSpan()
	return SeaPacked{Bands: sea.bands, Slopes: sea.slopes, Reach: sea.reach,
		Steepest: sea.steepest, Touch: seaTouch, MostSteps: seaMostSteps,
		ShownWhole: whole, ShownNone: none}
}

//go:build darwin

package gpu

import (
	"runtime/debug"

	"github.com/janearc/slayer/metal"
)

// stripe is how many rows of samples one stripe of rays holds: as many
// whole bands as fit in stripeBytes, and one band at least. It is asked
// only of a picture whose rays do not fit whole, so its width is never
// nought.
func (lensed *through) stripe() int {
	fit := stripeBytes / (lensed.width * rayStride)
	return max(fit/rows*rows, rows)
}

// releasePacked is what drawn does once the device holds every input
// and nothing on the host reads them again: the scene's packed faces,
// bends and boxes, the largest of the inputs, are let go from the scene
// and from the list of inputs, and the pages freed are handed back. A
// keeper's pool still holds its own, which it keeps for the next frame.
func (encoded *scene) releasePacked(inputs [][]byte) func() {
	return func() {
		for _, at := range []int{trianglesAt, nodesAt, bendsAt} {
			inputs[at] = nil
		}
		encoded.triangles.bytes, encoded.nodes.bytes = nil, nil
		encoded.bends.bytes = nil
		debug.FreeOSMemory()
		memoryStage("packed-released")
	}
}

// drawn runs the trace over every sample and hands its bands to the
// finisher, giving back the bytes of rays it made on the way. A picture
// whose rays the kernel makes, or whose rays were made whole, is drawn
// in one go; one whose rays would not fit in an input, a stripe at a
// time, each stripe's rays made just before it is drawn.
func (encoded *scene) drawn(kernel *metal.Kernel, inputs [][]byte,
	width, height int, band *bandFinisher) (int64, error) {
	var made int64
	lensed := encoded.through
	if lensed == nil || lensed.whole() {
		return made, kernel.GridBandsSent(width, height, rows,
			encoded.perSample(), inputs, encoded.releasePacked(inputs),
			band.consume)
	}
	for first := firstRow; first < height; first += lensed.stripe() {
		last := min(first+lensed.stripe(), height)
		inputs[raysAt] = lensed.rays(first, last)
		made += int64(len(inputs[raysAt]))
		stripe := shifted{band: band, from: first}
		if err := kernel.GridBands(width, last-first, rows,
			encoded.perSample(), inputs, stripe.consume); err != nil {
			return made, err
		}
	}
	return made, nil
}

// shifted hands a stripe's bands to the picture's finisher, each band's
// first row moved down to where the stripe starts in the picture. Inside
// the stripe the kernel counts rows from nought, which is what finds
// each sample's ray in the stripe's own.
type shifted struct {
	band *bandFinisher
	from int
}

// consume is the finisher's consume, at the band's row in the picture.
func (stripe shifted) consume(first, down int, samples []float32) error {
	return stripe.band.consume(stripe.from+first, down, samples)
}

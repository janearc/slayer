package searching

import (
	"fmt"
	"math"
	"slices"

	"github.com/janearc/slayer/render/frame"
)

// mergeBeamLight adds complementary linear radiance before development.
// Geometry metadata comes from the base, without another alpha weighting.
// Refusal leaves both inputs untouched, including on radiance overflow.
func mergeBeamLight(base, beam *frame.Frame) (*frame.Frame, error) {
	if !beamFrameValid(base) || !beamFrameValid(beam) ||
		base.W != beam.W || base.H != beam.H {
		return nil, fmt.Errorf("searching: complementary beam frames must match")
	}
	merged := &frame.Frame{W: base.W, H: base.H, Pix: slices.Clone(base.Pix)}
	for index := range merged.Pix {
		pixel := &merged.Pix[index]
		other := beam.Pix[index]
		pixel.R += other.R
		pixel.G += other.G
		pixel.B += other.B
		for _, value := range []float32{pixel.R, pixel.G, pixel.B} {
			if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
				return nil, fmt.Errorf("searching: nonfinite beam radiance")
			}
		}
	}
	return merged, nil
}

// beamFrameValid checks exact storage without multiplying dimensions
// that may overflow. A zero-size frame is not a renderable picture.
func beamFrameValid(picture *frame.Frame) bool {
	return picture != nil && picture.W > 0 && picture.H > 0 &&
		len(picture.Pix)/picture.H == picture.W &&
		len(picture.Pix)%picture.H == 0
}

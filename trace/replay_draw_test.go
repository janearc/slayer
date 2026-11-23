//go:build draw

package trace

import (
	"testing"
)

// Every pixel replayed alone is the pixel Sampled made, bit for bit: of a sky
// and a smoking plane, mirrors, a puff, a gas, a field, a lit floor and a
// fire, at one, two and three rays per pixel each way, twelve pixels by nine.
// The plain tier holds the same of a smaller picture.
func TestAReplayedPixelIsTheSampledPixelToTheBit(t *testing.T) {
	worlds := replayWorlds()
	worlds["burning"] = burning()
	replaysAreSampled(t, worlds, 12, 9, []int{1, 2, 3})
}

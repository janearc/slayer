package trace

import (
	"math"

	"github.com/janearc/slayer/render/frame"
)

// exactMean averages a pixel's samples, keeping its nearest depth.
func exactMean(count int, sample func(int) (frame.Pixel, bool)) (frame.Pixel, bool) {
	if count < 1 {
		return frame.Pixel{}, false
	}
	mean := frame.Pixel{Depth: float32(math.Inf(1))}
	for index := 0; index < count; index++ {
		each, whole := sample(index)
		if !whole {
			return frame.Pixel{}, false
		}
		mean.R += each.R
		mean.G += each.G
		mean.B += each.B
		mean.Cover += each.Cover
		mean.Depth = min(mean.Depth, each.Depth)
	}
	share := 1 / float32(count)
	mean.R, mean.G, mean.B = mean.R*share, mean.G*share, mean.B*share
	mean.Cover *= share
	return mean, true
}

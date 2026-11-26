package trace

import (
	"fmt"
	"math"
	"testing"

	"github.com/janearc/slayer/render/frame"
)

// These arithmetic checks use supplied pixels; they do not draw a scene.
func TestExactMeanUsesEverySampleAndNearestDepth(t *testing.T) {
	var calls int
	got, whole := exactMean(7, func(index int) (frame.Pixel, bool) {
		calls++
		return frame.Pixel{R: float32(index), G: 2, B: 4, Cover: 1, Depth: float32(10 - index)}, true
	})
	if !whole || calls != 7 || math.Abs(float64(got.R)-3) > 1e-6 || got.G != 2 || got.B != 4 || got.Cover != 1 || got.Depth != 4 {
		t.Fatalf("mean=%+v whole=%t calls=%d", got, whole, calls)
	}
}

func TestExactMeanUsesTheRequestedSampleCounts(t *testing.T) {
	for _, count := range []int{1, 4, 7, 16, 25, 64} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			calls := 0
			got, whole := exactMean(count, func(index int) (frame.Pixel, bool) {
				calls++
				return frame.Pixel{R: float32(index), Depth: float32(count - index)}, true
			})
			want := float64(count-1) / 2
			if !whole || calls != count || math.Abs(float64(got.R)-want) > 1e-5 || got.Depth != 1 {
				t.Fatalf("count=%d mean=%+v whole=%t calls=%d", count, got, whole, calls)
			}
		})
	}
}

func TestExactMeanRefusesAnIncompletePixel(t *testing.T) {
	calls := 0
	_, whole := exactMean(7, func(index int) (frame.Pixel, bool) {
		calls++
		return frame.Pixel{R: 1}, index < 3
	})
	if whole || calls != 4 {
		t.Fatalf("incomplete sample sequence accepted=%t calls=%d", whole, calls)
	}
}

func TestExactMeanRefusesZeroSamples(t *testing.T) {
	_, whole := exactMean(0, func(index int) (frame.Pixel, bool) {
		t.Fatal("zero samples called the sample function")
		return frame.Pixel{}, true
	})
	if whole {
		t.Fatal("zero samples made a complete pixel")
	}
}

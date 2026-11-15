//go:build draw

package hillside

import (
	"image"
	"io"
	"slices"
	"testing"
	"time"

	"github.com/janearc/slayer/gpu"
)

// Resume checks source timestamps, including the hyperlapse offset, not only
// an unchanged PNG hash. Only the damaged entry should be redrawn.
func TestHyperlapseResumeRechecksSourceTime(test *testing.T) {
	plan := sequencePlan{output: test.TempDir(), fps: 6, frames: 3,
		start: 45 * time.Second, sceneSpan: time.Minute}
	state := &sequenceState{Complete: map[int]sequenceEntry{}}
	var times []time.Duration
	draw := func(at time.Duration) (*image.RGBA, gpu.Profile, tally, error) {
		times = append(times, at)
		return image.NewRGBA(image.Rect(0, 0, 1, 1)), gpu.Profile{}, tally{}, nil
	}
	if err := sequencePictures(plan, state, io.Discard, draw); err != nil {
		test.Fatal(err)
	}
	expected := []time.Duration{45 * time.Second, 65 * time.Second,
		85 * time.Second}
	if !slices.Equal(times, expected) {
		test.Fatal("render callback received wrong scene times", times)
	}
	entry := state.Complete[1]
	entry.AtNanoseconds++
	state.Complete[1], times = entry, nil
	if err := sequencePictures(plan, state, io.Discard, draw); err != nil {
		test.Fatal(err)
	}
	if len(times) != 1 || times[0] != 65*time.Second {
		test.Fatal("resume did not isolate the wrong source time", times)
	}
}

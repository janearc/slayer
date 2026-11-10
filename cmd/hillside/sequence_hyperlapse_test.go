package main

import (
	"bytes"
	"image"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
)

// An hour in thirty seconds is 120x: six frames per playback second
// therefore sample scene time twenty seconds apart, without changing fps.
func TestHyperlapseHourFlightUsesSceneTimeForCameraAndZoom(test *testing.T) {
	plan := hyperlapseFixture(test)
	if plan.frames != 180 || plan.fps != 6 || plan.sceneSpan != time.Hour {
		test.Fatal("hyperlapse changed output timing", plan)
	}
	view := asked{wide: 90}
	at := plan.frameTime(90)
	if at != 1845*time.Second {
		test.Fatal("halfway source time is not 30m45s", at)
	}
	if err := plan.withCamera(&view, at); err != nil {
		test.Fatal(err)
	}
	if view.eye.Origin != maths.V(2, 0, 5) || view.wide != 44 {
		test.Fatal("camera and zoom did not sample the same scene time", view)
	}
}

// The clock is evaluated from the frame index, with no accumulated rounding.
func TestHyperlapseFrameClockHasExactTimesAndNoOverflow(test *testing.T) {
	plan := sequencePlan{start: 45 * time.Second, fps: 6,
		frames: 180, sceneSpan: time.Hour}
	for _, sample := range []struct{ index, seconds int }{
		{179, 3625}, {0, 45}, {1, 65}, {90, 1845}, {1, 65},
	} {
		expected := time.Duration(sample.seconds) * time.Second
		if got := plan.frameTime(sample.index); got != expected {
			test.Fatalf("frame %d: %s", sample.index, got)
		}
	}
	plan.start, plan.frames, plan.sceneSpan = 0, 432000, 24*time.Hour
	if got := plan.frameTime(431999); got != 24*time.Hour-200*time.Millisecond {
		test.Fatal("large frame count overflowed scene time", got)
	}
}

// A missing scene span retains the original integer nanosecond clock.
func TestHyperlapseAbsentPreservesRealTimeAndFractionalSpans(test *testing.T) {
	plan := sequencePlan{fps: 6, frames: 60, start: 45 * time.Second}
	if got := plan.frameTime(1); got != 45166666666 {
		test.Fatal("legacy real-time frame changed", got)
	}
	plan.start, plan.frames, plan.sceneSpan = 7, 3, 10
	for index, expected := range []time.Duration{7, 10, 13} {
		if got := plan.frameTime(index); got != expected {
			test.Fatalf("fractional frame %d: %s", index, got)
		}
	}
}

// The ordinary schema, rather than a parallel parser, refuses bad units/keys.
func TestHyperlapseAndZoomSchemaRejectInvalidPlans(test *testing.T) {
	for _, change := range [][2]string{
		{"3600, unit: s", "3600, unit: ms"},
		{"scene-duration:", "scene-duraton:"},
		{"scene-duration: {value: 3600", "scene-duration: {value: 0"},
		{"scene-duration: {value: 3600", "scene-duration: {value: 86401"},
		{"value: 24, unit: deg", "value: 24, unit: rad"},
		{"value: 24, unit: deg", "value: 0, unit: deg"},
		{"value: 64, unit: deg", "value: 180, unit: deg"},
		{"from: {value: 24", "frmo: {value: 24"},
		{"    to: {value: 64, unit: deg}\n", ""},
		{"\n    duration: {value: 3600, unit: s}",
			"\n    duration: {value: 3600, unit: min}"},
		{"\n    ease: linear", "\n    ease: typo"},
	} {
		raw := hyperlapseSource(test)
		changed := strings.Replace(string(raw), change[0], change[1], 1)
		path := filepath.Join(test.TempDir(), "sequence.yaml")
		sequenceWrite(test, path, []byte(changed))
		if _, err := readSequence(path); err == nil {
			test.Fatalf("accepted invalid plan: %s", change[1])
		}
	}
}

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

// A dry run reports both clocks and creates no frame directory.
func TestHyperlapseDryRunReportsBothClocks(test *testing.T) {
	plan := hyperlapseFixture(test)
	var output bytes.Buffer
	if err := sequenceDryRun(&output, *plan); err != nil {
		test.Fatal(err)
	}
	for _, expected := range []string{"scene span: 1h0m0s in 30s",
		"180 frames at 6 fps"} {
		if !strings.Contains(output.String(), expected) {
			test.Fatal("dry run lost timing", output.String())
		}
	}
	if _, err := os.Stat(plan.output); !os.IsNotExist(err) {
		test.Fatal("timing report created frame output")
	}
}

// hyperlapseFixture checks a complete plan through the production reader.
func hyperlapseFixture(test *testing.T) *sequencePlan {
	test.Helper()
	path := filepath.Join(test.TempDir(), "sequence.yaml")
	sequenceWrite(test, path, hyperlapseSource(test))
	plan, err := readSequence(path)
	if err != nil {
		test.Fatal(err)
	}
	return plan
}

// hyperlapseSource is small and does not require any survey data or rendering.
func hyperlapseSource(test *testing.T) []byte {
	test.Helper()
	raw, err := os.ReadFile("testdata/camera-hyperlapse.yaml")
	if err != nil {
		test.Fatal(err)
	}
	return raw
}

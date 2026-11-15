package job

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// TestAStillHasOneFrameAtItsTime holds a still to one frame, at its at.
func TestAStillHasOneFrameAtItsTime(t *testing.T) {
	times := read(t, still+"at: {value: 20, unit: s}\n").Times()
	if fmt.Sprint(times) != "[20s]" {
		t.Errorf("a still at 20s is at %v", times)
	}
	if times := read(t, still).Times(); fmt.Sprint(times) != "[0s]" {
		t.Errorf("a still that says no at is at %v", times)
	}
}

// TestAPresetsRateDoesNotMakeAStillAFilm is the anti-test: a preset with
// a rate on a job that says no span leaves it a still.
// anti-test of TestAStillHasOneFrameAtItsTime
func TestAPresetsRateDoesNotMakeAStillAFilm(t *testing.T) {
	clock := read(t, still+"film: {preset: video-preview}\n").Clock()
	if _, rated := clock.Rate(); !clock.Still() || rated ||
		clock.Frames() != one || clock.Duration() != nothing {
		t.Error("a preset's rate made a still a film")
	}
}

// TestAFilmsFramesAreItsRateApart holds frame i at start plus i over the
// rate, and the count at the duration times the rate.
func TestAFilmsFramesAreItsRateApart(t *testing.T) {
	job := read(t, film+"start: {value: 40, unit: s}\n")
	times := job.Times()
	got := fmt.Sprint(len(times), times[nothing], times[one],
		times[len(times)-one])
	if want := "48 40s 40.041666666s 41.958333333s"; got != want {
		t.Errorf("a two second film at 24 from 40s is %s, want %s", got,
			want)
	}
}

// TestAFilmIsNeverFewerThanOneFrame is the near miss of the count: a
// duration shorter than a frame still draws one.
// anti-test of TestAFilmsFramesAreItsRateApart
// anti-test of TestTimesAreExactForAnyFilm
func TestAFilmIsNeverFewerThanOneFrame(t *testing.T) {
	job := read(t, still+"fps: 1\nduration: {value: 0.001, unit: s}\n")
	if frames := job.Clock().Frames(); frames != one {
		t.Errorf("a film shorter than its frame has %d frames", frames)
	}
}

// TestAStretchedFilmSpansItsSceneDuration holds the hyperlapse's hour in
// thirty seconds: frame 90 of 180 at 30m45s, as the sequence plan has it.
func TestAStretchedFilmSpansItsSceneDuration(t *testing.T) {
	job := read(t, still+"fps: 6\nduration: {value: 30, unit: s}\n"+
		"start: {value: 45, unit: s}\n"+
		"scene-duration: {value: 3600, unit: s}\n")
	times := job.Times()
	got := fmt.Sprint(len(times), times[len(times)/halves])
	if want := "180 30m45s"; got != want {
		t.Errorf("an hour in thirty seconds is %s, want %s", got, want)
	}
}

// schemaLeast and schemaMost are the schema's bounds on the clock: the
// least and most frames a second, the most seconds of start or scene
// time, and the least and most seconds of a film.
const (
	leastRate    = 1
	mostRate     = 120
	mostInstant  = 86400
	leastSpan    = 0.001
	mostSpan     = 3600
	nanoseconds  = int64(time.Second)
	stretchOdds  = 2
	stretchEvery = 1
)

// exactTime is frame index's time worked in whole numbers with no
// rounding: start plus the floor of index over rate seconds, or of
// index times the span over the frame count.
func exactTime(start, span time.Duration, rate, frames,
	index int) time.Duration {
	offset := big.NewInt(int64(index))
	if span == nothing {
		offset.Mul(offset, big.NewInt(nanoseconds))
		offset.Quo(offset, big.NewInt(int64(rate)))
	} else {
		offset.Mul(offset, big.NewInt(int64(span)))
		offset.Quo(offset, big.NewInt(int64(frames)))
	}
	return start + time.Duration(offset.Int64())
}

// TestTimesAreExactForAnyFilm is the property: for any film the schema
// takes, frame i is where whole-number arithmetic puts it.
func TestTimesAreExactForAnyFilm(t *testing.T) {
	most := int(number(t, "most-frames"))
	rapid.Check(t, func(trial *rapid.T) {
		rate := rapid.IntRange(leastRate, mostRate).Draw(trial, "fps")
		playing := rapid.Float64Range(leastSpan, mostSpan).Draw(trial,
			"duration")
		begun := rapid.Float64Range(nothing, mostInstant).Draw(trial,
			"start")
		clock := Clock{rate: rate, playing: playing,
			start: &quantity{Value: begun, Unit: seconds}}
		if rapid.IntRange(nothing, stretchOdds).Draw(trial,
			"stretch") == stretchEvery {
			clock.stretch = &quantity{Value: rapid.Float64Range(
				leastSpan, mostInstant).Draw(trial, "scene"),
				Unit: seconds}
		}
		frames := min(clock.Frames(), most)
		index := rapid.IntRange(nothing, frames-one).Draw(trial, "index")
		start, _ := clock.Start()
		span, _ := clock.SceneDuration()
		want := exactTime(start, span, rate, clock.Frames(), index)
		if got := clock.frameTime(index); got != want {
			trial.Fatalf("frame %d at %s, want %s", index, got, want)
		}
	})
}

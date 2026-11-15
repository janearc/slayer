package cuesheet

import (
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/janearc/slayer/beats"
)

// measured is a supplied audio grid as a clock, trimmed from the file.
// It owns its grid so an editor cannot mutate a running film's clock.
type measured struct {
	grid *beats.Grid
	from time.Duration
}

// Measured makes a clock from a checked audio grid, which may still be
// explicitly estimated. From is the audio trim, not scene time. Bar
// numbers remain the original song's; trimming never renumbers the score.
func Measured(grid *beats.Grid, from time.Duration) (Clock, error) {
	if err := grid.Check(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrClock, err)
	}
	if from < 0 || from.Seconds() >= grid.Score.Duration {
		return nil, fmt.Errorf("%w: trim outside the audio grid", ErrClock)
	}
	copy := *grid
	copy.Score.Beats.Times = slices.Clone(grid.Score.Beats.Times)
	copy.Score.Beats.Measured = slices.Clone(grid.Score.Beats.Measured)
	return measured{grid: &copy, from: from}, nil
}

// Meter is the authored beats per bar.
func (clock measured) Meter() int { return clock.grid.Score.Meter }

// Bars is the number of bars touched by the grid, including its last.
func (clock measured) Bars() int {
	count := len(clock.grid.Score.Beats.Times)
	return (count + clock.Meter() - 1) / clock.Meter()
}

// Beats is the continuous original beat count at a playback moment.
func (clock measured) Beats(at time.Duration) float64 {
	return clock.grid.BeatAt((at + clock.from).Seconds())
}

// Moment is when a song beat falls relative to the trimmed audio start.
func (clock measured) Moment(beat float64) time.Duration {
	return seconds(clock.grid.SecondAt(beat)) - clock.from
}

// Notes provides a beat voice for flashes, not invented instrument notes.
// Each pulse lasts at most a frame at 60 fps, an authored cue convention.
func (clock measured) Notes(voice string) ([]Note, error) {
	if voice != "beat" {
		return nil, fmt.Errorf("%w: audio grid has only beat", ErrVoice)
	}
	var notes []Note
	for index, at := range clock.grid.Score.Beats.Times {
		on := seconds(at) - clock.from
		if on < 0 {
			continue
		}
		length := math.Min(1.0/60, clock.grid.Score.Duration-at)
		notes = append(notes, Note{On: on, Off: on + seconds(length),
			Loud: 1, Hit: fmt.Sprint(index%clock.Meter() + 1)})
	}
	return notes, nil
}

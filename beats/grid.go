// Package beats keeps an audio track's editable beat grid. Detected onsets
// suggest a pulse; they do not establish a song's metre or its downbeat.
package beats

import (
	"fmt"
	"math"
	"os"
	"sort"

	"github.com/janearc/puzzlemouth/yamlfile"
)

// Grid is a Moonlake-compatible score. Times are seconds from the audio
// file's start, before any film trim or hyperlapse time mapping.
type Grid struct {
	Score Score `yaml:"score" json:"score"`
}

// Score describes the track and the status of its rhythm reading.
// Status is estimated until a person has checked the visual metronome.
type Score struct {
	Audio    string  `yaml:"audio" json:"audio"`
	SHA256   string  `yaml:"sha256" json:"sha256"`
	Duration float64 `yaml:"duration" json:"duration"`
	Meter    int     `yaml:"meter" json:"meter"`
	Status   string  `yaml:"status" json:"status"`
	Method   string  `yaml:"method" json:"method"`
	Analysis *Input  `yaml:"analysis,omitempty" json:"analysis,omitempty"`
	Beats    Pulse   `yaml:"beats" json:"beats"`
}

// Input identifies an aligned analysis stem without replacing the original
// playback source. Separation is an estimate, never a recovered original.
type Input struct {
	File   string `yaml:"file" json:"file"`
	SHA256 string `yaml:"sha256" json:"sha256"`
}

// Pulse is the grid. Measured marks local onset support (one) versus
// counted-through beats (zero), not human approval of a musical beat.
type Pulse struct {
	Times    []float64 `yaml:"times" json:"times"`
	Measured []int     `yaml:"measured" json:"measured"`
}

// Read opens a checked YAML or JSON grid through the common reader.
func Read(path string) (*Grid, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var grid Grid
	if err := yamlfile.Read(path, raw, &grid); err != nil {
		return nil, err
	}
	return &grid, grid.Check()
}

// Check refuses empty, non-finite, repeated or reversed grids before a
// renderer sees them. An estimate must never silently become verified.
func (grid *Grid) Check() error {
	if grid == nil {
		return fmt.Errorf("beats: no grid")
	}
	score := grid.Score
	if !finite(score.Duration) || score.Duration <= 0 || score.Meter < 1 {
		return fmt.Errorf("beats: positive duration and meter required")
	}
	if score.Status != "estimated" && score.Status != "verified" {
		return fmt.Errorf("beats: status is estimated or verified")
	}
	if len(score.Beats.Times) < 2 ||
		len(score.Beats.Measured) != len(score.Beats.Times) {
		return fmt.Errorf("beats: two or more beats with matching marks required")
	}
	previous := -1.0
	for index, at := range score.Beats.Times {
		mark := score.Beats.Measured[index]
		if !finite(at) || at < 0 || at <= previous || at >= score.Duration ||
			(mark != 0 && mark != 1) {
			return fmt.Errorf("beats: invalid beat %d", index)
		}
		previous = at
	}
	return nil
}

// finite says a number can be used as a time.
func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

// BeatAt interpolates a continuous beat count, zero at the first beat.
// Outside the measured interval it continues the nearest beat spacing.
func (grid *Grid) BeatAt(seconds float64) float64 {
	times := grid.Score.Beats.Times
	index := sort.SearchFloat64s(times, seconds) - 1
	index = max(0, min(index, len(times)-2))
	return float64(index) + (seconds-times[index])/
		(times[index+1]-times[index])
}

// SecondAt is BeatAt's inverse, including its endpoint continuation.
func (grid *Grid) SecondAt(beat float64) float64 {
	times := grid.Score.Beats.Times
	index := max(0, min(int(math.Floor(beat)), len(times)-2))
	return times[index] + (beat-float64(index))*(times[index+1]-times[index])
}

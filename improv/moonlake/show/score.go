package show

import (
	"fmt"
	"math"
	"os"
	"sort"

	yaml "gopkg.in/yaml.v3"
)

// refusal is an error that is a constant, so nothing can assign it.
type refusal string

// Error is the refusal's words.
func (reason refusal) Error() string { return string(reason) }

// errShow is wrapped by every error the package returns.
const errShow refusal = "show"

// written is score.yaml as it is on disk.
type written struct {
	Score struct {
		Status   string            `yaml:"status"`
		Meter    int               `yaml:"meter"`
		Duration float64           `yaml:"duration"`
		Tracks   map[string][]note `yaml:"tracks"`
		Beats    struct {
			Times    []float64 `yaml:"times"`
			Measured []int     `yaml:"measured"`
		} `yaml:"beats"`
	} `yaml:"score"`
}

// note is a transcribed note as score.yaml writes it: its start, in the
// file's seconds, under either name, how long it sounds and how hard it
// is struck.
type note struct {
	Sec   float64 `yaml:"sec"`
	AtSec float64 `yaml:"at_sec"`
	Dur   float64 `yaml:"dur"`
	Vel   int     `yaml:"vel"`
}

// scoreNote is a transcribed note: when it starts and ends, in the song's
// seconds, and how hard it is struck, 0 to 127.
type scoreNote struct {
	start, length float64
	vel           int
}

// Score is the song as measured: its length, the beat grid, which beats
// were measured, and the transcribed tracks.
type Score struct {
	status   string
	meter    int
	duration float64
	beats    []float64
	measured []bool
	tracks   map[string][]scoreNote
}

// ReadScore reads score.yaml.
func ReadScore(path string) (*Score, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errShow, err)
	}
	return parseScore(raw)
}

// parseScore reads a score from its yaml. A score that names no meter
// is in four. A meter under one, a grid of under two beats, or one that
// does not rise, is refused by name.
func parseScore(raw []byte) (*Score, error) {
	var on written
	if err := yaml.Unmarshal(raw, &on); err != nil {
		return nil, fmt.Errorf("%w: the score: %w", errShow, err)
	}
	times := on.Score.Beats.Times
	if on.Score.Meter == 0 {
		on.Score.Meter = unsaidMeter
	}
	if on.Score.Meter < 1 {
		return nil, fmt.Errorf("%w: the score's meter is %d; a bar needs "+
			"one beat or more", errShow, on.Score.Meter)
	}
	if len(times) < 2 {
		return nil, fmt.Errorf("%w: the score has %d beats; a grid needs "+
			"two or more", errShow, len(times))
	}
	if !sort.Float64sAreSorted(times) {
		return nil, fmt.Errorf("%w: the score's beat times do not rise; "+
			"sort score.beats.times", errShow)
	}
	return newScore(on), nil
}

// newScore is a checked score.
func newScore(on written) *Score {
	made := &Score{status: on.Score.Status, meter: on.Score.Meter,
		duration: on.Score.Duration,
		beats:    on.Score.Beats.Times,
		tracks:   map[string][]scoreNote{}}
	made.measured = make([]bool, len(made.beats))
	for i, said := range on.Score.Beats.Measured {
		if i < len(made.measured) {
			made.measured[i] = said == 1
		}
	}
	for name, notes := range on.Score.Tracks {
		for _, one := range notes {
			made.tracks[name] = append(made.tracks[name], scoreNote{
				start: math.Max(one.Sec, one.AtSec), length: one.Dur,
				vel: one.Vel})
		}
	}
	return made
}

// Duration is the song's length in seconds.
func (of *Score) Duration() float64 { return of.duration }

// Estimated says this score still needs its visual and musical check.
func (of *Score) Estimated() bool { return of.status == "estimated" }

// Meter is the beats to a bar: the score's own, or four for a score
// that names none.
func (of *Score) Meter() int { return of.meter }

// Beats is every beat's time, in seconds, a copy.
func (of *Score) Beats() []float64 { return append([]float64(nil), of.beats...) }

// BarTime is the time of a bar and beat on the grid, both from one, or
// an error naming the bar when the score lacks it.
func (of *Score) BarTime(bar, beat int) (float64, error) {
	meter := of.meter
	index := (bar-1)*meter + beat - 1
	if bar < 1 || beat < 1 || beat > meter || index >= len(of.beats) {
		return 0, fmt.Errorf("%w: bar %d beat %d is off the grid: the score "+
			"has bars 1 to %d, beats 1 to %d", errShow, bar, beat,
			(len(of.beats)+meter-1)/meter, meter)
	}
	return of.beats[index], nil
}

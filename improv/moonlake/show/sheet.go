package show

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// cue is one line of the sheet: when, and only the fields it names. The
// fields are cues.yaml's, whose header says what each does.
type cue struct {
	At             string   `yaml:"at"`
	AtSec          *float64 `yaml:"at_sec"`
	Rig            *float64 `yaml:"rig"`
	Beams          *int     `yaml:"beams"`
	Palette        []string `yaml:"palette"`
	SweepBars      *float64 `yaml:"sweep_bars"`
	Strobes        *string  `yaml:"strobes"`
	Duty           *float64 `yaml:"duty"`
	Hot            *int     `yaml:"hot"`
	Hotter         *int     `yaml:"hotter"`
	Mist           *float64 `yaml:"mist"`
	Pulse          *string  `yaml:"pulse"`
	Decay          *float64 `yaml:"decay"`
	Split          *bool    `yaml:"split"`
	Floods         *string  `yaml:"floods"`
	Spots          *float64 `yaml:"spots"`
	SpotBars       *float64 `yaml:"spot_bars"`
	Search         *float64 `yaml:"search"`
	SearchBars     *float64 `yaml:"search_bars"`
	Master         *float64 `yaml:"master"`
	Overhead       *float64 `yaml:"overhead"`
	Exposure       *float64 `yaml:"exposure"`
	Fire           *float64 `yaml:"fire"`
	Moon           *float64 `yaml:"moon"`
	MoonAlt        *float64 `yaml:"moon_alt"`
	MoonPulse      *float64 `yaml:"moon_pulse"`
	Figures        []string `yaml:"figures"`
	FigureBars     *float64 `yaml:"figure_bars"`
	Banks          []string `yaml:"banks"`
	BanksBars      *float64 `yaml:"banks_bars"`
	SearchMoves    []string `yaml:"search_moves"`
	SearchMoveBars *float64 `yaml:"search_move_bars"`
	SearchBeat     *float64 `yaml:"search_beat"`
	Fade           *float64 `yaml:"fade"`
}

// placed is a cue with its moment found on the grid, in seconds, and its
// place on the grid, in beats.
type placed struct {
	line     cue
	at, grid float64
}

// Sheet is the cue sheet read against a score: every cue placed at its
// moment, in the sheet's order.
type Sheet struct {
	score *Score
	cues  []placed
}

// ReadSheet reads cues.yaml against a score.
func ReadSheet(path string, score *Score) (*Sheet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrShow, err)
	}
	return ParseSheet(raw, score)
}

// ParseSheet reads a cue sheet from its yaml against a score. A cue at a
// bar the score lacks, or naming a figure, bank or move lightshow does
// not play, is refused by name.
func ParseSheet(raw []byte, score *Score) (*Sheet, error) {
	if score == nil {
		return nil, fmt.Errorf("%w: a sheet is read against a score; "+
			"read the score first", ErrShow)
	}
	var on struct {
		Cues []cue `yaml:"cues"`
	}
	if err := yaml.Unmarshal(raw, &on); err != nil {
		return nil, fmt.Errorf("%w: the cue sheet: %w", ErrShow, err)
	}
	made := &Sheet{score: score}
	for _, line := range on.Cues {
		at, err := score.Moment(line.At, line.AtSec)
		if err != nil {
			return nil, err
		}
		if err := line.playable(at); err != nil {
			return nil, err
		}
		made.cues = append(made.cues, placed{line: line, at: at,
			grid: gridAt(at, score.beats)})
	}
	return made, nil
}

// Moment is the time of a place written as cues.yaml writes one: at_sec
// as it is, or "bar.beat" (a bare bar is its first beat) on the grid.
func (of *Score) Moment(at string, atSec *float64) (float64, error) {
	if atSec != nil {
		return *atSec, nil
	}
	fields := strings.SplitN(at, ".", 2)
	bar, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, fmt.Errorf("%w: at %q is not bar.beat: %w", ErrShow, at, err)
	}
	beat := 1
	if len(fields) == 2 {
		if beat, err = strconv.Atoi(fields[1]); err != nil {
			return 0, fmt.Errorf("%w: at %q is not bar.beat: %w", ErrShow,
				at, err)
		}
	}
	seconds, err := of.BarTime(bar, beat)
	if err != nil {
		return 0, fmt.Errorf("at %q: %w", at, err)
	}
	return seconds, nil
}

// playable refuses a cue naming a figure, bank or move lightshow does
// not play, saying which and what to choose.
func (of cue) playable(at float64) error {
	for _, each := range []struct {
		what         string
		list, choose []string
	}{
		{"figure", of.Figures, figureNames},
		{"banks", of.Banks, bankNames},
		{"move", of.SearchMoves, moveNames},
	} {
		for _, name := range each.list {
			if !slices.Contains(each.choose, name) {
				return fmt.Errorf("%w: cue at %v s: no %s %q; choose %v",
					ErrShow, at, each.what, name, each.choose)
			}
		}
	}
	return nil
}

// Score is the score the sheet was read against.
func (of *Sheet) Score() *Score { return of.score }

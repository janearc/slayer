package show

import (
	"fmt"
	"strings"
)

// Fired is a cue at its moment: when it fires, what the sheet calls the
// moment, the words for what it sets, and which of the rig's parts it
// sets.
type Fired struct {
	at    float64
	label string
	words []string
	kill  bool
	sets  map[string]bool
}

// At is when the cue fires, in seconds.
func (of Fired) At() float64 { return of.at }

// Words is the cue as the metronome's band shows it: its moment, then
// what it sets.
func (of Fired) Words() string {
	return of.label + "  " + strings.Join(of.words, "  ")
}

// Kill is whether the cue kills the rig: it sets the rig to nought.
func (of Fired) Kill() bool { return of.kill }

// Sets is whether the cue sets one of the rig's parts: Lasers, Strobes,
// Floods, Search, Moon, Overhead or Master.
func (of Fired) Sets(part string) bool { return of.sets[part] }

// the rig's parts a cue can set, by the name Sets takes.
const (
	PartLasers   = "lasers"
	PartStrobes  = "strobes"
	PartFloods   = "floods"
	PartSearch   = "search"
	PartMoon     = "moon"
	PartOverhead = "overhead"
	partMaster   = "master"
)

// parts is which of a cue's fields set which part of the rig.
var parts = map[string]func(cue) bool{
	PartLasers: func(c cue) bool {
		return c.Rig != nil || c.Beams != nil || c.Pulse != nil ||
			len(c.Figures) > 0
	},
	PartStrobes:  func(c cue) bool { return c.Strobes != nil },
	PartFloods:   func(c cue) bool { return c.Floods != nil },
	PartSearch:   func(c cue) bool { return c.Search != nil },
	PartMoon:     func(c cue) bool { return c.Moon != nil || c.MoonAlt != nil },
	PartOverhead: func(c cue) bool { return c.Overhead != nil },
	partMaster:   func(c cue) bool { return c.Master != nil },
}

// Fired is every cue at its moment, in the sheet's order.
func (of *Sheet) Fired() []Fired {
	all := make([]Fired, 0, len(of.cues))
	for _, each := range of.cues {
		all = append(all, fired(each))
	}
	return all
}

// fired is one placed cue as a Fired.
func fired(each placed) Fired {
	line := each.line
	made := Fired{at: each.at, label: line.At, sets: map[string]bool{},
		words: said(line)}
	if line.AtSec != nil {
		made.label = fmt.Sprintf("%.1fs", *line.AtSec)
	}
	made.kill = line.Rig != nil && *line.Rig == 0
	for part, sets := range parts {
		made.sets[part] = sets(line)
	}
	return made
}

// said is the words for what a cue sets, as the metronome shows them.
func said(line cue) []string {
	var words []string
	if line.Rig != nil {
		if *line.Rig == 0 {
			words = append(words, "KILL")
		} else {
			words = append(words, fmt.Sprintf("RIG %.2g", *line.Rig))
		}
	}
	if line.Beams != nil {
		words = append(words, fmt.Sprintf("%d BEAMS", *line.Beams))
	}
	if line.Strobes != nil {
		words = append(words, "STROBES "+strings.ToUpper(*line.Strobes))
	}
	if line.SweepBars != nil {
		words = append(words, fmt.Sprintf("SWEEP %g", *line.SweepBars))
	}
	if len(line.Palette) > 0 {
		words = append(words, strings.ToUpper(strings.Join(line.Palette, " ")))
	}
	if line.Mist != nil {
		words = append(words, fmt.Sprintf("MIST %.2g", *line.Mist))
	}
	return words
}

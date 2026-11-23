package cheeptune

import (
	"fmt"
	"sort"
)

// Loudness marks and tempo changes: the two things a score says about
// how it is played and not only what is played.
//
// A loudness mark is an at sign and a word on the end of a note, a chord
// or a hit, after its length and before its tie: c5:1/2@p, (c4 e4):2@ff,
// k@f. A token with no mark is mf, which is how every score written
// before the marks existed already plays.
//
// The studio turns the note up or down by the mark's gain, and an analog
// channel's filter may be set to open further for a loud note (its
// accent). The Game Boy renders a marked score exactly
// as an unmarked one: its tests pin its output, and its voicings already
// sit at the top of the sixteen volume steps it has, where louder is not
// there to give.
//
// A tempo change is a bar and a tempo; the piece takes that tempo from
// the start of that bar, at once. There is no gradual change: a slide is
// written as several small steps. Everything that turns beats into
// seconds asks SecondsAt, and everything that turns seconds into beats
// asks BeatsAt, so the notes, the playhead, the roll's grid and a cut
// sheet's seconds agree.

// marks are the loudness marks and the gain each gives. Picked by the
// writer of the studio, 2026-11-12, as steps in decibels around mf: pp is
// twelve down, p eight, mp four, f two up and ff four.
var marks = map[string]float64{
	"pp": 0.25, "p": 0.4, "mp": 0.63, "mf": 1, "f": 1.26, "ff": 1.58,
}

// markNames is the marks from quiet to loud, for an error.
var markNames = []string{"pp", "p", "mp", "mf", "f", "ff"}

// tempoChange is a new tempo from the start of a bar, counted from one.
type tempoChange struct {
	Bar   int     `json:"bar"`
	Tempo float64 `json:"tempo"`
}

// tempoSpan is a stretch at one tempo: the beat and the second it starts
// at, and how long a beat lasts in it.
type tempoSpan struct {
	beat, second, beatSeconds float64
}

// The range of a tempo, in beats a minute. Defined by the score's own
// rule for its first tempo.
const (
	tempoLeast = 40
	tempoMost  = 300
)

// checkTempos checks the tempo changes and builds the map from them.
func (s *Score) checkTempos() []error {
	var errs []error
	s.spans = nil
	if len(s.Tempos) == 0 || s.Tempo <= 0 || s.Meter < 1 {
		return nil
	}
	span := tempoSpan{beatSeconds: 60 / s.Tempo}
	spans := []tempoSpan{span}
	last := 1
	for i, c := range s.Tempos {
		at := fmt.Sprintf("tempo change %d", i+1)
		switch {
		case c.Tempo < tempoLeast || c.Tempo > tempoMost:
			errs = append(errs, fmt.Errorf("%s: tempo %v is outside %d "+
				"to %d", at, c.Tempo, tempoLeast, tempoMost))
			continue
		case c.Bar <= last:
			errs = append(errs, fmt.Errorf("%s: bar %d is not after bar "+
				"%d; changes are in order and the first tempo is bar "+
				"1's", at, c.Bar, last))
			continue
		case c.Bar > s.Bars():
			errs = append(errs, fmt.Errorf("%s: bar %d is not in the "+
				"score, which has bars 1 to %d", at, c.Bar, s.Bars()))
			continue
		}
		beat := float64((c.Bar - 1) * s.Meter)
		span = tempoSpan{beat: beat, beatSeconds: 60 / c.Tempo,
			second: span.second + (beat-span.beat)*span.beatSeconds}
		spans = append(spans, span)
		last = c.Bar
	}
	if len(errs) == 0 {
		s.spans = spans
	}
	return errs
}

// SecondsAt is the second a beat falls at, counted from the start. Past
// the end of a looping piece it goes round again.
func (s *Score) SecondsAt(beats float64) float64 {
	if len(s.spans) == 0 {
		return beats * s.beatSeconds()
	}
	whole := float64(s.Bars() * s.Meter)
	if s.Loop && beats >= whole {
		return s.spanAt(whole).secondOf(whole) + s.SecondsAt(beats-whole)
	}
	return s.spanAt(beats).secondOf(beats)
}

// BeatsAt is the beat a second falls at, counted from the start. Past
// the end of a looping piece it goes round again.
func (s *Score) BeatsAt(seconds float64) float64 {
	if len(s.spans) == 0 {
		return seconds / s.beatSeconds()
	}
	whole := float64(s.Bars() * s.Meter)
	length := s.spanAt(whole).secondOf(whole)
	if s.Loop && seconds >= length {
		return whole + s.BeatsAt(seconds-length)
	}
	at := sort.Search(len(s.spans), func(i int) bool {
		return s.spans[i].second > seconds
	})
	span := s.spans[max(at-1, 0)]
	return span.beat + (seconds-span.second)/span.beatSeconds
}

// spanAt is the stretch a beat falls in.
func (s *Score) spanAt(beats float64) tempoSpan {
	at := sort.Search(len(s.spans), func(i int) bool {
		return s.spans[i].beat > beats
	})
	return s.spans[max(at-1, 0)]
}

// secondOf is the second a beat falls at, by this stretch's tempo.
func (t tempoSpan) secondOf(beats float64) float64 {
	return t.second + (beats-t.beat)*t.beatSeconds
}

// tempoSpans is the piece as stretches of one tempo each, in seconds:
// where each starts and how long a beat lasts in it. A piece that keeps
// its tempo is one stretch.
func (s *Score) tempoSpans() []tempoSpan {
	if len(s.spans) == 0 {
		return []tempoSpan{{beatSeconds: s.beatSeconds()}}
	}
	return s.spans
}

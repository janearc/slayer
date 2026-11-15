package cuesheet

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/janearc/slayer/cheeptune"
)

// Note is one note a voice plays: when it starts and when it lets go, and
// the drum it strikes, when it is a hit.
type Note struct {
	On, Off time.Duration
	Hit     string
	// Loud is how loud the note is written, as a gain: 1 is mf.
	Loud float64
}

// Clock is where a moment falls in a piece of music.
type Clock interface {
	// Meter is how many beats a bar holds.
	Meter() int
	// Bars is how many bars the music holds.
	Bars() int
	// Beats is how many beats from the first a moment is.
	Beats(at time.Duration) float64
	// Moment is when a count of beats from the first falls.
	Moment(beats float64) time.Duration
	// Notes are the notes a voice plays, in order, or a refusal naming
	// the voices there are.
	Notes(voice string) ([]Note, error)
}

// ErrClock is a clock that cannot be made as asked.
var ErrClock = errors.New("cuesheet: no such clock")

// ErrVoice is a voice the music has not.
var ErrVoice = errors.New("cuesheet: no such voice")

// fixed is a plain tempo: so many beats a minute, so many to a bar, so
// many bars, and the notes each voice plays.
type fixed struct {
	beat   float64 // seconds
	meter  int
	bars   int
	voices map[string][]Note
}

// the counts a clock is made of, each defined
const (
	none      = 0                // defined: no beats, no bars
	aMinute   = time.Minute      // defined: a tempo is beats a minute
	fromFirst = 1                // defined: bars and beats count from one
	sixteenth = 16               // defined: moonlake's pulse is sixteen steps
	untold    = -math.MaxFloat64 // defined: before anything happened
)

// Fixed is a clock at a fixed tempo, beats a minute, meter beats to a bar
// and bars bars long, whose voices play the notes given. A tempo, meter or
// length of nought or less is refused.
func Fixed(tempo float64, meter, bars int,
	voices map[string][]Note) (Clock, error) {
	if !(tempo > none) || meter <= none || bars <= none {
		return nil, fmt.Errorf("%w: a fixed tempo of %v beats a minute, "+
			"%d to a bar, %d bars; each must be above nought", ErrClock,
			tempo, meter, bars)
	}
	held := map[string][]Note{}
	for name, notes := range voices {
		held[name] = sorted(notes)
	}
	return fixed{beat: aMinute.Seconds() / tempo, meter: meter,
		bars: bars, voices: held}, nil
}

// sorted is the notes in the order they start, a copy.
func sorted(notes []Note) []Note {
	held := append([]Note(nil), notes...)
	sort.SliceStable(held, func(i, k int) bool {
		return held[i].On < held[k].On
	})
	return held
}

// Meter is the beats to a bar.
func (clock fixed) Meter() int { return clock.meter }

// Bars is the bars the music holds.
func (clock fixed) Bars() int { return clock.bars }

// Beats is the beats from the first at a moment.
func (clock fixed) Beats(at time.Duration) float64 {
	return at.Seconds() / clock.beat
}

// Moment is when a count of beats falls.
func (clock fixed) Moment(beats float64) time.Duration {
	return seconds(beats * clock.beat)
}

// Notes are a voice's notes.
func (clock fixed) Notes(voice string) ([]Note, error) {
	notes, found := clock.voices[voice]
	if !found {
		return nil, refusedVoice(voice, clock.voices)
	}
	return notes, nil
}

// refusedVoice is the refusal of a voice the music has not, listing
// those it has.
func refusedVoice(voice string, voices map[string][]Note) error {
	names := make([]string, 0, len(voices))
	for name := range voices {
		names = append(names, name)
	}
	sort.Strings(names)
	return fmt.Errorf("%w: %q; the voices are %s", ErrVoice, voice,
		strings.Join(names, ", "))
}

// Piece is a cheeptune piece as a clock, its beats where its score's
// tempo changes put them. Its voices are its score's
// roles, each with the notes the piece plays for it (a role the voicing
// drops has none); a drum role's notes carry the hit each strikes.
func Piece(music *cheeptune.Piece) (Clock, error) {
	score := music.Score
	voices := map[string][]Note{}
	for _, part := range score.Parts {
		voices[part.Role] = []Note{}
		for _, one := range music.Notes(part.Role) {
			voices[part.Role] = append(voices[part.Role], Note{
				On:  seconds(one.On),
				Off: seconds(one.Off), Hit: one.Hit, Loud: one.Loud})
		}
	}
	clock, err := Fixed(score.Tempo, score.Meter, score.Bars(), voices)
	if err != nil {
		return nil, err
	}
	return scored{fixed: clock.(fixed), score: score}, nil
}

// scored is a cheeptune piece's clock: its voices and bars as a fixed
// clock's, and its beats where its score's tempo changes put them.
type scored struct {
	fixed
	score *cheeptune.Score
}

// Beats is the beats from the first at a moment, by the score's tempos.
func (clock scored) Beats(at time.Duration) float64 {
	return clock.score.BeatsAt(at.Seconds())
}

// Moment is when a count of beats falls, by the score's tempos.
func (clock scored) Moment(beats float64) time.Duration {
	return seconds(clock.score.SecondsAt(beats))
}

// seconds is a count of seconds as a duration.
func seconds(count float64) time.Duration {
	return time.Duration(count * float64(time.Second))
}

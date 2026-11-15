package cuesheet_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/cuesheet"
)

// A cue at a second holds from that second, as moonlake's at_sec does,
// and a later cue at the same channel takes over from its own moment.
func TestACueAtASecondHolds(t *testing.T) {
	read := sheet(t, `cues:
  - {at_sec: 1.5, star: 2}
  - {at: "2.1", star: 3}`)
	is(t, read, "star", time.Second, 1)
	is(t, read, "star", 1500*time.Millisecond, 2)
	is(t, read, "star", bar, 3)
}

// The sheet gives back the channels it was read against.
func TestTheSheetKeepsItsChannels(t *testing.T) {
	read := sheet(t, `cues: []`)
	if got := read.Channels(); len(got) != len(channels) ||
		got[0] != channels[0] {
		t.Errorf("the sheet's channels are %v", got)
	}
	is(t, read, "beam", bar, 6)
}

// anti-test of TestACueAtASecondHolds
// Cues written wrong are refused by name: a moment before
// the music, a ramp that ends before it starts, a flash with no strike,
// a pulse of other marks, a decay of nothing, a follow with nothing up,
// a quiet that is not a list, a cue that is not a map, a number that is
// not one, and a sheet with no clock.
func TestCuesWrittenWrongAreRefused(t *testing.T) {
	for _, wrong := range []struct {
		cues string
		is   error
	}{
		{`cues: [{at_sec: -1, star: 2}]`, cuesheet.ErrWhen},
		{`cues: [{at: "3.1", until: "2.1", star: 2}]`, cuesheet.ErrWhen},
		{`cues: [{at: "1.1", flash: {star: 2}, decay: 1}]`, cuesheet.ErrCue},
		{`cues: [{at: "1.1", flash: {star: 2}, pulse: "x-x-", decay: 1}]`,
			cuesheet.ErrCue},
		{`cues: [{at: "1.1", flash: {star: 2}, pulse: x, decay: 0}]`,
			cuesheet.ErrCue},
		{`cues: [{at: "1.1", flash: [star], pulse: x, decay: 1}]`,
			cuesheet.ErrCue},
		{`cues: [{at: "1.1", follow: lead}]`, cuesheet.ErrCue},
		{`cues: [{at: "1.1", quiet: star}]`, cuesheet.ErrCue},
		{`cues: [{at: "1.1", quiet: [stars]}]`, cuesheet.ErrChannel},
		{`cues: [star]`, cuesheet.ErrCue},
		{`cues: [{at: "1.1", star: bright}]`, cuesheet.ErrCue},
		{`cues: [{at: "one", star: 2}]`, cuesheet.ErrCue},
		{`cues: [{at: "1.1", fade: soon, star: 2}]`, cuesheet.ErrCue},
		{`cues: {at: 1}`, cuesheet.ErrCue},
	} {
		_, err := cuesheet.Read([]byte(wrong.cues), channels, clock(t))
		if !errors.Is(err, wrong.is) {
			t.Errorf("%s: %v; want %v", wrong.cues, err, wrong.is)
		}
	}
	if _, err := cuesheet.Read([]byte(`cues: []`), channels,
		nil); !errors.Is(err, cuesheet.ErrClock) {
		t.Errorf("no clock: %v; want ErrClock", err)
	}
}

// A step moves its channel on at each strike and it stays there: the
// drum's kicks at the first beat of bars one and two move the star a
// half each, and a quiet stops the steps, the star staying where they
// left it.
func TestAStepCountsItsStrikes(t *testing.T) {
	read := sheet(t, `cues:
  - {at: "1.1", step: {star: 0.5}, on: drums, hit: k}
  - {at: "3.1", quiet: [star]}`)
	is(t, read, "star", beat, 1.5)
	is(t, read, "star", bar+beat, 2)
	is(t, read, "star", 3*bar, 2)
}

// anti-test of TestAStepCountsItsStrikes
// A step of a channel the scene has not, or by no number, is
// refused by name; a step with nothing to strike it is refused.
func TestAStepWrittenWrongIsRefused(t *testing.T) {
	for _, wrong := range []struct {
		cues string
		is   error
	}{
		{`cues: [{at: "1.1", step: {stars: 1}, on: drums}]`,
			cuesheet.ErrChannel},
		{`cues: [{at: "1.1", step: {star: up}, on: drums}]`, cuesheet.ErrCue},
		{`cues: [{at: "1.1", step: [star], on: drums}]`, cuesheet.ErrCue},
		{`cues: [{at: "1.1", step: {star: 1}}]`, cuesheet.ErrCue},
	} {
		_, err := cuesheet.Read([]byte(wrong.cues), channels, clock(t))
		if !errors.Is(err, wrong.is) {
			t.Errorf("%s: %v; want %v", wrong.cues, err, wrong.is)
		}
	}
}

// A follow with an attack and a release breathes: the lead's note from
// beat one to beat two swells the beam over half a beat, holds, and lets
// it fall over a whole beat after the note ends.
func TestAFollowBreathes(t *testing.T) {
	read := sheet(t, `cues: [{at: "1.1", follow: lead, up: {beam: 16},
	  attack: 0.5, release: 1}]`)
	is(t, read, "beam", beat, 6)
	is(t, read, "beam", beat+beat/4, 11)
	is(t, read, "beam", beat+beat*3/4, 16)
	is(t, read, "beam", 2*beat+beat/2, 11)
	is(t, read, "beam", 3*beat+beat/2, 6)
}

// anti-test of TestAFollowBreathes
// An attack or a release of fewer than nought beats, or not
// a number, is refused.
func TestABreathWrittenWrongIsRefused(t *testing.T) {
	for _, cues := range []string{
		`cues: [{at: "1.1", follow: lead, up: {beam: 9}, attack: -1}]`,
		`cues: [{at: "1.1", follow: lead, up: {beam: 9}, release: soon}]`,
	} {
		_, err := cuesheet.Read([]byte(cues), channels, clock(t))
		if !errors.Is(err, cuesheet.ErrCue) {
			t.Errorf("%s: %v; want ErrCue", cues, err)
		}
	}
}

// hot keeps only notes written at least so loud, and soft only those
// quieter: of a loud kick and a quiet one, a flash on hot strikes the
// loud alone and a step on soft counts the quiet alone.
func TestHotAndSoftSortNotesByLoudness(t *testing.T) {
	made, err := cuesheet.Fixed(tempo, meter, bars,
		map[string][]cuesheet.Note{"drums": {
			{On: 0, Off: struck, Hit: "k", Loud: 1.58},
			{On: bar, Off: bar + struck, Hit: "k", Loud: 0.4}}})
	if err != nil {
		t.Fatal(err)
	}
	read, err := cuesheet.Read([]byte(`cues:
  - {at: "1.1", flash: {star: 4}, on: drums, hot: 1.5, decay: 0.25}
  - {at: "1.1", step: {beam: 1}, on: drums, soft: 1}`), channels, made)
	if err != nil {
		t.Fatal(err)
	}
	is(t, read, "star", 0, 4)
	is(t, read, "star", bar, 1+3*math.Exp(-16))
	is(t, read, "beam", 0, 6)
	is(t, read, "beam", bar, 7)
}

// anti-test of TestHotAndSoftSortNotesByLoudness
// A loudness that is not a number is refused.
func TestALoudnessWrittenWrongIsRefused(t *testing.T) {
	_, err := cuesheet.Read([]byte(`cues: [{at: "1.1", flash: {star: 2},
	  on: drums, hot: loud, decay: 1}]`), channels, clock(t))
	if !errors.Is(err, cuesheet.ErrCue) {
		t.Errorf("hot: loud: %v; want ErrCue", err)
	}
}

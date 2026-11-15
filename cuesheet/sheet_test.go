package cuesheet_test

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/cuesheet"
)

// the music the tests play: 120 beats a minute, so half a second a beat,
// four to a bar, eight bars; a drum striking kicks on the first beat of
// the first two bars, a quarter of a second each, and a lead holding one
// note through the second beat. Picked by the author of this package,
// 2026-11-12, for arithmetic that reads at a glance.
const (
	tempo  = 120
	meter  = 4
	bars   = 8
	beat   = 500 * time.Millisecond
	bar    = meter * beat
	struck = 250 * time.Millisecond
	near   = 1e-9
)

// the channels the tests play: a star from nought to four resting at one,
// and a beam from one to twenty resting at six
var channels = []cuesheet.Channel{
	{Name: "star", Least: 0, Most: 4, Rest: 1},
	{Name: "beam", Least: 1, Most: 20, Rest: 6},
}

// clock is the tests' music, or the test stops.
func clock(t *testing.T) cuesheet.Clock {
	t.Helper()
	made, err := cuesheet.Fixed(tempo, meter, bars, map[string][]cuesheet.Note{
		"drums": {{On: bar, Off: bar + struck, Hit: "k"},
			{On: 0, Off: struck, Hit: "k"},
			{On: 2 * beat, Off: 2*beat + struck, Hit: "s"}},
		"lead": {{On: beat, Off: 2 * beat}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// sheet is the cues read against the tests' music, or the test stops.
func sheet(t *testing.T, cues string) *cuesheet.Sheet {
	t.Helper()
	read, err := cuesheet.Read([]byte(cues), channels, clock(t))
	if err != nil {
		t.Fatal(err)
	}
	return read
}

// is fails the test unless the channel is the value at the moment.
func is(t *testing.T, read *cuesheet.Sheet, name string, at time.Duration,
	want float64) {
	t.Helper()
	if got := read.At(at)[name]; math.Abs(got-want) > near {
		t.Errorf("%s at %v is %v, want %v", name, at, got, want)
	}
}

// A level holds from its bar: the rest before it, its value from it on.
func TestALevelHoldsFromItsBar(t *testing.T) {
	read := sheet(t, `cues: [{at: "3.1", star: 2.5}]`)
	is(t, read, "star", 2*bar-time.Millisecond, 1)
	is(t, read, "star", 2*bar, 2.5)
	is(t, read, "star", 7*bar, 2.5)
}

// anti-test of TestALevelHoldsFromItsBar
// A level for one channel leaves the others at rest.
func TestALevelLeavesOtherChannelsAlone(t *testing.T) {
	read := sheet(t, `cues: [{at: "3.1", star: 2.5}]`)
	is(t, read, "beam", 3*bar, 6)
}

// A ramp is linear between its bars and held outside them.
func TestARampIsLinearBetweenItsBars(t *testing.T) {
	read := sheet(t, `cues: [{at: "2.1", until: "4.1", star: 3}]`)
	is(t, read, "star", bar-time.Millisecond, 1)
	is(t, read, "star", bar, 1)
	is(t, read, "star", 2*bar, 2)
	is(t, read, "star", 3*bar, 3)
	is(t, read, "star", 5*bar, 3)
}

// A fade glides over its seconds, as moonlake's does.
func TestAFadeGlidesOverItsSeconds(t *testing.T) {
	read := sheet(t, `cues: [{at: "2.1", fade: 2, star: 3}]`)
	is(t, read, "star", bar+time.Second, 2)
	is(t, read, "star", bar+2*time.Second, 3)
}

// A flash peaks on its strike and has decayed by its stated beats: to a
// third (1/e) of the way at one decay.
func TestAFlashPeaksOnTheBeatAndDecays(t *testing.T) {
	read := sheet(t, `cues: [{at: "1.1", flash: {star: 4},
	  pulse: "x...............", decay: 1}]`)
	is(t, read, "star", 0, 4)
	is(t, read, "star", beat, 1+3*math.Exp(-1))
	is(t, read, "star", bar, 4)
}

// anti-test of TestAFlashPeaksOnTheBeatAndDecays
// Before its first strike a flash leaves its channel at its
// level, and a quiet ends it.
func TestAFlashBeforeItsStrikeAndAfterAQuiet(t *testing.T) {
	read := sheet(t, `cues:
  - {at: "2.2", flash: {star: 4}, on: drums, hit: k, decay: 1}
  - {at: "3.1", quiet: [star]}`)
	is(t, read, "star", bar+beat, 1)
	is(t, read, "star", bar+bar/2, 1)
	is(t, read, "star", 2*bar+beat, 1)
}

// A flash on a voice strikes on its notes, only its hit's when it says.
func TestAFlashOnAVoiceStrikesOnItsNotes(t *testing.T) {
	read := sheet(t, `cues: [{at: "1.1", flash: {star: 4}, on: drums,
	  hit: k, decay: 0.5}]`)
	is(t, read, "star", bar, 4)
	is(t, read, "star", 2*beat, 1+3*math.Exp(-4))
}

// A follow is up while a note of its voice sounds and down between.
func TestAFollowIsUpWhileANoteSounds(t *testing.T) {
	read := sheet(t, `cues: [{at: "1.1", follow: lead, up: {beam: 15}}]`)
	is(t, read, "beam", beat/2, 6)
	is(t, read, "beam", beat+beat/2, 15)
	is(t, read, "beam", 2*beat+beat/2, 6)
}

// anti-test of TestALevelHoldsFromItsBar
// anti-test of TestAFollowIsUpWhileANoteSounds
// What a sheet cannot play is refused by name: a channel the
// scene has not, listing those it has; a value out of range, with the
// range; a bar past the music's end; a beat past the bar's; a voice the
// music has not.
func TestWhatASheetCannotPlayIsRefused(t *testing.T) {
	for _, wrong := range []struct {
		cues, says string
		is         error
	}{
		{`cues: [{at: "1.1", stars: 2}]`, "beam, star", cuesheet.ErrChannel},
		{`cues: [{at: "1.1", star: 4.5}]`, "0 to 4", cuesheet.ErrRange},
		{`cues: [{at: "1.1", flash: {beam: 0.5}, pulse: x, decay: 1}]`,
			"1 to 20", cuesheet.ErrRange},
		{`cues: [{at: "9.1", star: 2}]`, "bars 1 to 8", cuesheet.ErrWhen},
		{`cues: [{at: "2.5", star: 2}]`, "4 beats", cuesheet.ErrWhen},
		{`cues: [{at: "1.1", follow: bass, up: {beam: 2}}]`, "drums, lead",
			cuesheet.ErrVoice},
		{`cues: [{at: "1.1", flash: {star: 2}, pulse: x}]`, "decay",
			cuesheet.ErrCue},
		{`cues: [{star: 2}]`, "at or at_sec", cuesheet.ErrCue},
	} {
		_, err := cuesheet.Read([]byte(wrong.cues), channels, clock(t))
		if !errors.Is(err, wrong.is) ||
			!strings.Contains(err.Error(), wrong.says) {
			t.Errorf("%s: %v; want %v saying %q", wrong.cues, err,
				wrong.is, wrong.says)
		}
	}
}

// The fixed clock is arithmetic: half a second a beat at 120.
func TestTheFixedClockIsArithmetic(t *testing.T) {
	made := clock(t)
	if made.Beats(3*time.Second) != 6 || made.Moment(3) != 1500*
		time.Millisecond || made.Meter() != meter || made.Bars() != bars {
		t.Errorf("at 120 a minute: 3s is %v beats, 3 beats %v",
			made.Beats(3*time.Second), made.Moment(3))
	}
}

// anti-test of TestTheFixedClockIsArithmetic
// A fixed clock of no tempo, meter or length is refused.
func TestAFixedClockOfNothingIsRefused(t *testing.T) {
	for _, wrong := range [][3]float64{{0, 4, 8}, {120, 0, 8},
		{120, 4, 0}} {
		_, err := cuesheet.Fixed(wrong[0], int(wrong[1]), int(wrong[2]),
			nil)
		if !errors.Is(err, cuesheet.ErrClock) {
			t.Errorf("%v: %v; want ErrClock", wrong, err)
		}
	}
}

package cuesheet_test

import (
	"testing"
	"time"

	"github.com/janearc/slayer/cuesheet"
)

// anti-test of TestTheSheetKeepsItsChannels
// Changing the channels the sheet gave back does not change the sheet.
func TestTheSheetsChannelsAreACopy(t *testing.T) {
	read := sheet(t, `cues: []`)
	got := read.Channels()
	got[0].Name = "changed"
	if again := read.Channels(); again[0].Name != channels[0].Name {
		t.Errorf("the sheet's first channel is now %q", again[0].Name)
	}
}

// anti-test of TestACueAtASecondHolds
// Cues listed out of order play in order of their moments: the later
// moment written first does not win, and a second is not a bar.
func TestCuesOutOfOrderPlayInOrderOfTime(t *testing.T) {
	read := sheet(t, `cues:
  - {at: "2.1", star: 3}
  - {at_sec: 1.5, star: 2}`)
	is(t, read, "star", 1500*time.Millisecond, 2)
	is(t, read, "star", bar-time.Millisecond, 2)
	is(t, read, "star", bar, 3)
}

// anti-test of TestAStepCountsItsStrikes
// A step on the snare counts the snare and not the kicks.
func TestAStepOnAnotherHitCountsOtherStrikes(t *testing.T) {
	read := sheet(t, `cues: [{at: "1.1", step: {star: 0.5}, on: drums, hit: s}]`)
	is(t, read, "star", beat, 1)
	is(t, read, "star", 2*beat, 1.5)
	is(t, read, "star", bar+2*beat, 1.5)
}

// anti-test of TestAFollowBreathes
// A follow with no attack or release does not breathe: it is up or down
// at the note's own edges.
func TestAFollowWithoutBreathJumps(t *testing.T) {
	read := sheet(t, `cues: [{at: "1.1", follow: lead, up: {beam: 16}}]`)
	is(t, read, "beam", beat-time.Millisecond, 6)
	is(t, read, "beam", beat, 16)
	is(t, read, "beam", 2*beat-time.Millisecond, 16)
	is(t, read, "beam", 2*beat, 6)
}

// anti-test of TestHotAndSoftSortNotesByLoudness
// A bar set above every note strikes none for hot and a bar set below
// every note counts none for soft.
func TestHotAndSoftPastEveryNoteStrikeNothing(t *testing.T) {
	made, err := cuesheet.Fixed(tempo, meter, bars,
		map[string][]cuesheet.Note{"drums": {
			{On: 0, Off: struck, Hit: "k", Loud: 1.58},
			{On: bar, Off: bar + struck, Hit: "k", Loud: 0.4}}})
	if err != nil {
		t.Fatal(err)
	}
	read, err := cuesheet.Read([]byte(`cues:
  - {at: "1.1", flash: {star: 4}, on: drums, hot: 2, decay: 0.25}
  - {at: "1.1", step: {beam: 1}, on: drums, soft: 0.3}`), channels, made)
	if err != nil {
		t.Fatal(err)
	}
	is(t, read, "star", 0, 1)
	is(t, read, "star", bar, 1)
	is(t, read, "beam", 0, 6)
	is(t, read, "beam", 3*bar, 6)
}

// anti-test of TestARampIsLinearBetweenItsBars
// A ramp is not yet at its end just before it: one millisecond short of
// the end it has not arrived, and it is not a step at its start.
func TestARampHasNotArrivedBeforeItsEnd(t *testing.T) {
	read := sheet(t, `cues: [{at: "2.1", until: "4.1", star: 3}]`)
	if got := read.At(3*bar - time.Millisecond)["star"]; got >= 3 {
		t.Errorf("a millisecond short of its end the ramp is at %v", got)
	}
	if got := read.At(bar + time.Millisecond)["star"]; got > 1.01 {
		t.Errorf("a millisecond after its start the ramp is at %v", got)
	}
}

// anti-test of TestAFadeGlidesOverItsSeconds
// A level with no fade jumps at once; a fade has not arrived before its
// seconds are up.
func TestALevelWithNoFadeJumpsAtOnce(t *testing.T) {
	jumps := sheet(t, `cues: [{at: "2.1", star: 3}]`)
	is(t, jumps, "star", bar, 3)
	fades := sheet(t, `cues: [{at: "2.1", fade: 2, star: 3}]`)
	is(t, fades, "star", bar, 1)
	if got := fades.At(bar + 2*time.Second - time.Millisecond)["star"]; got >= 3 {
		t.Errorf("a millisecond short of its end the fade is at %v", got)
	}
}

// anti-test of TestAFlashOnAVoiceStrikesOnItsNotes
// A flash that names the snare does not strike on the kicks.
func TestAFlashOnAnotherHitLeavesTheKicksQuiet(t *testing.T) {
	read := sheet(t, `cues: [{at: "1.1", flash: {star: 4}, on: drums,
	  hit: s, decay: 0.5}]`)
	is(t, read, "star", 0, 1)
	is(t, read, "star", beat, 1)
	is(t, read, "star", 2*beat, 4)
}

// anti-test of TestAFollowIsUpWhileANoteSounds
// A follow of the drums is not up while the lead sounds.
func TestAFollowOfAnotherVoiceStaysDown(t *testing.T) {
	read := sheet(t, `cues: [{at: "1.1", follow: drums, hit: k, up: {beam: 15}}]`)
	is(t, read, "beam", beat+beat/2, 6)
	is(t, read, "beam", beat/4, 15)
}

// anti-test of TestTheFixedClockIsArithmetic
// At another tempo the same moments are other beats.
func TestAFixedClockAtAnotherTempoCountsOtherBeats(t *testing.T) {
	made, err := cuesheet.Fixed(60, meter, bars, nil)
	if err != nil {
		t.Fatal(err)
	}
	if made.Beats(3*time.Second) != 3 || made.Moment(3) != 3*time.Second {
		t.Errorf("at 60 a minute: 3s is %v beats, 3 beats %v",
			made.Beats(3*time.Second), made.Moment(3))
	}
}

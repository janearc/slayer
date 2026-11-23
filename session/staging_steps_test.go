package session

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// The counts and times the staging steps tests are held to.
const (
	stepsAt      = 3                // defined: a thing partway
	stepsOf      = 9                // defined: the things a set has
	stepsElapsed = 62 * time.Second // defined: a minute and two seconds
)

// a staging told to stop gives the stop's error at its next step,
// naming the step and the thing of how many, and saying nothing is kept.
func TestStagingStoppedSaysWhereAndKeepsNothing(t *testing.T) {
	stop, cancel := context.WithCancel(context.Background())
	cancel()
	made := newSteps(stop, &bytes.Buffer{}, time.Now)
	err := made.step("things", stepsAt, stepsOf)
	if !errors.Is(err, context.Canceled) ||
		!strings.Contains(err.Error(), "things 3 of 9") ||
		!strings.Contains(err.Error(), "nothing staged is kept") {
		t.Errorf("a stopped staging gave %v", err)
	}
}

// anti-test of TestStagingStoppedSaysWhereAndKeepsNothing: a staging not
// told to stop, and nil steps, go on.
func TestStagingNotStoppedGoesOn(t *testing.T) {
	made := newSteps(context.Background(), &bytes.Buffer{}, time.Now)
	if err := made.step("terrain", nothing, nothing); err != nil {
		t.Errorf("a staging not stopped gave %v", err)
	}
	var none *steps
	if err := none.step("terrain", nothing, nothing); err != nil {
		t.Errorf("nil steps gave %v", err)
	}
	none.begin()
	none.end()
}

// the line says the step, the thing of how many, and the time taken.
func TestStagingLineSaysWhereAndHowLong(t *testing.T) {
	began := time.Now()
	made := newSteps(context.Background(), &bytes.Buffer{},
		func() time.Time { return began.Add(stepsElapsed) })
	made.began = began
	_ = made.step("things", stepsAt, stepsOf)
	want := "staging at=things 3/9 elapsed=1m2s"
	if got := made.line(); got != want {
		t.Errorf("the line is %q, not %q", got, want)
	}
	_ = made.step("atmosphere", nothing, nothing)
	if got := made.line(); !strings.HasPrefix(got,
		"staging at=atmosphere elapsed=") {
		t.Errorf("an uncounted step's line is %q", got)
	}
}

// anti-test of TestStagingLineSaysWhereAndHowLong: steps begun and ended
// twice, or ended unbegun, stop cleanly and write nothing more.
func TestStagingEndedTwiceIsQuiet(t *testing.T) {
	said := &bytes.Buffer{}
	made := newSteps(context.Background(), said, time.Now)
	made.end()
	made.begin()
	made.begin()
	made.end()
	made.end()
	if said.Len() != nothing {
		t.Errorf("steps ended before an interval said %q", said.String())
	}
}

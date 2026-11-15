package progress_test

import (
	"testing"
	"time"

	"github.com/janearc/slayer/render/progress"
)

// the instant the throttle tests count from. It is picked: any would do.
var origin = time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

// a line is due once the interval has passed, and not before.
func TestALineIsDueAfterTheInterval(t *testing.T) {
	for what, row := range map[string]struct {
		since time.Duration
		due   bool
	}{
		"just after":      {1, false},
		"just before":     {progress.Interval - 1, false},
		"on the interval": {progress.Interval, true},
		"long after":      {progress.Interval * 9, true},
	} {
		if got := progress.Due(origin, origin.Add(row.since), false); got != row.due {
			t.Errorf("%s: due %v, want %v", what, got, row.due)
		}
	}
}

// the line that reports a frame's end is due however soon it follows.
// anti-test beside it: the same instant, not a frame's end, is not due.
func TestAFrameEndIsAlwaysDue(t *testing.T) {
	if !progress.Due(origin, origin, true) {
		t.Error("a frame's end, at once after a line, was held back")
	}
	if progress.Due(origin, origin, false) {
		t.Error("a line at once after a line was not held back")
	}
}

// the first line is due, and so is one after the clock has gone back.
func TestTheFirstLineAndAClockGoneBackAreDue(t *testing.T) {
	if !progress.Due(time.Time{}, origin, false) {
		t.Error("the first line was held back")
	}
	if !progress.Due(origin, origin.Add(-time.Second), false) {
		t.Error("a line after the clock went back was held back")
	}
}

package progress_test

import (
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/render/progress"
)

// a job in the shape of the worked example: 12 s in, 11 of 96 frames done
// at 1 s each, the twelfth with rows told. each count is picked, from the
// example in the design, 2026-11-12.
func inFlight(t *testing.T) *progress.Meter {
	t.Helper()
	meter, clock := begun(t, 96, 22)
	for range 11 {
		clock.pass(time.Second)
		mustNil(t, meter.FrameDone())
	}
	clock.pass(time.Second)
	mustNil(t, meter.Rows(10))
	return meter
}

// the plain line, whole: 96 frames of 22 rows is 2112 rows; 11 frames and
// 10 rows is 252; 252 / 2112 = 0.119 cut to 0.11. 12 s elapsed. the mean is
// 1 s a frame, the twelfth has had 1 s, so 84 s are left, 1m24s.
func TestThePlainLineIsStableWords(t *testing.T) {
	const want = "progress frame=12/96 rows=252/2112 share=0.11 " +
		"elapsed=12s remaining~1m24s"
	if got := progress.Plain(inFlight(t)); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

// the human line, whole: the same job, 11%, 84 s is 1 min 24 s.
func TestTheHumanLineIsShort(t *testing.T) {
	const want = "frame 12 of 96, 11%, about 1 min 24 s left"
	if got := progress.Human(inFlight(t)); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

// under a minute is in seconds; a finished job says done, and the plain
// line says it is at share 1.00 and nothing remains.
func TestTheLinesOfAShortAndAFinishedJob(t *testing.T) {
	meter, clock := begun(t, 4, progress.MinRows)
	clock.pass(10 * time.Second)
	mustNil(t, meter.FrameDone())
	if got, want := progress.Human(meter), "frame 2 of 4, 25%, about 30 s left"; got != want {
		t.Errorf("got %q want %q", got, want)
	}
	for range 3 {
		clock.pass(time.Second)
		mustNil(t, meter.FrameDone())
	}
	if got, want := progress.Human(meter), "frame 4 of 4, 100%, done"; got != want {
		t.Errorf("got %q want %q", got, want)
	}
	const plain = "progress frame=4/4 rows=32/32 share=1.00 elapsed=13s remaining~0s"
	if got := progress.Plain(meter); got != plain {
		t.Errorf("got %q want %q", got, plain)
	}
}

// anti-test: while no time is offered neither line says one, and a meter
// that has begun nothing says nought of nought. no line has an escape byte,
// a carriage return or a newline, and a share just under one does not
// round up to it.
func TestTheLinesSayNothingTheyDoNotKnow(t *testing.T) {
	cold, _ := progress.NewMeter(time.Now)
	plain, human := progress.Plain(cold), progress.Human(cold)
	if plain != "progress frame=0/0 rows=0/0 share=0.00 elapsed=0s" {
		t.Errorf("cold plain line %q", plain)
	}
	if human != "frame 0 of 0, 0%" {
		t.Errorf("cold human line %q", human)
	}
	meter, _ := begun(t, 1, 1000)
	mustNil(t, meter.Rows(999))
	for _, line := range []string{progress.Plain(meter), progress.Human(meter), plain, human} {
		if strings.ContainsAny(line, "\x1b\r\n") {
			t.Errorf("control bytes in %q", line)
		}
	}
	if got := progress.Plain(meter); !strings.Contains(got, "share=0.99") ||
		strings.Contains(got, "remaining") {
		t.Errorf("999 of 1000 rows: %q", got)
	}
	if got := progress.Human(meter); got != "frame 1 of 1, 99%" {
		t.Errorf("999 of 1000 rows: %q", got)
	}
}

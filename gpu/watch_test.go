package gpu

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// a progress line says the rows finished of all, nothing taken up, the
// time taken, and the time left at the pace so far once it is offered:
// 247 of 720 rows in 30 s leave about 57 s.
func TestAProgressLineIsTheProcessorsLine(t *testing.T) {
	began := time.Unix(0, 0)
	state := watching{began: began, done: 247, all: 720}
	got := state.line(began.Add(30 * time.Second))
	want := "progress rows=247/720 taken=0 elapsed=30s remaining~57s"
	if got != want {
		t.Errorf("the line is %q; want %q", got, want)
	}
}

// before a pace can be said, the line says no time left.
func TestAProgressLineOffersNoTimeTooSoon(t *testing.T) {
	began := time.Unix(0, 0)
	state := watching{began: began, done: 3, all: 720}
	got := state.line(began.Add(30 * time.Second))
	if strings.Contains(got, "remaining") {
		t.Errorf("after three rows the line is %q; want no time left", got)
	}
}

// a watched drawing stops once its context is done, saying the rows it
// finished and wrapping the context's own error; not before.
func TestAWatchedDrawingStopsOnItsContext(t *testing.T) {
	stop, cancel := context.WithCancel(context.Background())
	Watched(NewWatch(stop, nil))
	defer Unwatched()
	watched.rows(5, 9)
	if err := watched.stopped(); err != nil {
		t.Fatalf("a drawing whose context is live stopped: %v", err)
	}
	cancel()
	err := watched.stopped()
	if !errors.Is(err, context.Canceled) ||
		!strings.Contains(err.Error(), "5 of 9 rows") {
		t.Errorf("the stop is %v; want the rows and context.Canceled", err)
	}
}

// with no watch kept, a drawing stops on nothing, even after a watch
// whose context is done was let go.
func TestAnUnwatchedDrawingNeverStops(t *testing.T) {
	stop, cancel := context.WithCancel(context.Background())
	cancel()
	Watched(NewWatch(stop, nil))
	Unwatched()
	if err := watched.stopped(); err != nil {
		t.Errorf("a drawing with no watch stopped: %v", err)
	}
}

// letting a watch go that was kept a second or more says its last line
// where its lines go.
func TestLettingAWatchGoSaysTheLastLine(t *testing.T) {
	var lines bytes.Buffer
	watch := NewWatch(context.Background(), &lines)
	Watched(watch)
	watched.rows(2, 4)
	watched.mutex.Lock()
	watched.began = watched.began.Add(-quietUnder)
	watched.mutex.Unlock()
	Unwatched()
	if !strings.HasPrefix(lines.String(), "progress rows=2/4 taken=0") {
		t.Errorf("the last line is %q; want rows 2 of 4", lines.String())
	}
}

// a watch let go in under a second says nothing: a film's quick frames
// would each say a line.
func TestAQuickFrameSaysNoLastLine(t *testing.T) {
	var lines bytes.Buffer
	watch := NewWatch(context.Background(), &lines)
	Watched(watch)
	watched.rows(4, 4)
	Unwatched()
	if lines.Len() != 0 {
		t.Errorf("a frame let go at once said %q; want nothing",
			lines.String())
	}
}

package session

// Staging a set is done in steps: its terrain, its light, each of its
// things in turn, its air, and so on. Between two steps nothing is half
// made, so that is where staging looks at its context and stops; a stop
// gives an error and the session is never made, so nothing staged is
// kept. While it stages, a line at a steady interval says which step it
// is at, and which thing of how many where it is placing things.

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

// stagingEvery is the time between two lines saying where staging is.
// Picked: the same quarter minute as a still's rows, enough to see
// staging move without filling a log over a long one.
const stagingEvery = 15 * time.Second

// steps is where a staging is and what stops it: the step it is at, the
// thing of how many where it counts them, and when it began.
type steps struct {
	stop  context.Context
	out   io.Writer
	every time.Duration
	now   func() time.Time
	mutex sync.Mutex
	// what is the step; at of of the place in it, of nothing where the
	// step is not counted
	what        string
	at, of      int
	began       time.Time
	quit, ended chan struct{}
}

// newSteps is the steps of a staging that stops when stop is done and
// says where it is on out each interval.
func newSteps(stop context.Context, out io.Writer,
	now func() time.Time) *steps {
	return &steps{stop: stop, out: out, every: stagingEvery, now: now,
		what: "the set"}
}

// begin starts the lines. Nil steps do nothing.
func (made *steps) begin() {
	if made == nil {
		return
	}
	made.mutex.Lock()
	defer made.mutex.Unlock()
	if made.quit != nil {
		return
	}
	made.began = made.now()
	made.quit, made.ended = make(chan struct{}), make(chan struct{})
	go made.telling(made.quit, made.ended)
}

// step is staging come to step what, at at of of where it is counted (of
// nothing where not), and the stop's error, naming the step, where the
// staging is told to stop. Nil steps never stop.
func (made *steps) step(what string, at, of int) error {
	if made == nil {
		return nil
	}
	made.mutex.Lock()
	made.what, made.at, made.of = what, at, of
	made.mutex.Unlock()
	if made.stop == nil {
		return nil
	}
	stopped := made.stop.Err()
	if stopped == nil {
		return nil
	}
	return fmt.Errorf("hillside: staging stopped at %s; nothing staged "+
		"is kept: %w", made.where(), stopped)
}

// where is the step, and the count where it has one: "things 12 of 400".
func (made *steps) where() string {
	made.mutex.Lock()
	defer made.mutex.Unlock()
	if made.of == nothing {
		return made.what
	}
	return fmt.Sprintf("%s %d of %d", made.what, made.at, made.of)
}

// telling writes a line each interval until quit is closed, and closes
// ended once it writes no more.
func (made *steps) telling(quit <-chan struct{}, ended chan<- struct{}) {
	defer close(ended)
	ticker := time.NewTicker(made.every)
	defer ticker.Stop()
	for {
		select {
		case <-quit:
			return
		case <-ticker.C:
			_, _ = fmt.Fprintln(made.out, made.line())
		}
	}
}

// line is where staging is, as words a progress line writes:
//
//	staging at=things 12/400 elapsed=1m2s
func (made *steps) line() string {
	made.mutex.Lock()
	defer made.mutex.Unlock()
	elapsed := max(made.now().Sub(made.began), time.Duration(nothing))
	at := made.what
	if made.of > nothing {
		at = fmt.Sprintf("%s %d/%d", made.what, made.at, made.of)
	}
	return fmt.Sprintf("staging at=%s elapsed=%s", at,
		elapsed.Round(time.Second))
}

// end stops the lines. Nil steps, or steps never begun, do nothing.
func (made *steps) end() {
	if made == nil {
		return
	}
	made.mutex.Lock()
	quit, ended := made.quit, made.ended
	made.quit = nil
	made.mutex.Unlock()
	if quit == nil {
		return
	}
	close(quit)
	<-ended
}

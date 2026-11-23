package gpu

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/janearc/slayer/render/progress"
)

// A GPU drawing can be stopped, and says how far it is. A trace's
// samples are finished a band of rows at a time; after each band the
// drawing looks at the watch's context and stops if it is done, leaving
// at most the band drawn ahead to be waited out. While a watch is kept,
// a line on its writer every progress.StillEvery says the rows finished
// of the picture under way, in progress.StillLine's words.
//
// A band that is already on the device is not waited for: the wait for
// it ends the moment the context is done, with the context's error, and
// the device is left the band. The drawing's kernel is not used again.
//
// The watch is the package's for as long as a caller keeps it: the
// drawings it covers are the trace's many entry points, and none takes a
// context of its own.

// quietUnder is how long a watch must have been kept for its last line
// to be said when it is let go: a second, picked, so a frame drawn
// quicker, as a film's or a live view's often is, says nothing.
const quietUnder = time.Second

// Watch is what the GPU drawings of a stretch of work stop on, and where
// they say their progress.
type Watch struct {
	stop context.Context
	out  io.Writer
}

// NewWatch is a watch that stops on stop and says progress on out; a
// nil out says nothing.
func NewWatch(stop context.Context, out io.Writer) Watch {
	return Watch{stop: stop, out: out}
}

// watching is the watch kept now, the rows of the picture under way,
// and the progress lines' ticker.
type watching struct {
	mutex     sync.Mutex
	watch     Watch
	began     time.Time
	done, all int
	quit      chan struct{}
	ended     chan struct{}
	// release lets the device's waits go from the watch's context; see
	// stopDevice
	release func()
}

// watched is the package's watch; its zero stops on nothing and says
// nothing.
var watched watching

// Watched keeps watch until Unwatched: every GPU drawing stops on its
// context, and its writer is told the rows finished every
// progress.StillEvery. A watch already kept is let go first.
func Watched(watch Watch) {
	Unwatched()
	watched.mutex.Lock()
	defer watched.mutex.Unlock()
	watched.watch, watched.began = watch, time.Now()
	watched.done, watched.all = 0, 0
	watched.release = stopDevice(watch.stop)
	if watch.out == nil {
		return
	}
	watched.quit, watched.ended = make(chan struct{}), make(chan struct{})
	go watched.telling(watched.quit, watched.ended)
}

// Unwatched lets the watch go: drawings stop on nothing, and the lines
// end, the last one said where the watch was kept quietUnder or longer.
func Unwatched() {
	watched.mutex.Lock()
	quit, ended := watched.quit, watched.ended
	watched.quit, watched.ended = nil, nil
	release := watched.release
	watched.release = nil
	kept := time.Since(watched.began)
	watched.mutex.Unlock()
	if release != nil {
		release()
	}
	if quit != nil {
		close(quit)
		<-ended
		if kept >= quietUnder {
			watched.say()
		}
	}
	watched.mutex.Lock()
	watched.watch = Watch{}
	watched.mutex.Unlock()
}

// rows is told that done rows of a picture all rows tall are finished.
func (state *watching) rows(done, all int) {
	state.mutex.Lock()
	state.done, state.all = done, all
	state.mutex.Unlock()
}

// stopped is the context's error once the watch's context is done, with
// the rows finished; nil while it is not, or with no watch.
func (state *watching) stopped() error {
	state.mutex.Lock()
	defer state.mutex.Unlock()
	stop := state.watch.stop
	if stop == nil || stop.Err() == nil {
		return nil
	}
	return fmt.Errorf("gpu: stopped with %d of %d rows finished: %w",
		state.done, state.all, stop.Err())
}

// telling says a progress line each interval until quit is closed, and
// closes ended once it says no more.
func (state *watching) telling(quit <-chan struct{}, ended chan<- struct{}) {
	defer close(ended)
	ticker := time.NewTicker(progress.StillEvery)
	defer ticker.Stop()
	for {
		select {
		case <-quit:
			return
		case <-ticker.C:
			state.say()
		}
	}
}

// say writes one progress line where the watch's lines go; a line that
// cannot be written is let go, since it only says how far a drawing is.
func (state *watching) say() {
	state.mutex.Lock()
	out, line := state.watch.out, state.line(time.Now())
	state.mutex.Unlock()
	if out != nil {
		_, _ = fmt.Fprintln(out, line)
	}
}

// line is the progress at now: the GPU takes up no rows.
func (state *watching) line(now time.Time) string {
	return progress.StillLine(state.done, state.all, 0, now.Sub(state.began))
}

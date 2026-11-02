package terminal_test

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/janearc/slayer/interfaces/terminal"
	"github.com/janearc/slayer/interfaces/terminal/mocks"
	"go.uber.org/mock/gomock"
)

// streamOf is the stream of asked over a source that expects the frames
// of the shape given and no others, or the test stops.
func streamOf(t *testing.T, asked shape, frames shape) *terminal.Stream {
	t.Helper()
	source := mocks.NewMockSource(gomock.NewController(t))
	expectFrames(source, frames)
	stream, err := terminal.NewStream(sizeOf(t, asked), asked.first,
		asked.count, source)
	if err != nil {
		t.Fatal(err)
	}
	return stream
}

// asks counts the times a stream asked whether to go on, and answers
// with the stop and the error it was made with.
type asks struct {
	times int
	stop  bool
	err   error
}

// next is what RunBy is given.
func (waits *asks) next() (bool, error) {
	waits.times++
	return waits.stop, waits.err
}

// Told to go on, RunBy writes the bytes Run writes. It asks once after
// each frame, the last included.
func TestAStreamToldToGoOnWritesWhatRunWritesAndAsksBetweenFrames(t *testing.T) {
	asked := standard()
	want := drawn(asked)
	var out bytes.Buffer
	waits := &asks{}
	if err := streamOf(t, asked, asked).RunBy(&out, waits.next); err != nil {
		t.Fatal(err)
	}
	if out.String() != want {
		t.Errorf("RunBy wrote %d bytes, want the %d Run writes", out.Len(),
			len(want))
	}
	if waits.times != asked.count {
		t.Errorf("asked %d times for %d frames; want one each", waits.times,
			asked.count)
	}
}

// anti-test: a stream told to stop at its first frame draws that frame
// and no other. The source is asked for the first only, which the mock
// holds it to.
func TestAStreamToldToStopDrawsNoMore(t *testing.T) {
	asked := standard()
	first := asked
	first.count = once
	want := drawn(first)
	var out bytes.Buffer
	waits := &asks{stop: true}
	if err := streamOf(t, asked, first).RunBy(&out, waits.next); err != nil {
		t.Fatalf("a stop is not an error: %v", err)
	}
	if out.String() != want || waits.times != once {
		t.Errorf("wrote %d bytes after %d asks; want the first frame's %d "+
			"after one", out.Len(), waits.times, len(want))
	}
}

// anti-test: a stream whose way of asking fails stops there, with that
// error and the number of the frame it was waiting after. Nothing more is
// drawn.
func TestAStreamStopsAndSaysWhenItCannotAsk(t *testing.T) {
	asked := standard()
	first := asked
	first.count = once
	want := drawn(first)
	gone := errors.New("the keyboard went")
	waits := &asks{stop: true, err: gone}
	var out bytes.Buffer
	err := streamOf(t, asked, first).RunBy(&out, waits.next)
	if !errors.Is(err, gone) {
		t.Fatalf("got %v; want the keyboard's error", err)
	}
	says := fmt.Sprintf("frame %d: waiting for a key", asked.first)
	if !strings.Contains(err.Error(), says) {
		t.Errorf("got %q; want it to say %q", err, says)
	}
	if out.String() != want {
		t.Errorf("wrote %d bytes; want the first frame's %d", out.Len(),
			len(want))
	}
}

// The last frame is asked about. Told to go on, the stream ends anyway.
func TestTheLastFrameIsAskedAboutAndThenTheStreamEnds(t *testing.T) {
	asked := standard()
	asked.count = once
	waits := &asks{}
	if err := streamOf(t, asked, asked).RunBy(&bytes.Buffer{},
		waits.next); err != nil || waits.times != once {
		t.Errorf("one frame: %v after %d asks; want one", err, waits.times)
	}
}

// anti-test: a way of asking that fails at the last frame ends the stream
// with its error, as it does at any other.
func TestAnAskThatFailsAtTheLastFrameIsAnError(t *testing.T) {
	asked := standard()
	asked.count = once
	gone := errors.New("the keyboard went")
	waits := &asks{stop: true, err: gone}
	if err := streamOf(t, asked, asked).RunBy(&bytes.Buffer{},
		waits.next); !errors.Is(err, gone) {
		t.Errorf("got %v; want the keyboard's error", err)
	}
}

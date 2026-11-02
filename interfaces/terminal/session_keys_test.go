package terminal

import (
	"errors"
	"io"
	"os"
	"testing"
)

// Next reads one key, from a pipe standing in for the keyboard, and says
// whether it asks to stop. q, escape alone and ctrl-c (0x03) stop. any
// other key goes on to the next picture: enter, space, a letter, and a
// key that arrives as escape and more, whether two bytes (alt and a
// key), three (an arrow, a function key) or six (ctrl and an arrow).
func TestNextStopsOnQEscapeAndCtrlCOnly(t *testing.T) {
	for _, w := range []struct {
		keys string
		stop bool
	}{
		{"q", true}, {"\x1b", true}, {"\x03", true},
		{"\r", false}, {" ", false}, {"a", false}, {"\x7f", false},
		{"\x1bx", false}, {"\x1b[A", false}, {"\x1bOP", false}, {"\x1b[1;5C", false},
	} {
		s := &Session{in: pipeOf(t, w.keys)}
		stop, err := s.Next()
		if err != nil || stop != w.stop {
			t.Errorf("keys %q: stop %v, %v; want stop %v", w.keys, stop, err, w.stop)
		}
	}
}

// a keyboard that has gone, whether its end came or its file was closed,
// is a stop, with the reason. a pipe closed at the writer's end with
// nothing sent reads as the end of file.
func TestNextStopsAndSaysWhyWhenTheKeyboardIsGone(t *testing.T) {
	s := &Session{in: pipeOf(t, "")}
	if stop, err := s.Next(); !stop || !errors.Is(err, io.EOF) {
		t.Errorf("an empty closed pipe: stop %v, %v; want stop and end of file", stop, err)
	}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	w.Close()
	r.Close()
	s = &Session{in: r}
	if stop, err := s.Next(); !stop || !errors.Is(err, os.ErrClosed) {
		t.Errorf("a closed file: stop %v, %v; want stop and the file closed", stop, err)
	}
}

// Wait returns once a key has come, whatever it is, and the error when
// none can: it does not look at what was pressed.
func TestWaitReturnsOnAnyKeyAndSaysWhenThereCanBeNone(t *testing.T) {
	for _, keys := range []string{"x", "q", "\x1b", "\x1b[A", "0123456789abcdefghij"} {
		s := &Session{in: pipeOf(t, keys)}
		if err := s.Wait(); err != nil {
			t.Errorf("keys %q: %v", keys, err)
		}
	}
	s := &Session{in: pipeOf(t, "")}
	if err := s.Wait(); !errors.Is(err, io.EOF) {
		t.Errorf("an empty closed pipe: %v, want end of file", err)
	}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	w.Close()
	r.Close()
	s = &Session{in: r}
	if err := s.Wait(); !errors.Is(err, os.ErrClosed) {
		t.Errorf("a closed file: %v, want the file closed", err)
	}
}

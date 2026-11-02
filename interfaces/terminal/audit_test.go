package terminal

import (
	"strings"
	"testing"
)

// no file is no terminal, said so, not a panic at its name.
func TestMeasuringNoFileSaysSo(t *testing.T) {
	_, err := Measure(nil)
	if err == nil || !strings.Contains(err.Error(), "no file") {
		t.Errorf("measuring nil: %v", err)
	}
}

// a key that arrives as escape and more, an arrow or a function key, asks
// for the next picture; escape alone, q and ctrl-c stop. read from a pipe
// standing in for the keyboard.
func TestAnArrowIsTheNextPictureAndEscapeAloneStops(t *testing.T) {
	for keys, stop := range map[string]bool{"\x1b[A": false, "\x1bOP": false,
		"\x1b": true, "q": true, "\x03": true, "\r": false, " ": false} {
		s := &Session{in: pipeOf(t, keys)}
		got, err := s.Next()
		if err != nil || got != stop {
			t.Errorf("keys %q: stop %v, %v; want stop %v", keys, got, err, stop)
		}
	}
}

package terminal

import (
	"errors"
	"os"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

// a pipe is a file and not a terminal, and has no size to ask: Measure says so,
// by name, with the system's own reason kept in the error. it does not go on
// and report a screen of zero cells, which is a different error, about a
// terminal that was reached.
func TestMeasuringAPipeSaysItIsNotATerminal(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	_, err = Measure(r)
	if err == nil {
		t.Fatal("a pipe was measured")
	}
	if !strings.Contains(err.Error(), "is not a terminal that says its size") ||
		!strings.Contains(err.Error(), r.Name()) {
		t.Errorf("got %q, want it to name the file and say it is not a terminal", err)
	}
	if strings.Contains(err.Error(), "says it is") {
		t.Errorf("got %q: that is the error for a terminal of no size", err)
	}
	if !errors.Is(err, unix.ENOTTY) {
		t.Errorf("got %v, want the system's not-a-typewriter kept in it", err)
	}
}

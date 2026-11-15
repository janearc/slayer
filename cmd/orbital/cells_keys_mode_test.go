package main

// These tests drive -keys in the process, on pseudo-terminals the tests
// make. They are never the terminal the tests are run from.

import (
	"bytes"
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// refusal is what Open and Check say of a file that is not a terminal.
const refusal = "needs a terminal on both input and output"

// screen is what a pseudo-terminal's master has been sent.
type screen struct {
	mu   sync.Mutex
	seen bytes.Buffer
	done chan struct{}
}

// Write keeps what the master reads.
func (scr *screen) Write(chunk []byte) (int, error) {
	scr.mu.Lock()
	defer scr.mu.Unlock()
	return scr.seen.Write(chunk)
}

// text is everything the master has read.
func (scr *screen) text() string {
	scr.mu.Lock()
	defer scr.mu.Unlock()
	return scr.seen.String()
}

// copyFrom reads the master until the terminal ends.
func (scr *screen) copyFrom(master *os.File) {
	defer close(scr.done)
	io.Copy(scr, master)
}

// ends are the two ends of a pty.
type ends struct {
	master, slave *os.File
}

// close closes the slave and then the master.
func (pair ends) close() {
	pair.slave.Close()
	pair.master.Close()
}

// descriptor is the file's descriptor as the terminal package takes it.
func descriptor(file *os.File) int { return int(file.Fd()) }

// noMode is the mode a file is opened with when it is not made. It is defined
// as zero.
const noMode os.FileMode = 0

// terminalPair is a pty made for one test. The slave is the terminal the
// orbital is given. The screen is what the master reads of it.
func terminalPair(t *testing.T) (master *os.File, slave *os.File, seen *screen) {
	t.Helper()
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatalf("no pseudo-terminal to be had: %v", err)
	}
	t.Cleanup(ends{master: master, slave: slave}.close)
	seen = &screen{done: make(chan struct{})}
	go seen.copyFrom(master)
	return master, slave, seen
}

// finished closes the slave and gives all that was drawn on it.
func finished(t *testing.T, slave *os.File, seen *screen) string {
	t.Helper()
	slave.Close()
	select {
	case <-seen.done:
	case <-time.After(limit()):
		t.Fatal("the master never saw the end of the terminal")
	}
	return seen.text()
}

// limit is how long a test waits on a terminal.
func limit() time.Duration {
	return time.Duration(pick("run-limit-seconds")) * time.Second
}

// keysAsk is a good ask for cells, to be driven by the keys on in and
// drawn on out.
func keysAsk(in, out *os.File) cellsFlags {
	ask := goodAsk()
	ask.keys, ask.in, ask.out = true, in, out
	return ask
}

// anti-test: a terminal that cannot be drawn on is not taken over. Run
// says the screen could not be taken, and the keyboard is left as it was.
func TestRunWithKeysLeavesATerminalItCannotDrawOn(t *testing.T) {
	_, slave, seen := terminalPair(t)
	unwritable, err := os.OpenFile(slave.Name(), os.O_RDONLY, noMode)
	if err != nil {
		t.Fatal(err)
	}
	before, err := term.GetState(descriptor(slave))
	if err != nil {
		t.Fatal(err)
	}
	mode, err := askCells(keysAsk(slave, unwritable))
	if err != nil {
		t.Fatalf("both are terminals, so the ask is good: %v", err)
	}
	world, eye := smokeScene(t)
	err = mode.run(unwritable, world, eye)
	if err == nil || !strings.Contains(err.Error(), "take over the screen") {
		t.Errorf("run: %v; want it to say the screen could not be taken", err)
	}
	after, _ := term.GetState(descriptor(slave))
	if !reflect.DeepEqual(before, after) {
		t.Error("the keyboard was left raw")
	}
	unwritable.Close()
	if got := finished(t, slave, seen); got != "" {
		t.Errorf("something was drawn: %q", got)
	}
}

// -keys without -cells has nothing to wait between, and is refused.
func TestKeysWithoutCellsAreRefused(t *testing.T) {
	ask := cellsFlags{keys: true}
	if mode, err := askCells(ask); !errors.Is(err, errKeysAlone) || mode.on {
		t.Errorf("-keys alone: on %v, %v; want errKeysAlone", mode.on, err)
	}
}

// anti-tests: a pipe at either end is refused in Open's words.
func TestKeysNeedATerminalAtBothEnds(t *testing.T) {
	_, slave, _ := terminalPair(t)
	pipeIn, pipeOut, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer pipeIn.Close()
	defer pipeOut.Close()
	for _, row := range []struct {
		what    string
		in, out *os.File
	}{
		{"pipes for both", pipeIn, pipeOut},
		{"a pipe for the keys", pipeIn, slave},
		{"a pipe for the frames", slave, pipeOut},
	} {
		mode, err := askCells(keysAsk(row.in, row.out))
		if err == nil || err.Error() != refusal || mode.on {
			t.Errorf("%s: on %v, %v; want %q", row.what, mode.on, err,
				refusal)
		}
	}
}

// Asked well on a terminal, -keys turns cells on with the keyboard to
// read. Without -keys the mode has none.
func TestKeysOnATerminalTurnCellsOnWithItsKeyboard(t *testing.T) {
	_, slave, _ := terminalPair(t)
	mode, err := askCells(keysAsk(slave, slave))
	if err != nil || !mode.on || mode.in != slave {
		t.Errorf("a good ask with keys: on %v, in %v, %v", mode.on, mode.in, err)
	}
	if plain, _ := askCells(goodAsk()); plain.in != nil {
		t.Error("cells without -keys have a keyboard")
	}
}

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
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"golang.org/x/term"
)

// The bytes of a terminal, from xterm's control sequences. Control-c is
// the byte 3. Mode 1049 is the alternate screen. Mode 25 is the cursor.
// Mode 2026 begins a frame. The right arrow is escape then bracket then C.
const (
	arrowKey  = "\x1b[C"
	escapeKey = "\x1b"
	controlC  = "\x03"
	quitKey   = "q"
	takenOver = "\x1b[?1049h\x1b[?25l"
	givenBack = "\x1b[0m\x1b[?25h\x1b[?1049l"
	frameMark = "\x1b[?2026h"
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

// keyed is a pseudo-terminal that has one key waiting on it, and the
// cells mode asked of it. The terminal is made raw first, so that the key
// can be read when the mode asks for it.
func keyed(t *testing.T, key string) (cellsMode, *os.File, *screen) {
	t.Helper()
	mode, master, slave, seen := rawTerminal(t)
	if _, err := master.WriteString(key); err != nil {
		t.Fatal(err)
	}
	return mode, slave, seen
}

// rawTerminal is the cells mode on a new raw pty, with the pty's parts.
func rawTerminal(t *testing.T) (cellsMode, *os.File, *os.File, *screen) {
	t.Helper()
	master, slave, seen := terminalPair(t)
	if _, err := term.MakeRaw(descriptor(slave)); err != nil {
		t.Fatal(err)
	}
	mode, err := askCells(keysAsk(slave, slave))
	if err != nil {
		t.Fatal(err)
	}
	return mode, master, slave, seen
}

// tick is how often a test looks at the screen. It is defined as a
// fiftieth of a second.
const tick = 20 * time.Millisecond

// framesOn is how many frames the screen has begun.
func framesOn(seen *screen) int {
	return strings.Count(seen.text(), frameMark)
}

// typed sends each key once the frame before it is on the screen. Keys
// sent together would be read as one.
func typed(master *os.File, seen *screen, keys ...string) {
	for sent, key := range keys {
		for end := time.Now().Add(limit()); framesOn(seen) <= sent &&
			time.Now().Before(end); {
			time.Sleep(tick)
		}
		master.WriteString(key)
	}
}

// drive is a run of the cells mode, started to be waited for.
type drive struct {
	mode   cellsMode
	out    *os.File
	world  *thing.World
	eye    maths.Subspace
	result chan error
}

// run runs the mode and says how it ended.
func (job drive) run() { job.result <- job.mode.run(job.out, job.world, job.eye) }

// started runs the mode on its terminal over the smoke scene, apart from
// the test.
func started(t *testing.T, mode cellsMode, slave *os.File) drive {
	t.Helper()
	world, eye := smokeScene(t)
	job := drive{mode: mode, out: slave, world: world, eye: eye,
		result: make(chan error, aFrame)}
	go job.run()
	return job
}

// exits is where a mode that is stopped by a signal says how it left.
type exits chan int

// leave is the mode's way out, in place of ending the test.
func (gone exits) leave(code int) { gone <- code }

// drawnWith runs the cells mode on its terminal over the smoke scene,
// and gives what the run said and what was drawn.
func drawnWith(t *testing.T, mode cellsMode, slave *os.File,
	seen *screen) (string, error) {
	t.Helper()
	world, eye := smokeScene(t)
	err := mode.run(slave, world, eye)
	return finished(t, slave, seen), err
}

// With keys run draws the first frame, then the next at a key. Two frames
// are allowed, and a second key after the last ends it.
func TestRunWithKeysDrawsTheNextFrameAtAKey(t *testing.T) {
	mode, master, slave, seen := rawTerminal(t)
	go typed(master, seen, arrowKey, arrowKey)
	got, err := drawnWith(t, mode, slave, seen)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if frames := strings.Count(got, frameMark); frames != pick("cells-frames") {
		t.Errorf("%d frames drawn; want %d", frames, pick("cells-frames"))
	}
	if !strings.HasPrefix(got, takenOver) || !strings.HasSuffix(got, givenBack) {
		t.Errorf("the screen was not taken at the start and given back at "+
			"the end: starts %q, ends %q", got[:len(takenOver)],
			got[len(got)-len(givenBack):])
	}
}

// anti-test: a stop key ends run at the first frame with no error. The
// terminal is given back.
func TestRunWithKeysStopsAtAStopKey(t *testing.T) {
	for _, key := range []string{quitKey, escapeKey, controlC} {
		mode, slave, seen := keyed(t, key)
		got, err := drawnWith(t, mode, slave, seen)
		if err != nil {
			t.Errorf("key %q: run: %v", key, err)
		}
		if frames := strings.Count(got, frameMark); frames != aFrame {
			t.Errorf("key %q: %d frames drawn; want one", key, frames)
		}
		if !strings.HasSuffix(got, givenBack) {
			t.Errorf("key %q: the terminal was not given back: %q", key, got)
		}
	}
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

// After the last frame run waits for one more key. Any key then ends it.
func TestRunWithKeysWaitsAtTheLastFrame(t *testing.T) {
	mode, master, slave, seen := rawTerminal(t)
	job := started(t, mode, slave)
	go typed(master, seen, arrowKey)
	for end := time.Now().Add(limit()); framesOn(seen) < pick("cells-frames") &&
		time.Now().Before(end); {
		time.Sleep(tick)
	}
	time.Sleep(settle())
	select {
	case err := <-job.result:
		t.Fatalf("run ended at the last frame with no key: %v", err)
	default:
	}
	if strings.Contains(seen.text(), givenBack) {
		t.Error("the terminal was given back with no key")
	}
	if _, err := master.WriteString(arrowKey); err != nil {
		t.Fatal(err)
	}
	if err := <-job.result; err != nil {
		t.Errorf("run: %v", err)
	}
	if got := finished(t, slave, seen); !strings.HasSuffix(got, givenBack) {
		t.Errorf("the terminal was not given back: %q", got)
	}
}

// settle is how long a test lets run sit to see that it waits.
func settle() time.Duration {
	return time.Duration(pick("settle-milliseconds")) * time.Millisecond
}

// Each signal gives the terminal back and exits 128 plus its number. The
// test signals itself, and the way out is a channel.
func TestASignalGivesTheTerminalBackAndLeaves(t *testing.T) {
	for _, sent := range []syscall.Signal{syscall.SIGTERM, syscall.SIGINT,
		syscall.SIGHUP} {
		mode, master, slave, seen := rawTerminal(t)
		gone := make(exits, aFrame)
		mode.exit = gone.leave
		job := started(t, mode, slave)
		for end := time.Now().Add(limit()); framesOn(seen) < aFrame &&
			time.Now().Before(end); {
			time.Sleep(tick)
		}
		if err := syscall.Kill(syscall.Getpid(), sent); err != nil {
			t.Fatal(err)
		}
		select {
		case code := <-gone:
			if code != shellExit+int(sent) {
				t.Errorf("signal %d: exit %d; want %d", sent, code,
					shellExit+int(sent))
			}
		case <-time.After(limit()):
			t.Fatalf("signal %d: the mode did not leave", sent)
		}
		master.WriteString(quitKey)
		<-job.result
		if got := finished(t, slave, seen); !strings.HasSuffix(got, givenBack) {
			t.Errorf("signal %d: the terminal was not given back: %q", sent, got)
		}
	}
}

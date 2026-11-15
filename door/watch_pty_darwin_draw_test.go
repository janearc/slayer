//go:build draw

package door

// watch on a pseudo-terminal the test makes, drawing a smoke on the cpu:
// a picture appears with its status row; a resize draws again; a save
// draws again; byte 0x1d does nothing; q gives the terminal back as it
// was. The pty is never the terminal the tests are run from.

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/janearc/slayer/render/live"
	"golang.org/x/sys/unix"
)

// The pty's sizes, picked by the author of this slice, 2026-11-12: small,
// so a cpu frame is quick, and two that differ.
const (
	ptyCols, ptyRows     = 40, 16
	laterCols, laterRows = 50, 18
	// ptyLimit is how long the test waits for the screen to say a thing
	ptyLimit = 20 * time.Second
	// pollTwice is longer than watch takes to look at the file
	// twice, so a save is all seen before the next step
	pollTwice = 4 * live.PollEvery
)

// ptyPair is a pseudo-terminal closed with the test: a master the test
// reads as a screen and writes as a keyboard, and a slave that is a
// terminal. It is made the darwin way, as interfaces/terminal's tests
// make theirs; a machine with none fails the test.
func ptyPair(t *testing.T) (master, slave *os.File) {
	t.Helper()
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("no pseudo-terminal: %v", err)
	}
	for _, request := range []uint{unix.TIOCPTYGRANT, unix.TIOCPTYUNLK} {
		if err := unix.IoctlSetInt(int(master.Fd()), request,
			0); err != nil {
			t.Fatalf("no pseudo-terminal: %v", err)
		}
	}
	var name [128]byte
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, master.Fd(),
		unix.TIOCPTYGNAME, uintptr(unsafe.Pointer(&name[0]))); errno != 0 {
		t.Fatalf("no pseudo-terminal's name: %v", errno)
	}
	slave, err = os.OpenFile(string(bytes.TrimRight(name[:], "\x00")),
		os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		t.Fatalf("no pseudo-terminal: %v", err)
	}
	t.Cleanup(func() {
		slave.Close()
		master.Close()
	})
	return master, slave
}

// sizedPty tells the pty its size in cells.
func sizedPty(t *testing.T, master *os.File, cols, rows int) {
	t.Helper()
	size := &unix.Winsize{Col: uint16(cols), Row: uint16(rows)}
	if err := unix.IoctlSetWinsize(int(master.Fd()), unix.TIOCSWINSZ,
		size); err != nil {
		t.Fatal(err)
	}
}

// screen is everything the pty has been sent, read as it comes so the
// program writing to it never waits.
type screen struct {
	mu   sync.Mutex
	sent bytes.Buffer
}

// readingPty reads the master into a screen until it is closed.
func readingPty(master *os.File) *screen {
	shown := &screen{}
	go func() {
		chunk := make([]byte, 4096)
		for {
			count, err := master.Read(chunk)
			if err != nil {
				return
			}
			shown.mu.Lock()
			shown.sent.Write(chunk[:count])
			shown.mu.Unlock()
		}
	}()
	return shown
}

// since is what the screen was sent after mark bytes.
func (shown *screen) since(mark int) string {
	shown.mu.Lock()
	defer shown.mu.Unlock()
	return shown.sent.String()[min(mark, shown.sent.Len()):]
}

// length is how much the screen has been sent.
func (shown *screen) length() int {
	shown.mu.Lock()
	defer shown.mu.Unlock()
	return shown.sent.Len()
}

// waitFor waits until the screen, after mark, says every word.
func (shown *screen) waitFor(t *testing.T, mark int, words ...string) {
	t.Helper()
	deadline := time.Now().Add(ptyLimit)
	for time.Now().Before(deadline) {
		sent, all := shown.since(mark), true
		for _, word := range words {
			all = all && strings.Contains(sent, word)
		}
		if all {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("the screen never said %q; it said %q", words,
		shown.since(mark))
}

// statusOf is the watch's status file, read.
func statusOf(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(statusFolder(path),
		"status.json"))
	var said map[string]any
	if err != nil || json.Unmarshal(raw, &said) != nil {
		return nil
	}
	return said
}

// statusAt is the status folder's record once a key says a value, or
// the last record read after statusWait: the status is written just after
// the screen shows the rung, so a loaded machine can read it a moment
// early.
func statusAt(t *testing.T, path, key string, want any) map[string]any {
	t.Helper()
	until := time.Now().Add(statusWait)
	for {
		said := statusOf(t, path)
		if said[key] == want || time.Now().After(until) {
			return said
		}
		time.Sleep(statusPoll)
	}
}

// statusWait and statusPoll are how long statusAt waits for the status
// to catch up with the screen, and how often it looks.
const (
	statusWait = 5 * time.Second
	statusPoll = 20 * time.Millisecond
)

// truecolour is how the terminal package draws a cell's colour: a
// picture was drawn when the screen holds one.
const truecolour = "\x1b[38;2;"

func TestWatchUnderAPty(t *testing.T) {
	master, slave := ptyPair(t)
	sizedPty(t, master, ptyCols, ptyRows)
	before, err := unix.IoctlGetTermios(int(slave.Fd()), unix.TIOCGETA)
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	path := written(t, directory, "smoke.yaml", stillJob(t, ""))
	shown := readingPty(master)
	env := envIn(t, context.Background(), directory)
	ended := make(chan int, one)
	var wrong bytes.Buffer
	go func() {
		ended <- Tool(env).RunWith([]string{"watch", path}, slave, slave,
			&wrong)
	}()
	shown.waitFor(t, nothing, truecolour, "preview, 1/8 size")
	shown.waitFor(t, nothing, "pane size, 4 rays")
	if said := statusAt(t, path, "rays", 4.0); said["rays"] != 4.0 {
		t.Errorf("the status folder says %v", said)
	}

	mark := shown.length()
	sizedPty(t, master, laterCols, laterRows)
	syscall.Kill(syscall.Getpid(), syscall.SIGWINCH)
	shown.waitFor(t, mark, "\x1b[2J", "1/8 size", "pane size, 4 rays")

	mark = shown.length()
	later := time.Now().Add(time.Minute)
	written(t, directory, "smoke.yaml", stillJob(t,
		"print: {knee: 0.5, top: 4}\n"))
	os.Chtimes(path, later, later)
	shown.waitFor(t, mark, truecolour)
	time.Sleep(pollTwice)

	mark = shown.length()
	master.Write([]byte{0x1d})
	time.Sleep(300 * time.Millisecond)
	select {
	case code := <-ended:
		t.Fatalf("byte 0x1d ended watch with %d", code)
	default:
	}
	if sent := shown.since(mark); strings.Contains(sent, truecolour) {
		t.Errorf("byte 0x1d drew the picture again: %q", sent)
	}

	master.Write([]byte("q"))
	select {
	case code := <-ended:
		if code != nothing {
			t.Errorf("q ended watch with %d: %s", code, wrong.String())
		}
	case <-time.After(ptyLimit):
		t.Fatal("q did not end watch")
	}
	after, err := unix.IoctlGetTermios(int(slave.Fd()), unix.TIOCGETA)
	if err != nil || *after != *before {
		t.Errorf("the terminal was not given back as it was: %v", err)
	}
	shown.waitFor(t, mark, "\x1b[?1049l")
}

package main_test

// These tests run the built orbital with -keys as a window would: on a
// pseudo-terminal the test makes, with keys sent through its master. They
// are never the terminal the tests are run from.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
)

// The bytes of a terminal, from xterm's control sequences. Control-c is
// the byte 3. Mode 1049 is the alternate screen. Mode 25 is the cursor.
// The right arrow is escape then bracket then C.
const (
	arrowKey  = "\x1b[C"
	escapeKey = "\x1b"
	controlC  = "\x03"
	quitKey   = "q"
	takenOver = "\x1b[?1049h\x1b[?25l"
	givenBack = "\x1b[0m\x1b[?25h\x1b[?1049l"
)

// pollEvery is how often a test looks at what the window has been sent.
// It is defined as a fiftieth of a second.
const pollEvery = 20 * time.Millisecond

// pollMillis is how long pump waits on the master before it looks again
// at whether the window has left. It is defined as a tenth of a second.
const pollMillis = 100

// shellExit is what a shell adds to a signal's number to say a command
// died of it. It is defined as 128.
const shellExit = 128

// window is the orbital running on a pseudo-terminal. The test is the
// terminal: it sends the keys and reads what is drawn.
type window struct {
	t      *testing.T
	ctx    context.Context
	cancel context.CancelFunc
	cmd    *exec.Cmd
	master *os.File
	stderr bytes.Buffer
	mu     sync.Mutex
	seen   bytes.Buffer
	read   chan struct{}

	leaving atomic.Bool
}

// open starts the orbital on a new terminal, as a window starts a command.
// If the test ends first, the orbital is stopped by its pid.
func open(t *testing.T, program string, args []string) *window {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), limit())
	win := &window{t: t, ctx: ctx, cancel: cancel, read: make(chan struct{}),
		cmd: exec.CommandContext(ctx, program, args...)}
	win.cmd.Dir = t.TempDir()
	win.cmd.Stderr = &win.stderr
	master, err := pty.Start(win.cmd)
	if err != nil {
		cancel()
		t.Fatalf("the orbital would not start on a terminal: %v", err)
	}
	win.master = master
	t.Cleanup(win.stop)
	go win.pump()
	return win
}

// stop ends the orbital if it is still running, and closes the window.
func (win *window) stop() {
	win.cancel()
	win.leave()
}

// pump keeps what the orbital draws until the terminal ends or the window
// leaves. It asks the master whether there is anything to read, since a
// read of a master opened blocking cannot be stopped by closing it.
func (win *window) pump() {
	defer close(win.read)
	chunk := make([]byte, picked("read-chunk"))
	for !win.leaving.Load() {
		fds := []unix.PollFd{{Fd: int32(win.master.Fd()),
			Events: unix.POLLIN}}
		ready, err := unix.Poll(fds, pollMillis)
		if err != nil && !errors.Is(err, unix.EINTR) {
			return
		}
		if ready <= nothing {
			continue
		}
		got, err := win.master.Read(chunk)
		win.Write(chunk[:got])
		if err != nil {
			return
		}
	}
}

// leave closes the master, as a window does that has gone away. The
// orbital then has no terminal to read.
func (win *window) leave() {
	win.leaving.Store(true)
	<-win.read
	win.master.Close()
}

// Write keeps what the orbital draws.
func (win *window) Write(chunk []byte) (int, error) {
	win.mu.Lock()
	defer win.mu.Unlock()
	return win.seen.Write(chunk)
}

// text is what has been drawn so far.
func (win *window) text() string {
	win.mu.Lock()
	defer win.mu.Unlock()
	return win.seen.String()
}

// framesDrawn waits until that many whole frames have been drawn, or the
// limit passes and the test stops.
func (win *window) framesDrawn(count int) {
	win.t.Helper()
	for end := time.Now().Add(limit()); time.Now().Before(end); {
		if strings.Count(win.text(), frameEnd) >= count {
			return
		}
		time.Sleep(pollEvery)
	}
	win.t.Fatalf("%d frames did not come in %v; drawn so far %q\nstderr %q",
		count, limit(), win.text(), win.stderr.String())
}

// press sends keys, as a person's fingers would.
func (win *window) press(keys string) {
	win.t.Helper()
	if _, err := win.master.WriteString(keys); err != nil {
		win.t.Fatal(err)
	}
}

// ended waits for the orbital to exit and says how it did, and gives all
// it drew.
func (win *window) ended() (code int, drawn string) {
	win.t.Helper()
	waited := win.cmd.Wait()
	if win.ctx.Err() != nil {
		win.t.Fatalf("the orbital did not exit in %v; drawn %q\nstderr %q",
			limit(), win.text(), win.stderr.String())
	}
	var exited *exec.ExitError
	if errors.As(waited, &exited) {
		code = exited.ExitCode()
	} else if waited != nil {
		win.t.Fatalf("the orbital did not exit: %v", waited)
	}
	select {
	case <-win.read:
	case <-time.After(limit()):
		win.t.Fatal("the master never saw the end of the terminal")
	}
	return code, win.text()
}

// framedArgs draw the picked size from the picked first frame. The count
// and the rest are given.
func framedArgs(frames int, rest ...string) []string {
	size := fmt.Sprintf("%dx%d", picked("cells-cols"), picked("cells-rows"))
	first, rays := picked("keys-first"), picked("cells-rays")
	args := []string{"-cells", size, "-first", fmt.Sprint(first),
		"-frames", fmt.Sprint(frames), "-n", fmt.Sprint(rays)}
	return append(args, rest...)
}

// framesOf is the frames of a stream, each with its end.
func framesOf(stream string) []string {
	frames := strings.SplitAfter(stream, frameEnd)
	return frames[:len(frames)-once]
}

// settle is how long a test lets the orbital sit, to see that it waits.
func settle() time.Duration {
	return time.Duration(picked("settle-milliseconds")) * time.Millisecond
}

// With -keys the orbital draws a frame and waits for a key. Each frame is
// the cells the stream has for it. After the last frame it waits for one
// more key and the picture stays. The screen is taken before the first
// frame and given back after that key, with nothing else on it.
func TestKeysDrawTheStreamsFramesOneKeyAtATime(t *testing.T) {
	program, count := binary(t), picked("keys-frames")
	streamed := run(t, program, framedArgs(count))
	if streamed.err != nil {
		t.Fatalf("%v\n%s", streamed.err, streamed.stderr)
	}
	if got := len(framesOf(streamed.stdout)); got != count {
		t.Fatalf("the stream has %d frames; want %d", got, count)
	}
	win := open(t, program, framedArgs(count, "-keys"))
	for shown := once; shown <= count; shown++ {
		win.framesDrawn(shown)
		time.Sleep(settle())
		if got := strings.Count(win.text(), frameEnd); got != shown {
			t.Fatalf("%d frames drawn before key %d; want %d", got, shown,
				shown)
		}
		if strings.Contains(win.text(), givenBack) {
			t.Fatalf("the screen was given back before key %d", shown)
		}
		win.press(arrowKey)
	}
	code, drawn := win.ended()
	if code != nothing {
		t.Errorf("exit %d after the last frame; stderr %q", code,
			win.stderr.String())
	}
	if wanted := takenOver + streamed.stdout + givenBack; drawn != wanted {
		t.Errorf("the screen is not the stream between taking and giving "+
			"back: %d bytes, want %d", len(drawn), len(wanted))
	}
	if !strings.Contains(win.stderr.String(), "ring 0:") {
		t.Errorf("the chat did not go to standard error: %q",
			win.stderr.String())
	}
}

// anti-tests: q, escape and control-c each end a run with no other end.
// It exits well after one frame and gives the terminal back.
func TestKeysEndAtQEscapeAndControlC(t *testing.T) {
	program := binary(t)
	for _, key := range []string{quitKey, escapeKey, controlC} {
		win := open(t, program, framedArgs(nothing, "-keys"))
		win.framesDrawn(once)
		win.press(key)
		code, drawn := win.ended()
		if code != nothing {
			t.Errorf("key %q: exit %d; stderr %q", key, code,
				win.stderr.String())
		}
		if got := strings.Count(drawn, frameEnd); got != once {
			t.Errorf("key %q: %d frames drawn; want the first only", key,
				got)
		}
		if !strings.HasPrefix(drawn, takenOver) ||
			!strings.HasSuffix(drawn, givenBack) {
			t.Errorf("key %q: the screen was not taken and given back: %q",
				key, drawn)
		}
	}
}

// One frame stays until a key. Then q ends the run with exit 0.
func TestKeysQAfterTheLastFrameEndsIt(t *testing.T) {
	win := open(t, binary(t), framedArgs(once, "-keys"))
	win.framesDrawn(once)
	time.Sleep(settle())
	if strings.Contains(win.text(), givenBack) {
		t.Fatal("the screen was given back with no key")
	}
	win.press(quitKey)
	code, drawn := win.ended()
	if code != nothing || !strings.HasPrefix(drawn, takenOver) ||
		!strings.HasSuffix(drawn, givenBack) {
		t.Errorf("exit %d, screen %q; want 0, taken and given back", code,
			drawn)
	}
}

// Each signal gives the terminal back and exits 128 plus its number.
func TestKeysSignalsGiveTheTerminalBack(t *testing.T) {
	program := binary(t)
	for _, sent := range []syscall.Signal{syscall.SIGTERM, syscall.SIGINT,
		syscall.SIGHUP} {
		win := open(t, program, framedArgs(nothing, "-keys"))
		win.framesDrawn(once)
		if err := win.cmd.Process.Signal(sent); err != nil {
			t.Fatal(err)
		}
		code, drawn := win.ended()
		if code != shellExit+int(sent) || !strings.HasSuffix(drawn, givenBack) {
			t.Errorf("signal %d: exit %d, screen %q; want %d and the "+
				"terminal given back", sent, code, drawn, shellExit+int(sent))
		}
	}
}

// anti-test: a window that goes away ends the orbital. It does not wait
// for a key from a keyboard that is gone.
func TestKeysEndWhenTheWindowGoes(t *testing.T) {
	win := open(t, binary(t), framedArgs(nothing, "-keys"))
	win.framesDrawn(once)
	win.leave()
	if code, _ := win.ended(); code == nothing {
		t.Errorf("exit 0 with no window to draw on; stderr %q",
			win.stderr.String())
	}
}

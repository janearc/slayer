package terminal

// Live is the input and output side of a live view: the screen's size,
// told again when it changes; keys; and a place to put cells. It draws
// nothing of its own.
//
// Design. OpenLive takes the terminal as a Session does: the alternate
// screen, no cursor, raw input. One reader goroutine polls the keyboard
// and a wake pipe, decodes keys, and holds the wait that tells a lone
// escape from the start of a sequence.
//
// One more goroutine follows the window-change signal and measures the
// screen again. Both send to one channel of events and both stop on Close,
// which waits for them. A signal that would kill the process gives the
// terminal back first.
//
// Byte 0x1d, control-right-bracket, belongs to the program that may be
// hosting this one. It arrives as KeyHost, and a caller must not bind it.
//
// Present draws cells clipped to the screen, clearing first when the size
// has changed. Status writes one line on the last row. A caller that uses
// both keeps its picture to one row fewer than the screen has, or Status
// writes over the picture's last row. A clear wipes the status row too:
// write it again after a resize.

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// The sequences a live view sends, from xterm's control sequences. The
// first four are Session's: the alternate screen, the cursor, colour reset.
const (
	liveTake  = "\x1b[?1049h\x1b[?25l"
	liveGive  = "\x1b[0m\x1b[?25h\x1b[?1049l"
	clearAll  = "\x1b[?2026h\x1b[2J" // begin a frame, clear the screen
	wrapOff   = "\x1b[?7l"           // no wrapping at the right edge
	wrapOn    = "\x1b[?7h"
	eraseRest = "\x1b[K"  // erase from the cursor to the end of the row
	resetSGR  = "\x1b[0m" // default colours
)

// EscapeWait is how long a lone escape byte waits for more before it is
// reported as escape. It is picked by the author of this slice, 2026-11-12,
// open to cutting: longer than the gap inside one key's bytes on a slow
// link, shorter than a person notices.
const EscapeWait = 50 * time.Millisecond

// eventRoom is how many events wait for a caller before the reader waits.
// It is picked by the author of this slice, 2026-11-12: a burst of keys
// held down, and a few resizes.
const eventRoom = 32

// readRoom is how many bytes one read takes. It is picked by the author of
// this slice, 2026-11-12: a paste is split over many reads, which the
// decoder handles.
const readRoom = 256

// EventKind says what an Event is.
type EventKind int

// The kinds of event.
const (
	EventKey    EventKind = iota // a key was pressed
	EventResize                  // the screen's size changed
	EventEnd                     // the input ended; no more keys will come
)

// Event is one thing that happened to the terminal.
type Event struct {
	kind   EventKind
	key    Key
	screen Screen
}

// Kind says what happened.
func (event Event) Kind() EventKind { return event.kind }

// Key is the key pressed, of an EventKey.
func (event Event) Key() Key { return event.key }

// Screen is the new size, of an EventResize.
func (event Event) Screen() Screen { return event.screen }

// KeyEvent is the event of a key pressed, as Live sends it; for a view's
// tests, which have no terminal to press it on. A key comes from a
// Decoder.
func KeyEvent(key Key) Event { return Event{kind: EventKey, key: key} }

// ResizeEvent is the event of the screen's size changing, as Live sends
// it; for a view's tests.
func ResizeEvent(screen Screen) Event {
	return Event{kind: EventResize, screen: screen}
}

// EndEvent is the event of the input ending, as Live sends it; for a
// view's tests.
func EndEvent() Event { return Event{kind: EventEnd} }

// Live is a terminal taken over for a live view.
type Live struct {
	in, out *os.File
	old     *term.State
	wait    time.Duration
	events  chan Event
	winch   chan os.Signal
	wakeIn  *os.File // the reader polls this; closing its mate wakes it
	wakeOut *os.File
	done    chan struct{}
	workers sync.WaitGroup
	watch   *Watch

	mu        sync.Mutex
	screen    Screen
	presented [2]int // cols and rows the last present was made for
	restored  sync.Once
	closed    sync.Once
	problem   error
}

// LiveHooks are what a test replaces. The zero value is what a program
// wants: the real process exit, and EscapeWait.
type LiveHooks struct {
	Exit       func(int)     // called with 128 plus the signal; os.Exit
	EscapeWait time.Duration // the wait for a lone escape; EscapeWait
}

// OpenLive takes the terminal on in and out for a live view. It needs a
// terminal at both ends and a terminal that says its size, and refuses
// otherwise before it has changed anything.
func OpenLive(in, out *os.File) (*Live, error) {
	return OpenLiveWith(in, out, LiveHooks{})
}

// OpenLiveWith is OpenLive with hooks.
func OpenLiveWith(in, out *os.File, hooks LiveHooks) (*Live, error) {
	if err := Check(in, out); err != nil {
		return nil, err
	}
	live := &Live{in: in, out: out, wait: hooks.EscapeWait,
		events: make(chan Event, eventRoom), done: make(chan struct{}),
		winch: make(chan os.Signal, signalRoom)}
	if live.wait <= 0 {
		live.wait = EscapeWait
	}
	signal.Notify(live.winch, syscall.SIGWINCH)
	if err := live.take(); err != nil {
		signal.Stop(live.winch)
		return nil, err
	}
	live.workers.Add(2)
	go live.read()
	go live.follow()
	exit := hooks.Exit
	if exit == nil {
		exit = os.Exit
	}
	live.watch = LeaveOnSignal(live.restore, exit)
	return live, nil
}

// take measures, goes raw, takes the screen and makes the wake pipe, and
// undoes what it did if any step fails.
func (live *Live) take() error {
	screen, err := Measure(live.out)
	if err != nil {
		return err
	}
	old, err := term.MakeRaw(int(live.in.Fd()))
	if err != nil {
		return err
	}
	if _, err := io.WriteString(live.out, liveTake); err != nil {
		term.Restore(int(live.in.Fd()), old)
		return fmt.Errorf("could not take over the screen: %w", err)
	}
	live.old, live.screen = old, screen
	live.presented = [2]int{screen.Cols, screen.Rows}
	live.wakeIn, live.wakeOut, err = os.Pipe()
	if err != nil {
		live.restore()
		return fmt.Errorf("could not make a pipe to wake the reader: %w",
			err)
	}
	return nil
}

// Events is the channel of what happens: keys, resizes, the end of input.
// It is closed by Close, after the last event.
func (live *Live) Events() <-chan Event { return live.events }

// Screen is the size the terminal has now.
func (live *Live) Screen() Screen {
	live.mu.Lock()
	defer live.mu.Unlock()
	return live.screen
}

// send gives an event to the caller, or gives up when Close has been called.
func (live *Live) send(event Event) bool {
	select {
	case live.events <- event:
		return true
	case <-live.done:
		return false
	}
}

// follow measures the screen again on each window-change signal and sends a resize when
// the size is not what it was. A terminal that will not say its size, or says
// none, leaves the screen as it was.
func (live *Live) follow() {
	defer live.workers.Done()
	for {
		select {
		case <-live.winch:
			if !live.remeasure() {
				return
			}
		case <-live.done:
			return
		}
	}
}

// remeasure is false when the live view was closed while it sent.
func (live *Live) remeasure() bool {
	screen, err := Measure(live.out)
	live.mu.Lock()
	changed := err == nil && screen != live.screen
	if changed {
		live.screen = screen
	}
	live.mu.Unlock()
	return !changed || live.send(Event{kind: EventResize, screen: screen})
}

// read polls the keyboard and the wake pipe and decodes what comes. It ends
// at the end of input or on Close.
func (live *Live) read() {
	defer live.workers.Done()
	var decoder Decoder
	buf := make([]byte, readRoom)
	for {
		ready, lapsed := live.poll(decoder.Waiting())
		var keys []Key
		switch {
		case ready == pollClosed:
			return
		case lapsed:
			keys = decoder.Flush()
		case ready == pollInput:
			count, err := unix.Read(int(live.in.Fd()), buf)
			if err == unix.EAGAIN || err == unix.EINTR {
				continue
			}
			if err != nil || count <= 0 {
				live.send(Event{kind: EventEnd})
				return
			}
			keys = decoder.Feed(buf[:count])
		}
		for _, key := range keys {
			if !live.send(Event{kind: EventKey, key: key}) {
				return
			}
		}
	}
}

// What a poll found.
const (
	pollNothing = iota // the poll was interrupted; ask again
	pollInput          // the keyboard has bytes, or has ended
	pollClosed         // the live view was closed
)

// poll waits for the keyboard or the wake pipe. With waiting set it waits
// no longer than the escape wait, and says lapsed if nothing came.
func (live *Live) poll(waiting bool) (ready int, lapsed bool) {
	timeout := -1
	if waiting {
		timeout = int(live.wait / time.Millisecond)
	}
	fds := []unix.PollFd{
		{Fd: int32(live.in.Fd()), Events: unix.POLLIN},
		{Fd: int32(live.wakeIn.Fd()), Events: unix.POLLIN}}
	count, err := unix.Poll(fds, timeout)
	switch {
	case err != nil:
		return pollNothing, false
	case count == 0:
		return pollNothing, true
	case fds[1].Revents != 0:
		return pollClosed, false
	}
	return pollInput, false
}

// restore gives the terminal back: colour reset, the cursor shown, the main
// screen, raw mode undone. It runs once, from Close or from a signal.
func (live *Live) restore() {
	live.restored.Do(func() {
		_, err := io.WriteString(live.out, liveGive)
		live.mu.Lock()
		live.problem = err
		live.mu.Unlock()
		term.Restore(int(live.in.Fd()), live.old)
	})
}

// Close gives the terminal back and stops the goroutines, and waits until
// they have stopped, then closes the event channel. It says if the screen
// could not be written to. Calling it again does nothing and says nothing.
func (live *Live) Close() error {
	var first error
	live.closed.Do(func() {
		live.watch.Stop()
		signal.Stop(live.winch)
		close(live.done)
		live.wakeOut.Close()
		live.workers.Wait()
		live.wakeIn.Close()
		close(live.events)
		live.restore()
		live.mu.Lock()
		first = live.problem
		live.mu.Unlock()
	})
	return first
}

// Present draws cells at the top left, cut to the screen's size so that
// nothing wraps or piles on the last row, and clears the screen first when
// the size has changed since the last present. It draws with Draw.
func (live *Live) Present(cells *Cells) error {
	if cells == nil {
		return errors.New("nothing to present: give it cells, such as " +
			"from Chop")
	}
	screen := live.Screen()
	var frame strings.Builder
	live.mu.Lock()
	size := [2]int{screen.Cols, screen.Rows}
	if live.presented != size {
		frame.WriteString(clearAll)
		live.presented = size
	}
	live.mu.Unlock()
	shown := clipped(cells, screen.Cols, screen.Rows)
	if err := Draw(&frame, shown); err != nil {
		return err
	}
	_, err := io.WriteString(live.out, frame.String())
	return err
}

// clipped is cells cut to at most cols across and rows down.
func clipped(cells *Cells, cols, rows int) *Cells {
	wide, high := min(cells.W, cols), min(cells.H, rows)
	if wide == cells.W && high == cells.H {
		return cells
	}
	out := &Cells{W: wide, H: high, Cells: make([]Cell, 0, wide*high)}
	for y := range high {
		start := y * cells.W
		out.Cells = append(out.Cells, cells.Cells[start:start+wide]...)
	}
	return out
}

// Status writes one line of plain text on the screen's last row, in the
// terminal's default colours, and nothing above it. Text past the width is
// cut. Control characters, escape among them, are not written. A character
// is counted as one cell, so text with wide characters may be cut short of
// the width and never past it: wrapping is off while it is written.
func (live *Live) Status(text string) error {
	screen := live.Screen()
	line := statusLine(text, screen.Cols, screen.Rows)
	_, err := io.WriteString(live.out, line)
	return err
}

// Foot writes lines on the screen's last rows, the last line on the last
// row, each as Status writes its one. Lines past the screen's height are
// dropped from the top. A caller that uses it keeps its picture as many
// rows fewer than the screen has as it writes lines.
func (live *Live) Foot(lines []string) error {
	screen := live.Screen()
	_, err := io.WriteString(live.out, footLines(lines, screen.Cols,
		screen.Rows))
	return err
}

// footLines is the bytes that write lines on the last of rows, cut to
// cols, the first dropped where there are more lines than rows.
func footLines(lines []string, cols, rows int) string {
	if extra := len(lines) - rows; extra > 0 {
		lines = lines[extra:]
	}
	var written strings.Builder
	first := rows - len(lines) + 1
	for place, text := range lines {
		written.WriteString(statusLine(text, cols, first+place))
	}
	return written.String()
}

// statusLine is the bytes that write text on row rows, cut to cols.
func statusLine(text string, cols, rows int) string {
	var line strings.Builder
	fmt.Fprintf(&line, "%s\x1b[%d;1H%s", wrapOff, rows, resetSGR)
	written := 0
	for _, char := range text {
		if written == cols {
			break
		}
		if !unicode.IsPrint(char) {
			continue
		}
		line.WriteRune(char)
		written++
	}
	line.WriteString(eraseRest + wrapOn)
	return line.String()
}

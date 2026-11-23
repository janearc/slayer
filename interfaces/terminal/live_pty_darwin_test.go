package terminal

// Live on pseudo-terminals the tests make. They are never the terminal the
// tests are run from.

import (
	"os"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// how long a test waits for an event before it says none came. It is
// picked by the author of this slice, 2026-11-12: far longer than any event
// takes, short enough for a failing test to end.
const eventLimit = 3 * time.Second

// aLongWait is an escape wait no test sits through: a split sequence is
// finished well before it.
const aLongWait = time.Minute

// aShortWait is an escape wait a test sits through.
const aShortWait = 10 * time.Millisecond

// the size the live tests give the pty, and one that differs. picked by the
// test author, 2026-11-12.
const (
	startCols, startRows = 6, 4
	laterCols, laterRows = 9, 5
)

// liveOn is a live view on a pty of the starting size. exits is what the
// signal watch would have exited with.
func liveOn(t *testing.T, hooks LiveHooks) (*Live, *os.File, *os.File, chan int) {
	t.Helper()
	master, slave := ptyPair(t)
	resize(t, master, startCols, startRows, 0, 0)
	exits := make(chan int, 1)
	if hooks.Exit == nil {
		hooks.Exit = func(code int) { exits <- code }
	}
	live, err := OpenLiveWith(slave, slave, hooks)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { live.Close() })
	if got := screenOf(t, master, len(takenOver)); got != takenOver {
		t.Fatalf("OpenLive sent %q, want %q", got, takenOver)
	}
	return live, master, slave, exits
}

// next is the next event, or the test stops.
func next(t *testing.T, live *Live) Event {
	t.Helper()
	select {
	case event, open := <-live.Events():
		if !open {
			t.Fatal("the events were closed")
		}
		return event
	case <-time.After(eventLimit):
		t.Fatal("no event came")
	}
	return Event{}
}

// nextKey is the next event, which must be a key.
func nextKey(t *testing.T, live *Live) Key {
	t.Helper()
	event := next(t, live)
	if event.Kind() != EventKey {
		t.Fatalf("got event kind %d, want a key", event.Kind())
	}
	return event.Key()
}

// OpenLive takes the terminal as a session does, measures it, and Close
// gives it back to the last bit and says nothing the second time.
func TestLiveOpensAndClosesTheTerminal(t *testing.T) {
	master, slave := ptyPair(t)
	resize(t, master, startCols, startRows, 0, 0)
	before := settings(t, slave)
	live, err := OpenLive(slave, slave)
	if err != nil {
		t.Fatal(err)
	}
	if got := screenOf(t, master, len(takenOver)); got != takenOver {
		t.Errorf("OpenLive sent %q, want %q", got, takenOver)
	}
	if during := settings(t, slave); during.Lflag&(unix.ECHO|unix.ICANON) != 0 {
		t.Errorf("local modes %#x while open", during.Lflag)
	}
	if got := live.Screen(); got.Cols != startCols || got.Rows != startRows {
		t.Errorf("the screen is %+v", got)
	}
	if err := live.Close(); err != nil {
		t.Error(err)
	}
	if got := screenOf(t, master, len(givenBack)); got != givenBack {
		t.Errorf("Close sent %q, want %q", got, givenBack)
	}
	if after := settings(t, slave); after != before {
		t.Errorf("modes after Close differ:\n%+v\n%+v", after, before)
	}
	if err := live.Close(); err != nil {
		t.Errorf("the second Close said %v", err)
	}
	if _, err := slave.WriteString(probe); err != nil {
		t.Fatal(err)
	}
	if got := screenOf(t, master, 1); got != probe {
		t.Errorf("after a second Close the master read %q, want only %q", got, probe)
	}
}

// anti-test: no live view on a pipe at either end, in the words Session
// uses; and none on a terminal that says it is no size. each leaves the
// terminal as it was and sends nothing.
func TestLiveRefusesWhatIsNotATerminalOfSize(t *testing.T) {
	const refusal = "needs a terminal on both input and output"
	master, slave := ptyPair(t)
	resize(t, master, startCols, startRows, 0, 0)
	pipeIn, pipeOut, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer pipeIn.Close()
	defer pipeOut.Close()
	before := settings(t, slave)
	for what, files := range map[string][2]*os.File{
		"a pipe for both":   {pipeIn, pipeOut},
		"a pipe for input":  {pipeIn, slave},
		"a pipe for output": {slave, pipeOut},
	} {
		live, err := OpenLive(files[0], files[1])
		if live != nil || err == nil || err.Error() != refusal {
			t.Errorf("%s: got %v, %v", what, live, err)
		}
	}
	empty, emptySlave := ptyPair(t)
	_ = empty
	if live, err := OpenLive(emptySlave, emptySlave); live != nil ||
		err == nil || !strings.Contains(err.Error(), "0 by 0 cells") {
		t.Errorf("a terminal of no size: got %v, %v", live, err)
	}
	if after := settings(t, slave); after != before {
		t.Errorf("a refusal changed the modes")
	}
	if _, err := slave.WriteString(probe); err != nil {
		t.Fatal(err)
	}
	if got := screenOf(t, master, 1); got != probe {
		t.Errorf("a refusal sent %q", got)
	}
}

// a terminal that cannot be written to is not taken over and is not left
// raw.
func TestLiveGivesBackAKeyboardItCouldNotUse(t *testing.T) {
	master, slave := ptyPair(t)
	resize(t, master, startCols, startRows, 0, 0)
	before := settings(t, slave)
	live, err := OpenLive(slave, reopen(t, slave))
	if live != nil || err == nil || !strings.Contains(err.Error(), "could not take over") {
		t.Fatalf("got %v, %v", live, err)
	}
	if after := settings(t, slave); after != before {
		t.Error("the terminal was left raw")
	}
}

// a resize arrives as one event with the new size, and the screen is the
// new size. anti-test: a SIGWINCH that changes nothing sends no event, so
// the first event after it is the real change.
func TestLiveReportsAResize(t *testing.T) {
	live, master, _, _ := liveOn(t, LiveHooks{})
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGWINCH); err != nil {
		t.Fatal(err)
	}
	resize(t, master, laterCols, laterRows, 0, 0)
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGWINCH); err != nil {
		t.Fatal(err)
	}
	event := next(t, live)
	if event.Kind() != EventResize || event.Screen().Cols != laterCols ||
		event.Screen().Rows != laterRows {
		t.Errorf("got kind %d, screen %+v", event.Kind(), event.Screen())
	}
	if got := live.Screen(); got != event.Screen() {
		t.Errorf("the screen is %+v, the event said %+v", got, event.Screen())
	}
}

// anti-test: a terminal that comes to say it is no size is not a screen of
// none: the screen stays, and no event is sent.
func TestLiveKeepsTheScreenWhenItIsToldNone(t *testing.T) {
	live, master, _, _ := liveOn(t, LiveHooks{})
	resize(t, master, 0, 0, 0, 0)
	syscall.Kill(syscall.Getpid(), syscall.SIGWINCH)
	resize(t, master, laterCols, laterRows, 0, 0)
	syscall.Kill(syscall.Getpid(), syscall.SIGWINCH)
	if event := next(t, live); event.Kind() != EventResize || event.Screen().Cols != laterCols {
		t.Errorf("got %+v", event)
	}
}

// keys typed on the master arrive as events, in order.
func TestLiveHandsOverKeys(t *testing.T) {
	live, master, _, _ := liveOn(t, LiveHooks{EscapeWait: aLongWait})
	master.WriteString("a\r\x1b[A\x1bOB\x03")
	for _, want := range []Key{char('a'), key(KeyEnter), key(KeyUp), key(KeyDown), key(KeyCtrlC)} {
		if got := nextKey(t, live); got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

// a sequence split over two writes is one key, and escape alone is escape
// once the wait lapses. anti-test: the lone escape is not reported before
// the wait, which here is a minute, and the split sequence is not reported
// as an escape and stray characters.
func TestLiveJoinsASplitSequenceAndReportsALoneEscape(t *testing.T) {
	live, master, _, _ := liveOn(t, LiveHooks{EscapeWait: aLongWait})
	master.WriteString("\x1b")
	master.WriteString("[B")
	if got := nextKey(t, live); got != key(KeyDown) {
		t.Errorf("a split sequence: %v", got)
	}
	master.WriteString("\x1b")
	select {
	case event := <-live.Events():
		t.Errorf("a lone escape was reported before the wait: %+v", event)
	case <-time.After(5 * aShortWait):
	}

	lone, loneMaster, _, _ := liveOn(t, LiveHooks{EscapeWait: aShortWait})
	loneMaster.WriteString("\x1b")
	if got := nextKey(t, lone); got != key(KeyEscape) {
		t.Errorf("a lone escape: %v", got)
	}
}

// control-right-bracket arrives as the host key and nothing else eats it.
func TestLiveReportsTheHostKey(t *testing.T) {
	live, master, _, _ := liveOn(t, LiveHooks{EscapeWait: aLongWait})
	master.WriteString("\x1d\x1b[\x1dz")
	for _, want := range []Key{key(KeyHost), key(KeyUnknown), key(KeyHost), char('z')} {
		if got := nextKey(t, live); got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

// the end of the input is an event, once, and the live view still closes.
func TestLiveReportsTheEndOfInput(t *testing.T) {
	live, master, _, _ := liveOn(t, LiveHooks{})
	master.WriteString("a")
	if got := nextKey(t, live); got != char('a') {
		t.Errorf("got %v", got)
	}
	master.Close()
	if event := next(t, live); event.Kind() != EventEnd {
		t.Errorf("got kind %d, want the end", event.Kind())
	}
}

// present draws what Draw draws, cut to the screen: here 6 by 4 cells from
// a picture of 8 by 6, with no clear, since the size has not changed.
func TestLivePresentClipsToTheScreen(t *testing.T) {
	live, master, _, _ := liveOn(t, LiveHooks{})
	cells := someCells(8, 6)
	var want strings.Builder
	Draw(&want, clipped(cells, startCols, startRows))
	if got := presented(t, live, master, cells, want.Len()); got != want.String() {
		t.Errorf("presented %d bytes, want %d", len(got), want.Len())
	}
	if strings.Contains(want.String(), "\x1b[5;1H") || strings.Contains(want.String(), "\x1b[2J") {
		t.Error("the expected frame is past the screen or clears it")
	}
}

// after a resize the next present clears first, once, and draws for the new
// size; the present after that does not clear. anti-test: nothing is
// cleared when the size has not changed, and no cells are presented that
// were not given.
func TestLivePresentClearsWhenTheSizeChanged(t *testing.T) {
	live, master, _, _ := liveOn(t, LiveHooks{})
	cells := someCells(12, 8)
	resize(t, master, laterCols, laterRows, 0, 0)
	syscall.Kill(syscall.Getpid(), syscall.SIGWINCH)
	if next(t, live).Kind() != EventResize {
		t.Fatal("no resize")
	}
	var drawn strings.Builder
	Draw(&drawn, clipped(cells, laterCols, laterRows))
	want := clearAll + drawn.String()
	if got := presented(t, live, master, cells, len(want)); got != want {
		t.Errorf("after a resize presented %d bytes, want %d", len(got), len(want))
	}
	if got := presented(t, live, master, cells, drawn.Len()); got != drawn.String() {
		t.Errorf("a second present cleared again or drew differently")
	}
}

// presented is what the master reads while live presents cells: the write
// is made on its own goroutine, since a pty holds only so many bytes.
func presented(t *testing.T, live *Live, master *os.File, cells *Cells, want int) string {
	t.Helper()
	failed := make(chan error, 1)
	go func() { failed <- live.Present(cells) }()
	got := screenOf(t, master, want)
	if err := <-failed; err != nil {
		t.Fatal(err)
	}
	return got
}

// anti-test: nothing to present is refused by name, and writes nothing.
func TestLivePresentRefusesNoCells(t *testing.T) {
	live, master, _, _ := liveOn(t, LiveHooks{})
	if err := live.Present(nil); err == nil || !strings.Contains(err.Error(), "give it cells") {
		t.Errorf("got %v", err)
	}
	slaveWrites(t, master)
}

// slaveWrites checks nothing is waiting on the master but a probe sent
// after: the test sends it from the other end.
func slaveWrites(t *testing.T, master *os.File) {
	t.Helper()
	fds := []unix.PollFd{{Fd: int32(master.Fd()), Events: unix.POLLIN}}
	if count, _ := unix.Poll(fds, 50); count != 0 {
		t.Error("something was written")
	}
}

// the status row is written on the last row, clipped to the width, in
// plain bytes; the picture above it is not touched by it.
func TestLiveStatusWritesTheLastRowClipped(t *testing.T) {
	live, master, _, _ := liveOn(t, LiveHooks{})
	if err := live.Status("frame 12 of 96, 41%"); err != nil {
		t.Fatal(err)
	}
	want := statusLine("frame 12 of 96, 41%", startCols, startRows)
	got := screenOf(t, master, len(want))
	if got != want || !strings.Contains(got, "\x1b[4;1H") || !strings.Contains(got, "frame ") ||
		strings.Contains(got, "frame 1") {
		t.Errorf("wrote %q", got)
	}
}

// the foot is written on the last rows, the last line on the last row,
// each as the status row is.
func TestLiveFootWritesTheLastRows(t *testing.T) {
	live, master, _, _ := liveOn(t, LiveHooks{})
	if err := live.Foot([]string{"a note", "frame 1 of 3"}); err != nil {
		t.Fatal(err)
	}
	want := statusLine("a note", startCols, startRows-1) +
		statusLine("frame 1 of 3", startCols, startRows)
	if got := screenOf(t, master, len(want)); got != want {
		t.Errorf("wrote %q, want %q", got, want)
	}
}

// anti-test: more lines than the screen has rows drop the oldest, and no
// line is written above the first row or below the last.
func TestFootLinesDropTheOldestPastTheRows(t *testing.T) {
	got := footLines([]string{"one", "two", "three"}, 80, 2)
	want := statusLine("two", 80, 1) + statusLine("three", 80, 2)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if strings.Contains(got, "one") || strings.Contains(got, "\x1b[0;1H") ||
		strings.Contains(got, "\x1b[3;1H") {
		t.Errorf("wrote outside the rows: %q", got)
	}
}

// a signal that stops the process gives the terminal back first, then exits
// 128 plus the signal. the test signals itself, and the exit is a channel.
// anti-test: before the signal nothing is given back.
func TestLiveGivesTheTerminalBackOnASignal(t *testing.T) {
	for _, sent := range []syscall.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP} {
		live, master, slave, exits := liveOn(t, LiveHooks{})
		slaveWrites(t, master)
		syscall.Kill(syscall.Getpid(), sent)
		select {
		case code := <-exits:
			if code != SignalExit+int(sent) {
				t.Errorf("signal %d: exit %d", sent, code)
			}
		case <-time.After(eventLimit):
			t.Fatalf("signal %d: no exit", sent)
		}
		if got := screenOf(t, master, len(givenBack)); got != givenBack {
			t.Errorf("signal %d: sent %q", sent, got)
		}
		_ = slave
		live.Close()
	}
}

// goroutinesIn is how many goroutines have name in their stacks.
func goroutinesIn(name string) int {
	buf := make([]byte, 1<<20)
	used := runtime.Stack(buf, true)
	return strings.Count(string(buf[:used]), name)
}

// after Close the reader and the follower have ended and the events are
// closed. anti-test: while open they are there, one each, so the count is
// not vacuous; and a caller that never read its events does not stop Close.
func TestLiveLeavesNothingRunningAfterClose(t *testing.T) {
	live, master, _, _ := liveOn(t, LiveHooks{})
	if goroutinesIn("(*Live).read") != 1 || goroutinesIn("(*Live).follow") != 1 {
		t.Fatalf("while open: %d readers, %d followers",
			goroutinesIn("(*Live).read"), goroutinesIn("(*Live).follow"))
	}
	for range 5 * eventRoom {
		master.WriteString("k")
	}
	closed := make(chan struct{})
	go func() { live.Close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(eventLimit):
		t.Fatal("Close did not return with events unread")
	}
	end := time.Now().Add(eventLimit)
	for goroutinesIn("(*Live).read")+goroutinesIn("(*Live).follow") != 0 &&
		time.Now().Before(end) {
		time.Sleep(time.Millisecond)
	}
	if count := goroutinesIn("(*Live).read") + goroutinesIn("(*Live).follow"); count != 0 {
		t.Errorf("%d goroutines after Close", count)
	}
	for range live.Events() {
	}
}

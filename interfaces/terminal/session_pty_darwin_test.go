package terminal

import (
	"errors"
	"os"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

// the sequences a session sends, from xterm's control sequences and not
// from the code. CSI ? 1049 h switches to the alternate screen, saving
// the cursor, and CSI ? 25 l hides the cursor; CSI 0 m resets the colour,
// CSI ? 25 h shows the cursor again and CSI ? 1049 l returns to the main
// screen.
const (
	takenOver = "\x1b[?1049h\x1b[?25l"
	givenBack = "\x1b[0m\x1b[?25h\x1b[?1049l"
)

// Open takes the terminal and End gives it back, on a pseudo-terminal
// made for the test. Open sends the sequence to take the screen over and
// leaves the keys unechoed and unbuffered (neither ECHO nor ICANON set);
// End sends the one that gives it back and restores the modes the
// terminal had, to the last bit.
//
// the second End does nothing: a byte sent after it is the next byte the
// master reads, with no second sequence before it.
func TestOpenTakesTheTerminalAndEndGivesItBack(t *testing.T) {
	master, slave := ptyPair(t)
	before := settings(t, slave)
	s, err := Open(slave, slave)
	if err != nil {
		t.Fatal(err)
	}
	if got := screenOf(t, master, len(takenOver)); got != takenOver {
		t.Errorf("Open sent %q, want %q", got, takenOver)
	}
	if during := settings(t, slave); during.Lflag&(unix.ECHO|unix.ICANON) != 0 {
		t.Errorf("local modes %#x while open: echo or line buffering is still on", during.Lflag)
	}
	s.End()
	if got := screenOf(t, master, len(givenBack)); got != givenBack {
		t.Errorf("End sent %q, want %q", got, givenBack)
	}
	if after := settings(t, slave); after != before {
		t.Errorf("modes after End are not the modes before Open:\n%+v\n%+v", after, before)
	}
	s.End()
	if _, err := slave.WriteString("Z"); err != nil {
		t.Fatal(err)
	}
	if got := screenOf(t, master, 1); got != "Z" {
		t.Errorf("after a second End the master read %q, want only the Z written after it", got)
	}
}

// Open wants a terminal at both ends and says so when it has not got
// one, by the same message in each case, before it has done anything: no
// bytes sent, the modes as they were.
func TestOpenRefusesAnythingButATerminalAtBothEnds(t *testing.T) {
	const refusal = "needs a terminal on both input and output"
	master, slave := ptyPair(t)
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	before := settings(t, slave)
	for _, c := range []struct {
		what    string
		in, out *os.File
	}{
		{"a pipe for both", r, w},
		{"a pipe for the input", r, slave},
		{"a pipe for the output", slave, w},
	} {
		s, err := Open(c.in, c.out)
		if s != nil || err == nil || err.Error() != refusal {
			t.Errorf("%s: got %v, %v; want no session and %q", c.what, s, err, refusal)
		}
	}
	if after := settings(t, slave); after != before {
		t.Errorf("a refused Open changed the modes:\n%+v\n%+v", after, before)
	}
	if _, err := slave.WriteString("Z"); err != nil {
		t.Fatal(err)
	}
	if got := screenOf(t, master, 1); got != "Z" {
		t.Errorf("a refused Open sent %q, want only the Z written after it", got)
	}
}

// a terminal that cannot be written to is not taken over, and is not
// left raw either. here the output is the same terminal opened for
// reading only, which is a terminal to every check and refuses a write
// with a bad file descriptor.
func TestOpenGivesBackAKeyboardItCouldNotUse(t *testing.T) {
	_, slave := ptyPair(t)
	before := settings(t, slave)
	s, err := Open(slave, reopen(t, slave))
	if s != nil || err == nil {
		t.Fatalf("got %v, %v; want no session and an error", s, err)
	}
	if !strings.Contains(err.Error(), "could not take over the screen") || !errors.Is(err, unix.EBADF) {
		t.Errorf("got %q, want it to say the screen could not be taken over, from the bad descriptor", err)
	}
	if after := settings(t, slave); after != before {
		t.Errorf("the terminal was left raw:\n%+v\n%+v", after, before)
	}
}

// the one byte the Check tests send after the call, to see that nothing
// else came before it.
const probe = "Z"

// Check passes a terminal at both ends and touches nothing. No bytes are
// sent. The modes stay as they were. The terminal is a pty the test made.
func TestCheckPassesTerminalsAndTouchesNothing(t *testing.T) {
	master, slave := ptyPair(t)
	before := settings(t, slave)
	if err := Check(slave, slave); err != nil {
		t.Fatalf("a terminal at both ends: %v", err)
	}
	if after := settings(t, slave); after != before {
		t.Errorf("Check changed the modes:\n%+v\n%+v", after, before)
	}
	if _, err := slave.WriteString(probe); err != nil {
		t.Fatal(err)
	}
	if got := screenOf(t, master, len(probe)); got != probe {
		t.Errorf("Check sent %q, want only the %q written after it", got, probe)
	}
}

// anti-test: Check refuses a pipe at either end in the words Open uses.
func TestCheckRefusesAnythingButATerminalAtBothEnds(t *testing.T) {
	const refusal = "needs a terminal on both input and output"
	_, slave := ptyPair(t)
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
		{"a pipe for both", pipeIn, pipeOut},
		{"a pipe for the input", pipeIn, slave},
		{"a pipe for the output", slave, pipeOut},
	} {
		if err := Check(row.in, row.out); err == nil || err.Error() != refusal {
			t.Errorf("%s: got %v; want %q", row.what, err, refusal)
		}
	}
}

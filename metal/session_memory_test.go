//go:build draw && darwin

package metal

import (
	"errors"
	"syscall"
	"testing"
)

// The numbers of the session memory tests.
const (
	memoryFloats = 8 << 20 // picked: a buffer of eight million numbers, 32 MB, big enough to see above the noise
	memoryBytes  = memoryFloats * 4
	memoryWarm   = 2  // picked: rounds run before the peak is first read, so one session's own memory is already counted
	memoryRounds = 24 // picked: 768 MB a buffer if nothing is handed back, small enough for a machine short of memory
	memoryRoom   = 4  // picked: how many buffers' worth the peak may rise over all the rounds; handed back, it rises by none
)

// peak is the most memory this process has ever had resident, in bytes,
// as the kernel counts it.
func peak(t *testing.T) int64 {
	t.Helper()
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		t.Fatal(err)
	}
	return usage.Maxrss
}

// round opens a session, makes so many buffers in it, runs a kernel over
// them when asked, and closes the session.
func round(t *testing.T, buffers int, run bool) {
	t.Helper()
	made, err := OpenSession(twoSteps, "add", "twice", "rowsum")
	if errors.Is(err, ErrNoDevice) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer made.Close()
	held := make([]Buffer, buffers)
	for index := range held {
		if held[index], err = made.Buffer(memoryFloats); err != nil {
			t.Fatal(err)
		}
	}
	if !run {
		return
	}
	if err := made.Dispatch("add", memoryFloats, 1, nil, held[0], held[1]); err != nil {
		t.Fatal(err)
	}
	if err := made.Commit(); err != nil {
		t.Fatal(err)
	}
}

// grown is how far the process's peak memory rises over the rounds,
// after the warm rounds have been counted in.
func grown(t *testing.T, buffers int, run bool) int64 {
	t.Helper()
	for range memoryWarm {
		round(t, buffers, run)
	}
	before := peak(t)
	for range memoryRounds {
		round(t, buffers, run)
	}
	return peak(t) - before
}

// A closed session hands its buffers back: opening one, making a buffer
// and closing it, over and over, does not grow the process. The
// omnipositor's Metal ocean opens nine sessions a frame, and on
// 2026-11-11 a film of 120 frames took 22 GB.
func TestAClosedSessionHandsItsBufferBack(t *testing.T) {
	rose := grown(t, 1, false)
	t.Logf("%d rounds of one %d MB buffer, never run: the peak rose %d MB",
		memoryRounds, memoryBytes>>20, rose>>20)
	if rose > memoryRoom*memoryBytes {
		t.Errorf("the peak rose %d MB over %d rounds: closed sessions keep their buffers",
			rose>>20, memoryRounds)
	}
}

// The same when the buffers were used: a session that ran a kernel over
// its two buffers and was closed hands both back.
func TestAClosedSessionThatRanHandsItsBuffersBack(t *testing.T) {
	rose := grown(t, 2, true)
	t.Logf("%d rounds of two %d MB buffers, run once: the peak rose %d MB",
		memoryRounds, memoryBytes>>20, rose>>20)
	if rose > memoryRoom*memoryBytes {
		t.Errorf("the peak rose %d MB over %d rounds: closed sessions that ran keep their buffers",
			rose>>20, memoryRounds)
	}
}

// And a session with no buffer at all: opening and closing compiles the
// source each time, and that too is handed back.
func TestAClosedSessionWithNoBufferLeavesNothing(t *testing.T) {
	rose := grown(t, 0, false)
	t.Logf("%d rounds of no buffer: the peak rose %d MB", memoryRounds, rose>>20)
	if rose > memoryRoom*memoryBytes {
		t.Errorf("the peak rose %d MB over %d rounds of opening and closing alone",
			rose>>20, memoryRounds)
	}
}

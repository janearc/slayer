//go:build draw && darwin

package metal

import (
	"context"
	"encoding/binary"
	"errors"
	"testing"
	"time"
)

// spinProbe runs one thread through as many dependent steps as the input
// says, so a command lasts as long as the count asks and cannot be done
// early by the compiler.
const spinProbe = `
#include <metal_stdlib>
using namespace metal;
kernel void spin(device const uint *count [[buffer(0)]],
                 device float *out [[buffer(1)]],
                 constant uint2 &origin [[buffer(2)]],
                 uint2 at [[thread_position_in_grid]]) {
	uint x = at.x + 1u;
	for (uint step = 0; step < count[0]; step++) {
		x = x * 1664525u + 1013904223u;
	}
	out[at.x] = float(x & 255u);
}
`

// the counts of spinProbe: one that is over at once, and one the device
// takes many seconds over, picked by the step's cost on the machines the
// checks run on, so a stop lands long before the command ends. The
// command is left running when a check ends; it ends by itself.
const (
	spinQuick = 1
	spinLong  = 3_000_000_000
)

// stopAfter is when the long checks cancel their context, and stopWithin
// is how long after that a stopped wait may take to return: the look
// between polls is under a millisecond, so this is generous, and far
// short of the command's own time.
const (
	stopAfter  = 200 * time.Millisecond
	stopWithin = 5 * time.Second
)

// spinStream is a stream of the spin kernel over one pixel, counting
// count steps, and its kernel.
func spinStream(t *testing.T, count uint32) (*Kernel, *GridStream) {
	t.Helper()
	kernel, err := Open(spinProbe, "spin")
	requireDevice(t, err)
	t.Cleanup(kernel.Close)
	stream, err := kernel.OpenGrid(1, 1, 1, 1,
		[][]byte{binary.LittleEndian.AppendUint32(nil, count)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(stream.Close)
	return kernel, stream
}

// TestACancelledContextEndsAWaitOnARunningBand cancels the context while
// the device runs a band of many seconds: the draw returns the context's
// error well before the band could end, and the kernel is not used again.
func TestACancelledContextEndsAWaitOnARunningBand(t *testing.T) {
	_, stream := spinStream(t, spinLong)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	release := StopOn(ctx)
	defer release()
	time.AfterFunc(stopAfter, cancel)
	began := time.Now()
	err := stream.DrawBands(func(int, int, []float32) error { return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("a cancelled draw gave %v, want the context's error", err)
	}
	if took := time.Since(began); took > stopAfter+stopWithin {
		t.Errorf("the wait ended %v after the cancel, over %v",
			took-stopAfter, stopWithin)
	}
	release()
	if err := stream.DrawBands(func(int, int, []float32) error {
		return nil
	}); err == nil {
		t.Error("a kernel whose wait was stopped drew again")
	}
}

// TestADeadlineEndsAWaitWithTheDeadlineError is a context that ends by
// its deadline: the error is the deadline's, not a cancel.
func TestADeadlineEndsAWaitWithTheDeadlineError(t *testing.T) {
	_, stream := spinStream(t, spinLong)
	ctx, cancel := context.WithTimeout(context.Background(), stopAfter)
	defer cancel()
	defer StopOn(ctx)()
	err := stream.DrawBands(func(int, int, []float32) error { return nil })
	if !errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled) {
		t.Fatalf("a draw past its deadline gave %v", err)
	}
}

// TestAContextAlreadyCancelledStopsTheFirstWait gives StopOn a context
// that is done before the draw begins.
func TestAContextAlreadyCancelledStopsTheFirstWait(t *testing.T) {
	_, stream := spinStream(t, spinLong)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	defer StopOn(ctx)()
	err := stream.DrawBands(func(int, int, []float32) error { return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("a draw under a cancelled context gave %v", err)
	}
}

// TestAContextNotDoneStopsNothing draws a quick band under a context
// that lives through it: the band is drawn and the picture is right.
func TestAContextNotDoneStopsNothing(t *testing.T) {
	_, stream := spinStream(t, spinQuick)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer StopOn(ctx)()
	var drawn int
	err := stream.DrawBands(func(_, down int, _ []float32) error {
		drawn += down
		return nil
	})
	if err != nil || drawn != 1 {
		t.Fatalf("a draw under a live context drew %d rows (%v)", drawn, err)
	}
}

// TestLettingGoOfAStoppedContextLetsLaterDrawsWait cancels a context,
// lets it go, and draws on a new kernel: the stop does not outlive its
// release, and a nil context stops nothing.
func TestLettingGoOfAStoppedContextLetsLaterDrawsWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	release := StopOn(ctx)
	cancel()
	release()
	StopOn(nil)()
	_, stream := spinStream(t, spinQuick)
	if err := stream.DrawBands(func(int, int, []float32) error {
		return nil
	}); err != nil {
		t.Fatalf("a draw after the stop was let go gave %v", err)
	}
}

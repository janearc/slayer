//go:build draw && darwin

package metal

import (
	"encoding/binary"
	"errors"
	"strings"
	"testing"
	"time"
)

// spinning is a kernel that adds to a number as many times as the
// params say, so a test sets how long one command runs on the device.
const spinning = `
#include <metal_stdlib>
using namespace metal;
kernel void spin(device const float *in [[buffer(0)]],
                 device float *out [[buffer(1)]],
                 constant uint &times [[buffer(2)]],
                 uint at [[thread_position_in_grid]]) {
	float held = in[at];
	for (uint k = 0; k < times; k++) {
		held = sin(held) + cos(held);
	}
	out[at] = held;
}
kernel void spun(device float *into [[buffer(0)]],
                 constant uint &times [[buffer(1)]],
                 uint at [[thread_position_in_grid]]) {
	float held = into[at];
	for (uint k = 0; k < times; k++) {
		held = sin(held) + cos(held);
	}
	into[at] = held;
}
`

// brief is a few trips round the loop, over at once; long is trips enough
// to keep the device busy for seconds, measured with
// `go test -run TestALongCommand -v ./metal/`.
const (
	brief = 1000
	long  = 60000000
)

// trips is the params bytes for a number of trips round the loop.
func trips(count uint32) []byte {
	return binary.LittleEndian.AppendUint32(nil, count)
}

// spinKernel opens the spinning kernel with a limit, or skips with no gpu.
func spinKernel(t *testing.T, limit time.Duration) *Kernel {
	t.Helper()
	kernel, err := Open(spinning, "spin")
	requireDevice(t, err)
	t.Cleanup(kernel.Close)
	kernel.SetLimit(limit)
	return kernel
}

// a command that runs longer than the limit comes back as an error that
// names the kernel and the limit, about when the limit is up, and the
// kernel is not used again.
func TestACommandOverTheLimitIsAnErrorNearTheLimit(t *testing.T) {
	limit := 300 * time.Millisecond
	kernel := spinKernel(t, limit)
	begun := time.Now()
	_, err := kernel.Points(make([][3]float32, 64), trips(long))
	waited := time.Since(begun)
	if err == nil {
		t.Fatalf("a command of %d trips finished in %v under a limit of %v; want a timeout error",
			long, waited, limit)
	}
	for _, want := range []string{`kernel "spin"`, "did not complete", "0.3 s", "busy"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the timeout error is %q; want it to say %q", err, want)
		}
	}
	if waited < limit || waited > limit+time.Second {
		t.Errorf("the timeout came after %v; want about %v", waited, limit)
	}
	_, again := kernel.Points(make([][3]float32, 64), trips(brief))
	if again == nil || !strings.Contains(again.Error(), "do not reuse") {
		t.Errorf("a kernel after a timeout gave %v; want a refusal saying not to reuse it", again)
	}
}

// a command well inside the limit is not changed by it, and the
// default limit is not a small one.
func TestACommandInsideTheLimitIsUnaffected(t *testing.T) {
	kernel := spinKernel(t, defaultLimit)
	points := make([][3]float32, 64)
	for k := range points {
		points[k][0] = float32(k)
	}
	got, err := kernel.Points(points, trips(brief))
	if err != nil {
		t.Fatalf("a brief command under the default limit of %v: %v", defaultLimit, err)
	}
	if len(got) != len(points) {
		t.Fatalf("got %d numbers; want %d", len(got), len(points))
	}
	for k := range got {
		// each trip sets the number to sin plus cos of it, which is never
		// outside plus or minus the square root of two
		if !(got[k] >= -1.5 && got[k] <= 1.5) {
			t.Errorf("number %d is %v; want within 1.5", k, got[k])
		}
	}
}

// banded is a grid kernel that spins once in the first band, and as many
// times as its one input says in every band after it.
const banded = `
#include <metal_stdlib>
using namespace metal;
kernel void banded(device const uint *times [[buffer(0)]],
                   device float4 *out [[buffer(1)]],
                   constant uint2 &origin [[buffer(2)]],
                   uint2 at [[thread_position_in_grid]],
                   uint2 grid [[threads_per_grid]]) {
	float held = float(at.x);
	uint count = origin.y > 0 ? times[0] : 1;
	for (uint k = 0; k < count; k++) {
		held = sin(held) + cos(held);
	}
	out[at.y * grid.x + at.x] = float4(held, at.y + origin.y, 0, 0);
}
`

// a grid band that runs over the limit is an error that says which
// rows it was drawing, and a band inside the limit draws. the first
// band is quick and the second is long, so the second is the one named.
func TestABandOverTheLimitNamesItsRows(t *testing.T) {
	limit := 300 * time.Millisecond
	kernel, err := Open(banded, "banded")
	requireDevice(t, err)
	t.Cleanup(kernel.Close)
	kernel.SetLimit(limit)
	// the bands are fixed at four rows, so the long one is rows 4 to 7
	kernel.FixBands(4)
	if _, err := kernel.Grid(64, 8, 4, 4, [][]byte{trips(brief)}); err != nil {
		t.Fatalf("a brief band under a limit of %v: %v", limit, err)
	}
	begun := time.Now()
	_, err = kernel.Grid(64, 8, 4, 4, [][]byte{trips(long)})
	waited := time.Since(begun)
	if err == nil {
		t.Fatalf("a band of %d trips finished in %v under a limit of %v; want a timeout error",
			long, waited, limit)
	}
	for _, want := range []string{`kernel "banded"`, "rows 4 to 7", "did not complete", "busy"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the band timeout error is %q; want it to say %q", err, want)
		}
	}
	if waited < limit || waited > limit+time.Second {
		t.Errorf("the band timeout came after %v; want about %v", waited, limit)
	}
}

// a session batch that runs over the limit is an error saying so, and the
// session is not used again.
func TestASessionBatchOverTheLimitIsAnError(t *testing.T) {
	limit := 300 * time.Millisecond
	made, err := OpenSession(spinning, "spun")
	requireDevice(t, err)
	t.Cleanup(made.Close)
	made.SetLimit(limit)
	into, err := made.Buffer(64)
	if err != nil {
		t.Fatal(err)
	}
	if err := made.Dispatch("spun", 64, 1, trips(brief), into); err != nil {
		t.Fatal(err)
	}
	if err := made.Commit(); err != nil {
		t.Fatalf("a brief batch under a limit of %v: %v", limit, err)
	}
	if err := made.Dispatch("spun", 64, 1, trips(long), into); err != nil {
		t.Fatal(err)
	}
	begun := time.Now()
	err = made.Commit()
	waited := time.Since(begun)
	if err == nil {
		t.Fatalf("a batch of %d trips finished in %v under a limit of %v; want a timeout error",
			long, waited, limit)
	}
	for _, want := range []string{"session batch", "did not complete", "busy"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the session timeout error is %q; want it to say %q", err, want)
		}
	}
	if waited > limit+time.Second {
		t.Errorf("the session timeout came after %v; want about %v", waited, limit)
	}
	again := made.Dispatch("spun", 64, 1, trips(brief), into)
	if again == nil || !strings.Contains(again.Error(), "do not reuse") {
		t.Errorf("a session after a timeout took a dispatch (%v); want a refusal saying not to reuse it", again)
	}
}

// a band drawn ahead that outruns the limit is an error from the drawing
// that left it in flight, beside its consumer's own, and not a silence.
func TestABandDrawnAheadThatTimesOutIsReported(t *testing.T) {
	limit := 300 * time.Millisecond
	kernel, err := Open(banded, "banded")
	requireDevice(t, err)
	t.Cleanup(kernel.Close)
	kernel.SetLimit(limit)
	// the bands are fixed at four rows, so the one drawn ahead is rows 4
	// to 7
	kernel.FixBands(4)
	stream, err := kernel.OpenGrid(64, 8, 4, 4, [][]byte{trips(long)})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	stop := errors.New("stop after the first band")
	first := func(int, int, []float32) error { return stop }
	err = stream.DrawBands(first)
	if !errors.Is(err, stop) {
		t.Fatalf("the first band came back as %v; want the consumer's own error", err)
	}
	for _, want := range []string{"rows 4 to 7", "did not complete"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the drawing's error is %q; want it to say %q", err, want)
		}
	}
}

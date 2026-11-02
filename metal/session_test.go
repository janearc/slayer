//go:build darwin

package metal

import (
	"errors"
	"testing"
)

// two kernels, one adding a buffer into another, one doubling it, in
// one batch, in order
const twoSteps = `
#include <metal_stdlib>
using namespace metal;
kernel void add(device float *into [[buffer(0)]],
                device const float *from [[buffer(1)]],
                uint at [[thread_position_in_grid]]) {
	into[at] += from[at];
}
kernel void twice(device float *into [[buffer(0)]],
                  uint at [[thread_position_in_grid]]) {
	into[at] *= 2;
}
kernel void rowsum(device float *into [[buffer(0)]],
                   device const float *from [[buffer(1)]],
                   uint row [[threadgroup_position_in_grid]],
                   uint at [[thread_position_in_threadgroup]],
                   uint wide [[threads_per_threadgroup]]) {
	threadgroup float held[64];
	held[at] = from[row * wide + at];
	threadgroup_barrier(mem_flags::mem_threadgroup);
	if (at == 0) {
		float sum = 0;
		for (uint k = 0; k < wide; k++) { sum += held[k]; }
		into[row] = sum;
	}
}
`

// session opens the test's session, or skips with no gpu.
func session(t *testing.T) *Session {
	t.Helper()
	made, err := OpenSession(twoSteps, "add", "twice", "rowsum")
	if errors.Is(err, ErrNoDevice) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(made.Close)
	return made
}

// a batch runs its dispatches in the order given, on buffers that keep
// what the last batch left and what go wrote between.
func TestASessionKeepsItsBuffersAcrossBatches(t *testing.T) {
	made := session(t)
	into, err := made.Buffer(8)
	if err != nil {
		t.Fatal(err)
	}
	from, err := made.Buffer(8)
	if err != nil {
		t.Fatal(err)
	}
	for k := range from.Numbers() {
		from.Numbers()[k] = float32(k)
	}
	for range 2 {
		if err := made.Dispatch("add", 8, 1, nil, into, from); err != nil {
			t.Fatal(err)
		}
		if err := made.Dispatch("twice", 8, 1, nil, into); err != nil {
			t.Fatal(err)
		}
		if err := made.Commit(); err != nil {
			t.Fatal(err)
		}
	}
	for k, got := range into.Numbers() {
		if want := float32(6 * k); got != want {
			t.Errorf("number %d: %g, want %g", k, got, want)
		}
	}
}

// grouped, each group is a row whose threads share their memory.
func TestAGroupIsARow(t *testing.T) {
	made := session(t)
	into, _ := made.Buffer(3)
	from, _ := made.Buffer(3 * 16)
	for k := range from.Numbers() {
		from.Numbers()[k] = 1
	}
	if err := made.Grouped("rowsum", 16, 3, nil, into, from); err != nil {
		t.Fatal(err)
	}
	if err := made.Commit(); err != nil {
		t.Fatal(err)
	}
	for row, got := range into.Numbers() {
		if got != 16 {
			t.Errorf("row %d sums to %g, want 16", row, got)
		}
	}
}

// a kernel the source does not hold is refused at the dispatch.
func TestAnUnknownKernelIsRefused(t *testing.T) {
	made := session(t)
	if err := made.Dispatch("none", 1, 1, nil); err == nil {
		t.Error("a kernel that is not there was dispatched")
	}
}

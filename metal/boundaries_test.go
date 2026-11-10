//go:build darwin

package metal

import (
	"encoding/binary"
	"errors"
	"math"
	"strings"
	"testing"
)

const fillGrid = `
#include <metal_stdlib>
using namespace metal;
kernel void fill_grid(device float *out [[buffer(0)]],
 constant uint2 &origin [[buffer(1)]],
 uint2 at [[thread_position_in_grid]], uint2 grid [[threads_per_grid]]) {
 out[at.y*grid.x+at.x] = (at.y+origin.y)*grid.x+at.x;
}`

func TestClosedKernelsAndSessionsRefuseWork(t *testing.T) {
	kernel := &Kernel{}
	if _, err := kernel.Points(nil, nil); err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("closed points: %v", err)
	}
	if _, err := kernel.Grid(1, 1, 1, 1, nil); err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("closed grid: %v", err)
	}
	if kernel.GPUTime() != 0 {
		t.Fatal("closed kernel retained GPU time")
	}
	session := &Session{}
	if _, err := session.Buffer(1); err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("closed buffer: %v", err)
	}
	if err := session.Commit(); err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("closed commit: %v", err)
	}
	if session.GPUTime() != 0 {
		t.Fatal("closed session retained GPU time")
	}
}

func TestGridChecksDimensionsBeforeAllocationAndNarrowsItsBands(t *testing.T) {
	kernel, err := Open(fillGrid, "fill_grid")
	requireDevice(t, err)
	defer kernel.Close()
	for _, dimensions := range [][3]int{{-1, 2, 1}, {2, -1, 1}, {0, 2, 1}, {2, 0, 1}} {
		got, err := kernel.Grid(dimensions[0], dimensions[1], 1, dimensions[2], nil)
		if err != nil || len(got) != 0 {
			t.Fatalf("empty grid %v: %v, %v", dimensions, got, err)
		}
	}
	if _, err := kernel.Grid(1, 1, 1, 0, nil); !errors.Is(err, errNoNumbers) {
		t.Fatalf("no output channels: %v", err)
	}
	for _, dimensions := range [][3]int{{math.MaxInt, 1, 1}, {1, math.MaxInt, 1}, {1, 1, math.MaxInt}, {1 << 30, 1 << 30, 16}} {
		if _, err := kernel.Grid(dimensions[0], dimensions[1], 1, dimensions[2], nil); !errors.Is(err, errGridSize) {
			t.Fatalf("oversized output %v: %v", dimensions, err)
		}
	}
	for _, rows := range []int{0, -1, 1, 2, math.MaxInt} {
		got, err := kernel.Grid(2, 3, rows, 1, nil)
		if err != nil {
			t.Fatal(err)
		}
		for i, value := range got {
			if value != float32(i) {
				t.Fatalf("band %d pixel %d: %g", rows, i, value)
			}
		}
	}
}

func TestNativeCompilerErrorsAreNotMissingDevices(t *testing.T) {
	healthy, err := Open(first, "first")
	requireDevice(t, err)
	defer healthy.Close()
	if got, err := healthy.Points(nil, nil); err != nil || len(got) != 0 {
		t.Fatalf("empty points: %v, %v", got, err)
	}
	for _, open := range []func() error{
		func() error { _, err := Open("invalid metal source", "missing"); return err },
		func() error { _, err := OpenSession("invalid metal source", "missing"); return err },
		func() error { _, err := OpenSession(twoSteps, "missing"); return err },
	} {
		if err := open(); err == nil || errors.Is(err, ErrNoDevice) {
			t.Fatalf("compiler failure classified as no device: %v", err)
		}
	}
}

func TestSessionCarriesParametersAndNativeDispatchRefusals(t *testing.T) {
	const source = `#include <metal_stdlib>
using namespace metal;
kernel void plus(device float *out [[buffer(0)]],
 constant float &value [[buffer(1)]], uint i [[thread_position_in_grid]]) {
 out[i] += value;
}`
	made, err := OpenSession(source, "plus")
	requireDevice(t, err)
	defer made.Close()
	for _, count := range []int{-1, math.MaxInt} {
		if _, err := made.Buffer(count); err == nil || !strings.Contains(err.Error(), "invalid session buffer size") {
			t.Fatalf("invalid buffer count %d: %v", count, err)
		}
	}
	if _, err := made.Buffer(math.MaxInt / 8); err == nil || !strings.Contains(err.Error(), "buffer exceeds device limit") {
		t.Fatalf("native buffer limit: %v", err)
	}
	buffer, err := made.Buffer(3)
	if err != nil {
		t.Fatal(err)
	}
	params := binary.LittleEndian.AppendUint32(nil, math.Float32bits(7))
	if err := made.Dispatch("plus", 3, 1, params, buffer); err != nil {
		t.Fatal(err)
	}
	if err := made.Commit(); err != nil {
		t.Fatal(err)
	}
	for _, value := range buffer.Numbers() {
		if value != 7 {
			t.Fatalf("session parameters lost: %v", buffer.Numbers())
		}
	}
	if made.GPUTime() <= 0 {
		t.Fatal("committed session measured no GPU time")
	}
	// The C bridge checks its own kernel index independently of Go's map.
	// A damaged map must return that actual native refusal, not success.
	made.kernels["plus"] = -1
	if err := made.Dispatch("plus", 3, 1, params, buffer); err == nil ||
		!strings.Contains(err.Error(), "no such kernel") {
		t.Fatalf("native dispatch cause lost: %v", err)
	}
	made.kernels["plus"] = 0
	if err := made.Commit(); err != nil {
		t.Fatalf("refusal left an invalid command pending: %v", err)
	}
}

func TestAccelerationPlanRejectsNonfiniteVerticesAndMissingPlans(t *testing.T) {
	data := triangleData()
	binary.LittleEndian.PutUint32(data, math.Float32bits(float32(math.NaN())))
	if _, err := NewAcceleration(data, 16); err == nil || !strings.Contains(err.Error(), "nonfinite triangle") {
		t.Fatalf("nonfinite vertex: %v", err)
	}
	for _, plan := range []*Acceleration{nil, {}} {
		if _, err := plan.Triangles(0, 0); err == nil || !strings.Contains(err.Error(), "no acceleration plan") {
			t.Fatalf("missing triangle plan: %v", err)
		}
		if _, err := plan.Instances(); err == nil || !strings.Contains(err.Error(), "no acceleration plan") {
			t.Fatalf("missing instance plan: %v", err)
		}
	}
}

func TestAcceleratedKernelRequiresGridRatherThanPoints(t *testing.T) {
	kernel, err := Open(acceleratedProbe, "probe")
	requireDevice(t, err)
	defer kernel.Close()
	plan, err := NewAcceleration(nil, 16)
	if err != nil {
		t.Fatal(err)
	}
	root, err := plan.Instances()
	if err != nil {
		t.Fatal(err)
	}
	if err := kernel.Accelerate(plan, root); err != nil {
		t.Fatal(err)
	}
	if _, err := kernel.Points([][3]float32{{}}, nil); err == nil ||
		!strings.Contains(err.Error(), "requires Grid") {
		t.Fatalf("native Points cause: %v", err)
	}
}

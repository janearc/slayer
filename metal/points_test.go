//go:build darwin

package metal

import (
	"errors"
	"testing"
)

// a points kernel that reads no parameters: it gives back each point's
// first number
const first = `
#include <metal_stdlib>
using namespace metal;
kernel void first(device const float *points [[buffer(0)]],
                  device float *out [[buffer(1)]],
                  uint at [[thread_position_in_grid]]) {
	out[at] = points[at * 3];
}
`

func TestAKernelWithNoParamsRunsOverPoints(t *testing.T) {
	kernel, err := Open(first, "first")
	if errors.Is(err, ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	points := [][3]float32{{1, 9, 9}, {2, 9, 9}, {3, 9, 9}}
	out, err := kernel.Points(points, nil)
	if err != nil {
		t.Fatal(err)
	}
	for index, point := range points {
		if out[index] != point[0] {
			t.Fatalf("point %d gave %v, want %v", index, out[index],
				point[0])
		}
	}
}

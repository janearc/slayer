//go:build darwin

package metal

import (
	"encoding/binary"
	"errors"
	"math"
	"testing"
	"time"
)

// a kernel that writes each pixel's own column and row, the band's
// first row, and an input it was given, so a test can see that every
// pixel was reached once, in its place
const where = `
#include <metal_stdlib>
using namespace metal;
kernel void where(device const float *given [[buffer(0)]],
                  device float4 *out [[buffer(1)]],
                  constant uint2 &origin [[buffer(2)]],
                  uint2 at [[thread_position_in_grid]],
                  uint2 grid [[threads_per_grid]]) {
	uint row = at.y + origin.y;
	out[at.y * grid.x + at.x] = float4(at.x, row, origin.y, given[0]);
}
`

// expected is what the kernel writes at a pixel: its column, its row,
// its band's first row, and the input it was given
func expected(column, row, rows int) [4]float32 {
	first := row / rows * rows
	return [4]float32{float32(column), float32(row), float32(first), 7}
}

func TestEveryPixelIsReachedOnceInItsPlace(t *testing.T) {
	kernel, err := Open(where, "where")
	if errors.Is(err, ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	given := binary.LittleEndian.AppendUint32(nil, math.Float32bits(7))
	width, height, rows := 13, 10, 4
	out, err := kernel.Grid(width, height, rows, 4, [][]byte{given})
	if err != nil {
		t.Fatal(err)
	}
	for row := range height {
		for column := range width {
			pixel := [4]float32(out[(row*width+column)*4:][:4])
			if want := expected(column, row, rows); pixel != want {
				t.Fatalf("pixel %d,%d is %v, want %v", column,
					row, pixel, want)
			}
		}
	}
}

// a run says how long the gpu spent on it, from the system's own timestamps on
// the commands: more than zero, and no more than the time the run took by the
// clock, since the gpu cannot work longer than the caller waited. a kernel that
// has not run has spent none.
func TestAGridSaysHowLongTheGPUSpent(t *testing.T) {
	kernel, err := Open(where, "where")
	if errors.Is(err, ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	if spent := kernel.GPUTime(); spent != 0 {
		t.Errorf("a kernel that has not run has spent %v", spent)
	}
	given := binary.LittleEndian.AppendUint32(nil, math.Float32bits(7))
	start := time.Now()
	_, err = kernel.Grid(2000, 2000, 500, 4, [][]byte{given})
	if err != nil {
		t.Fatal(err)
	}
	waited := time.Since(start)
	spent := kernel.GPUTime()
	if spent <= 0 || spent > waited {
		t.Errorf("the gpu spent %v of the %v the run took", spent,
			waited)
	}
	if _, err := kernel.Grid(2, 2, 2, 4, [][]byte{given}); err != nil {
		t.Fatal(err)
	}
	if again := kernel.GPUTime(); again <= spent {
		t.Errorf("a small run after a large left %v, from %v", again,
			spent)
	}
}

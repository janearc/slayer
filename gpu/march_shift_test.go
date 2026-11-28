//go:build draw && darwin

package gpu

import (
	"testing"

	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/trace"
)

// shiftKernel is what the kernel's shift source needs before it.
const shiftKernel = "#include <metal_stdlib>\nusing namespace metal;\n"

// shiftSide is how many fine pixels a side of the grid the test asks.
const shiftSide = 64

// The Metal kernel's shift for a fine pixel is trace's, to the last bit:
// both are the top 24 bits of one integer hash over 2^24, which float32
// and float64 hold exactly. A grid of pixels, including large ones, and
// none may differ.
func TestTheKernelsMarchShiftIsTracesToTheLastBit(t *testing.T) {
	source := shiftKernel + shiftSource + `
kernel void shifts(device const float *p [[buffer(0)]],
	device float *out [[buffer(1)]], uint i [[thread_position_in_grid]]) {
	out[i] = march_shift(uint2(uint(p[3 * i]), uint(p[3 * i + 1])));
}
`
	kernel, err := metal.Open(source, "shifts")
	if err != nil {
		t.Skipf("no Metal device to ask: %v", err)
	}
	defer kernel.Close()
	var asked [][3]float32
	var pixels [][2]uint32
	for down := range shiftSide {
		for across := range shiftSide {
			// float32 holds whole numbers to 2^24 exactly: sweep from
			// the corner and from far out
			for _, base := range []uint32{0, 1 << 20} {
				x, y := base+uint32(across), base+uint32(down)
				asked = append(asked, [3]float32{float32(x), float32(y), 0})
				pixels = append(pixels, [2]uint32{x, y})
			}
		}
	}
	got, err := kernel.Points(asked, nil)
	if err != nil {
		t.Fatal(err)
	}
	for index, pixel := range pixels {
		want := trace.MarchShift(pixel[0], pixel[1])
		if float64(got[index]) != want {
			t.Fatalf("pixel %v: the kernel's shift %v, trace's %v", pixel,
				got[index], want)
		}
	}
}

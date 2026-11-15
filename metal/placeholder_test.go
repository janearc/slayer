//go:build draw && darwin

package metal

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// reading is a grid kernel that declares its first input as a run of
// sixteen-byte values and reads the first, as a scene with no baked
// volumes does through its baked buffer.
const reading = `
#include <metal_stdlib>
using namespace metal;
kernel void reading(device const float4 *given [[buffer(0)]],
                    device float4 *out [[buffer(1)]],
                    constant uint2 &origin [[buffer(2)]],
                    uint2 at [[thread_position_in_grid]],
                    uint2 grid [[threads_per_grid]]) {
	out[at.y * grid.x + at.x] = float4(at.x, at.y + origin.y, given[0].x, 0);
}
`

// drawEmpty draws a grid with its one input empty, then updates the input
// to empty again, which is what a stream kept between pictures does.
func drawEmpty(t *testing.T) {
	t.Helper()
	kernel, err := Open(reading, "reading")
	requireDevice(t, err)
	defer kernel.Close()
	if _, err := kernel.Grid(4, 4, 2, 4, [][]byte{nil}); err != nil {
		t.Fatal(err)
	}
	stream, err := kernel.OpenGrid(4, 4, 2, 4, [][]byte{{1}})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	if err := stream.Update(0, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := stream.Draw(); err != nil {
		t.Fatal(err)
	}
}

// an input with no bytes is bound as a buffer as long as the kernel
// declares for it, so Metal's API validation does not stop the process.
// The draw runs in a child process with validation on, since a failed
// validation aborts its process.
func TestAnEmptyInputIsAsLongAsTheKernelDeclares(t *testing.T) {
	if os.Getenv("SLAYER_METAL_EMPTY_INPUT") != "" {
		drawEmpty(t)
		return
	}
	child := exec.Command(os.Args[0],
		"-test.run=^TestAnEmptyInputIsAsLongAsTheKernelDeclares$", "-test.v")
	child.Env = append(os.Environ(), "MTL_DEBUG_LAYER=1",
		"SLAYER_METAL_EMPTY_INPUT=1")
	out, err := child.CombinedOutput()
	if err != nil {
		t.Fatalf("an empty input with API validation on: %v\n%s", err, out)
	}
	if strings.Contains(string(out), "SKIP") {
		t.Skip("no gpu here")
	}
}

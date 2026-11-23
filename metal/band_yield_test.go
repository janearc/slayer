//go:build draw && darwin

package metal

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"
)

// globalRows writes each pixel's column, its row in the whole picture, a
// number spun from both and the input it was given: nothing of its band,
// as no kernel that draws a picture may.
const globalRows = `
#include <metal_stdlib>
using namespace metal;
kernel void global_rows(device const float *given [[buffer(0)]],
                        device float4 *out [[buffer(1)]],
                        constant uint2 &origin [[buffer(2)]],
                        uint2 at [[thread_position_in_grid]],
                        uint2 grid [[threads_per_grid]]) {
	uint row = at.y + origin.y;
	float held = float(at.x * 131 + row);
	for (uint k = 0; k < 64; k++) {
		held = sin(held) + cos(held);
	}
	out[at.y * grid.x + at.x] = float4(at.x, row, held, given[0]);
}
`

// the yield fixture: a picture tall enough for many bands of each kind
const (
	yieldWidth, yieldHeight, yieldMost = 37, 203, 64
)

// yieldKernel opens the global-rows kernel, or skips with no gpu.
func yieldKernel(t *testing.T) *Kernel {
	t.Helper()
	kernel, err := Open(globalRows, "global_rows")
	requireDevice(t, err)
	t.Cleanup(kernel.Close)
	return kernel
}

// yieldGiven is the fixture's one input, the number seven.
func yieldGiven() [][]byte {
	return [][]byte{binary.LittleEndian.AppendUint32(nil,
		math.Float32bits(7))}
}

// drawnWith is the fixture drawn by Grid with every band fixed rows
// tall, or as Grid draws it when rows is nought.
func drawnWith(t *testing.T, kernel *Kernel, rows int) []float32 {
	t.Helper()
	kernel.FixBands(rows)
	out, err := kernel.Grid(yieldWidth, yieldHeight, yieldMost, 4,
		yieldGiven())
	if err != nil {
		t.Fatalf("bands of %d rows: %v", rows, err)
	}
	return out
}

// the same picture comes back bit for bit from fixed bands of the most
// rows, from Grid's own bands, and from bands of one and of three rows.
func TestAGridIsTheSameWhateverItsBands(t *testing.T) {
	kernel := yieldKernel(t)
	want := drawnWith(t, kernel, yieldMost)
	for _, rows := range []int{0, 1, 3} {
		if got := drawnWith(t, kernel, rows); !slices.Equal(got, want) {
			t.Errorf("bands of %d rows drew another picture than bands "+
				"of %d", rows, yieldMost)
		}
	}
}

// a band the system stops for the display is drawn again smaller, the
// picture is the same as one never stopped, and the kernel goes on.
func TestABandStoppedForTheDisplayIsDrawnAgainSmaller(t *testing.T) {
	kernel := yieldKernel(t)
	want := drawnWith(t, kernel, yieldMost)
	kernel.pretendStops = 2
	got := drawnWith(t, kernel, yieldMost)
	if kernel.pretendStops != 0 {
		t.Fatalf("%d pretended stops were never met", kernel.pretendStops)
	}
	if !slices.Equal(got, want) {
		t.Error("a picture with stopped bands differs from one without")
	}
	if again := drawnWith(t, kernel, 0); !slices.Equal(again, want) {
		t.Error("the kernel after a stop drew another picture")
	}
}

// rows stopped more times than mostStops fail, naming the rows and the
// stops, and the kernel is not refused after: a stop is not a timeout.
func TestTooManyStopsFailNamingTheRows(t *testing.T) {
	kernel := yieldKernel(t)
	kernel.pretendStops = mostStops
	kernel.FixBands(yieldMost)
	_, err := kernel.Grid(yieldWidth, yieldHeight, yieldMost, 4,
		yieldGiven())
	if err == nil {
		t.Fatalf("%d stops in a row drew the picture; want an error",
			mostStops)
	}
	for _, want := range []string{"rows from 0", "for the display",
		fmt.Sprintf("%d times", mostStops), "1 rows tall"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error is %q; want it to say %q", err, want)
		}
	}
	kernel.pretendStops = 0
	if _, err := kernel.Grid(yieldWidth, yieldHeight, yieldMost, 4,
		yieldGiven()); err != nil {
		t.Errorf("the kernel after the stops refused a picture: %v", err)
	}
}

// a one-row band that is stopped is not halved to nothing: the same row
// is sent again, and the picture is the same.
func TestAStoppedRowIsSentAgainWhole(t *testing.T) {
	kernel := yieldKernel(t)
	want := drawnWith(t, kernel, 1)
	kernel.pretendStops = 2
	if got := drawnWith(t, kernel, 1); !slices.Equal(got, want) {
		t.Error("a row stopped twice drew another picture")
	}
}

// a consumer that fails on a band, first, middle or last, leaves no band
// in flight: the stream's next drawing succeeds and is the same picture
// bit for bit, with bands drawn ahead.
func TestAStreamRedrawsAfterAConsumerFails(t *testing.T) {
	kernel := yieldKernel(t)
	want := drawnWith(t, kernel, yieldMost)
	kernel.FixBands(0)
	stream, err := kernel.OpenGrid(yieldWidth, yieldHeight, yieldMost, 4,
		yieldGiven())
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	stop := errors.New("stop at this band")
	// the bands are sized by their time, so how many a picture is drawn
	// in is counted from a drawing; the last has no band ahead of it
	bands := 0
	counted := func(int, int, []float32) error {
		bands++
		return nil
	}
	if err := stream.DrawBands(counted); err != nil {
		t.Fatal(err)
	}
	for _, failing := range []int{0, bands / 2, bands - 1} {
		seen := 0
		consume := func(int, int, []float32) error {
			seen++
			if seen > failing {
				return stop
			}
			return nil
		}
		if err := stream.DrawBands(consume); !errors.Is(err, stop) {
			t.Fatalf("band %d: the drawing gave %v; want the consumer's "+
				"error alone", failing, err)
		}
		got, err := stream.Draw()
		if err != nil {
			t.Fatalf("after a failure at band %d the stream refused: %v",
				failing, err)
		}
		if !slices.Equal(got, want) {
			t.Errorf("after a failure at band %d the stream drew another "+
				"picture", failing)
		}
	}
}

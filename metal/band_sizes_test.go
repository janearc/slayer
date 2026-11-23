//go:build draw && darwin

package metal

import (
	"slices"
	"testing"
)

// bands of a cheap kernel under the default limit begin at one row,
// never hold more than twice the tallest before them, and grow to the
// most the buffers hold; the picture is the one fixed bands draw.
func TestQuickBandsGrowFromOneRow(t *testing.T) {
	kernel := yieldKernel(t)
	want := drawnWith(t, kernel, yieldMost)
	kernel.FixBands(0)
	var heights []int
	got := make([]float32, 0, len(want))
	consume := func(_, down int, samples []float32) error {
		heights = append(heights, down)
		got = append(got, samples...)
		return nil
	}
	if err := kernel.GridBands(yieldWidth, yieldHeight, yieldMost, 4,
		yieldGiven(), consume); err != nil {
		t.Fatal(err)
	}
	if heights[0] != 1 {
		t.Errorf("the first band is %d rows; want one", heights[0])
	}
	for index := 1; index < len(heights); index++ {
		if tallest := slices.Max(heights[:index]); heights[index] > 2*tallest {
			t.Errorf("band %d is %d rows, after none over %d", index,
				heights[index], tallest)
		}
	}
	if tallest := slices.Max(heights); tallest != yieldMost {
		t.Errorf("the tallest band is %d rows; want %d", tallest, yieldMost)
	}
	if !slices.Equal(got, want) {
		t.Error("bands sized by their time drew another picture")
	}
}

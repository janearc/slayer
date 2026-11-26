//go:build draw && darwin

package metal

import "testing"

const regionCoordinates = `
#include <metal_stdlib>
using namespace metal;
kernel void coordinates(device float2 *out [[buffer(0)]],
                        constant uint2 &origin [[buffer(1)]],
                        uint2 at [[thread_position_in_grid]],
                        uint2 grid [[threads_per_grid]]) {
    out[at.y * grid.x + at.x] = float2(at + origin);
}
`

func TestRegionBandsCarryGlobalCoordinatesIntoCompactRows(t *testing.T) {
	kernel, err := Open(regionCoordinates, "coordinates")
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	kernel.FixBands(2)
	const width, height = 3, 5
	seen := make([]bool, height)
	err = kernel.RegionBandsSent(11, 17, width, height, 2, 2, nil, nil,
		func(first, down int, samples []float32) error {
			for y := 0; y < down; y++ {
				if seen[first+y] {
					t.Fatalf("row %d arrived twice", first+y)
				}
				seen[first+y] = true
				for x := 0; x < width; x++ {
					at := (y*width + x) * 2
					if samples[at] != float32(11+x) || samples[at+1] != float32(17+first+y) {
						t.Fatalf("pixel %d,%d: %v", x, first+y, samples[at:at+2])
					}
				}
			}
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}
	for row, got := range seen {
		if !got {
			t.Errorf("row %d was not delivered", row)
		}
	}
}

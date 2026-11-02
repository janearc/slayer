package kernel_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
)

// pinholeRay is the harness's own pinhole: looking down minus z from
// the origin, wide radians from the picture's left edge to its right,
// square pixels, each pixel seen through its middle, up the picture up.
func pinholeRay(wide float64, x, y, across, down int) maths.Vec {
	half := math.Tan(wide / 2)
	right := (2*(float64(x)+0.5)/float64(across) - 1) * half
	up := (1 - 2*(float64(y)+0.5)/float64(down)) * half *
		float64(down) / float64(across)
	return maths.V(right, up, -1).Unit()
}

// film.Pinhole sees each pixel where the harness's geometry says.
func TestThePinholeSeesWhereItsGeometrySays(t *testing.T) {
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	for _, pixel := range [][2]int{{0, 0}, {23, 15}, {12, 8}, {5, 11}} {
		got := lens.Ray(pixel[0], pixel[1], 24, 16, maths.Band{}).Dir
		want := pinholeRay(lens.Wide, pixel[0], pixel[1], 24, 16)
		if got.Sub(want).Len() > 1e-12 {
			t.Errorf("pixel %v looks along %v; its geometry is %v", pixel,
				got, want)
		}
	}
}

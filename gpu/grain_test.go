//go:build darwin

package gpu

import (
	"slices"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
)

// grainSide is the samples along a pixel's side in the grain tests: the
// count that ships.
const grainSide = 4

// a closed shutter still tells the kernel the samples along a pixel's
// side, so a wrinkle fades by the finished pixel and not by a sample.
func TestAClosedShutterCarriesThePixelsSide(t *testing.T) {
	given := viewValues(maths.Subspace{Scale: 1}, film.Pinhole{},
		shutter{side: grainSide}, 32, 24)
	want := []float64{0, 0, grainSide, 0}
	got := append(slices.Clone(given["exposure"]), 0, 0, 0, 0)[:4]
	if !slices.Equal(got, want) {
		t.Errorf("the closed exposure is %v; want %v", got, want)
	}
}

// a shutter that names no side is one sample a side, never nought,
// which the kernel would divide by.
func TestASideOfNoughtIsOne(t *testing.T) {
	if got := closedExposure(true, 0); !slices.Equal(got,
		[]float64{0, 1, 1}) {
		t.Errorf("the closed exposure of no side is %v; want [0 1 1]", got)
	}
}

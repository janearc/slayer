//go:build darwin

package metal

import (
	"math"
	"testing"
)

func TestRegionOriginFitsTheNativeCoordinates(t *testing.T) {
	for _, check := range []struct {
		x, y, width, height int
		want                bool
	}{
		{0, 0, 1, 1, true},
		{19, 23, 7, 5, true},
		{math.MaxInt32 - 1, math.MaxInt32 - 1, 1, 1, true},
		{-1, 0, 1, 1, false},
		{0, -1, 1, 1, false},
		{math.MaxInt32, 0, 2, 1, false},
		{0, math.MaxInt32, 1, 2, false},
		{0, 0, 0, 1, false},
		{0, 0, 1, -1, false},
	} {
		if got := regionOriginFits(check.x, check.y, check.width, check.height); got != check.want {
			t.Errorf("origin (%d,%d), size %dx%d: got %v, want %v", check.x, check.y, check.width, check.height, got, check.want)
		}
	}
}

func TestRegionBandsRefusesInvalidOriginBeforeOpeningAStream(t *testing.T) {
	kernel := &Kernel{}
	called := false
	err := kernel.RegionBandsSent(-1, 0, 1, 1, 1, 1, nil, nil,
		func(int, int, []float32) error { called = true; return nil })
	if err != errGridSize || called {
		t.Fatalf("invalid origin returned %v, callback called %v", err, called)
	}
}

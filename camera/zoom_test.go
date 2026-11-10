package camera

import (
	"math"
	"testing"
	"time"
)

// Expectations are hand-computed angles, including reordered samples.
func TestZoomHoldsEndpointsAndSamplesWithoutAPlayhead(test *testing.T) {
	zoom, err := NewZoom(20, 80, 4*time.Second, "")
	if err != nil {
		test.Fatal(err)
	}
	for _, sample := range []struct {
		at   time.Duration
		want float64
	}{{2 * time.Second, 50}, {-time.Second, 20}, {0, 20},
		{time.Second, 35}, {4 * time.Second, 80}, {9 * time.Second, 80},
		{2 * time.Second, 50}} {
		got, err := zoom.At(sample.at)
		if err != nil || got != sample.want {
			test.Fatalf("at %s: got %g, want %g: %v",
				sample.at, got, sample.want, err)
		}
	}
}

// Cubic smoothstep at one quarter is 5/32; 20+80*5/32 is 32.5.
func TestZoomSmoothAngleAndConstantLens(test *testing.T) {
	zoom, err := NewZoom(20, 100, 4*time.Second, "smooth")
	if err != nil {
		test.Fatal(err)
	}
	if got, err := zoom.At(time.Second); err != nil || got != 32.5 {
		test.Fatalf("smooth quarter: %g, %v", got, err)
	}
	hold, err := NewZoom(45, 45, time.Second, "linear")
	if err != nil {
		test.Fatal(err)
	}
	if got, err := hold.At(time.Second / 2); err != nil || got != 45 {
		test.Fatalf("constant lens: %g, %v", got, err)
	}
}

// Invalid lenses are rejected before interpolation or conversion to radians.
func TestZoomRejectsInvalidAnglesTimingAndEasing(test *testing.T) {
	for _, bad := range []float64{0, -1, 180, 181, math.NaN(),
		math.Inf(1), math.Inf(-1), math.SmallestNonzeroFloat64} {
		if _, err := NewZoom(bad, 60, time.Second, ""); err == nil {
			test.Fatalf("accepted initial angle %g", bad)
		}
		if _, err := NewZoom(60, bad, time.Second, ""); err == nil {
			test.Fatalf("accepted final angle %g", bad)
		}
	}
	for _, duration := range []time.Duration{0, -1, maxDuration + 1} {
		if _, err := NewZoom(20, 60, duration, ""); err == nil {
			test.Fatalf("accepted duration %s", duration)
		}
	}
	if _, err := NewZoom(20, 60, time.Second, "typo"); err == nil {
		test.Fatal("accepted unknown easing")
	}
}

// A zero or absent checked lens must not silently select a default angle.
func TestZoomNeedsACheckedLens(test *testing.T) {
	for _, zoom := range []*Zoom{nil, {}} {
		if _, err := zoom.At(time.Second); err == nil {
			test.Fatal("unconstructed zoom supplied a lens")
		}
	}
}

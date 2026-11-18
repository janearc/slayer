package thing

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestWakeFoamMaximumCoverageAndDisabledDefault(t *testing.T) {
	foam := WakeFoamResponse{OriginEast: 12, Heading: math.Pi / 2,
		HalfWidth: 5, Length: 24, Wavelength: 9.6,
		Coverage: .5, Band: .2, Reflectance: maths.Grey(.9), Satin: .8}
	maximum := 0.0
	for x := -20; x <= 30; x++ {
		for z := -12; z <= 12; z++ {
			at := foam.At(maths.V(float64(x), 0, float64(z)), 0)
			if at < 0 || at > .5 {
				t.Fatalf("coverage outside authored bound at (%d,%d): %g", x, z, at)
			}
			maximum = math.Max(maximum, at)
		}
	}
	if maximum == 0 {
		t.Fatal("authored nonzero foam has no occupied patch")
	}
	if got := (WakeFoamResponse{}).At(maths.Vec{}, 0); got != 0 {
		t.Fatalf("zero-value foam=%g, want disabled", got)
	}
}

func TestWakeFoamSnapshotTimeKeepsMaterialWithFrozenGeometry(t *testing.T) {
	foam := WakeFoamResponse{OriginEast: 12, Heading: math.Pi / 2,
		Speed: 6, HalfWidth: 5, Length: 24, Wavelength: 9.6,
		Coverage: 1, Band: 1, Reflectance: maths.Grey(.9), Satin: .8,
		FrozenTime: true, Seconds: 0}
	if first, later := foam.At(maths.Vec{}, 0), foam.At(maths.Vec{}, 3); first != later || first == 0 {
		t.Fatalf("snapshot changed with render time: %g to %g", first, later)
	}
	foam.FrozenTime = false
	if first, later := foam.At(maths.Vec{}, 0), foam.At(maths.Vec{}, 3); first == later {
		t.Fatalf("scene-clock foam did not move: %g", first)
	}
}

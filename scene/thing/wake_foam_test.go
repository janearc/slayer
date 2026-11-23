package thing

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// wakeFoamNowhere is the point at the origin.
var wakeFoamNowhere = maths.V(nought, nought, nought)

// wakeFoamPick is a number the foam tests chose, by its name in
// numbers.yaml after the stem wake-foam.
func wakeFoamPick(t *testing.T, name string) float64 {
	t.Helper()
	return picked(t, "wake-foam-"+name)
}

// wakeFoamAuthored is the foam both tests start from: the wake's place,
// size and material. A test adds the coverage, the band and the time.
func wakeFoamAuthored(t *testing.T) WakeFoamResponse {
	t.Helper()
	return WakeFoamResponse{
		OriginEast:  wakeFoamPick(t, "origin-east"),
		Heading:     math.Pi / pair,
		HalfWidth:   wakeFoamPick(t, "half-width"),
		Length:      wakeFoamPick(t, "length"),
		Wavelength:  wakeFoamPick(t, "wavelength"),
		Reflectance: maths.Grey(wakeFoamPick(t, "reflectance")),
		Satin:       wakeFoamPick(t, "satin"),
	}
}

// wakeFoamScan is the most coverage the foam gives over the patch the
// tests scan. A coverage below nought or past the bound stops the test.
func wakeFoamScan(
	t *testing.T, foam WakeFoamResponse, bound float64,
) float64 {
	t.Helper()
	west := int(wakeFoamPick(t, "west-edge"))
	east := int(wakeFoamPick(t, "east-edge"))
	south := int(wakeFoamPick(t, "south-edge"))
	north := int(wakeFoamPick(t, "north-edge"))
	maximum := float64(nought)
	for x := west; x <= east; x++ {
		for z := south; z <= north; z++ {
			point := maths.V(float64(x), nought, float64(z))
			at := foam.At(point, nought)
			if at < nought || at > bound {
				t.Fatalf("coverage outside authored bound at (%d,%d): %g",
					x, z, at)
			}
			maximum = math.Max(maximum, at)
		}
	}
	return maximum
}

// foam authored at a coverage never goes past it, has somewhere an
// occupied patch, and the zero value of foam gives none.
func TestWakeFoamMaximumCoverageAndDisabledDefault(t *testing.T) {
	foam := wakeFoamAuthored(t)
	foam.Coverage = wakeFoamPick(t, "coverage")
	foam.Band = wakeFoamPick(t, "band")
	if wakeFoamScan(t, foam, foam.Coverage) == nought {
		t.Fatal("authored nonzero foam has no occupied patch")
	}
	got := (WakeFoamResponse{}).At(wakeFoamNowhere, nought)
	if got != nought {
		t.Fatalf("zero-value foam=%g, want disabled", got)
	}
}

// foam at a frozen time keeps its geometry whatever the render time and
// keeps its material; foam on the scene clock moves.
func TestWakeFoamSnapshotTimeKeepsMaterialWithFrozenGeometry(t *testing.T) {
	foam := wakeFoamAuthored(t)
	foam.Speed = wakeFoamPick(t, "speed")
	foam.Coverage = wakeFoamPick(t, "full")
	foam.Band = wakeFoamPick(t, "full")
	foam.FrozenTime = true
	foam.Seconds = nought
	later := wakeFoamPick(t, "later")
	first := foam.At(wakeFoamNowhere, nought)
	again := foam.At(wakeFoamNowhere, later)
	if first != again || first == nought {
		t.Fatalf("snapshot changed with render time: %g to %g", first, again)
	}
	foam.FrozenTime = false
	first = foam.At(wakeFoamNowhere, nought)
	again = foam.At(wakeFoamNowhere, later)
	if first == again {
		t.Fatalf("scene-clock foam did not move: %g", first)
	}
}

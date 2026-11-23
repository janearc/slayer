package shape

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// tubeDistancesFile is the places the distance is asked at, and what it is
// there.
const tubeDistancesFile = "testdata/tube-distances.csv"

// tubeSample is the tube the tests measure: a ring with a thick wall and a
// short length.
func tubeSample(t *testing.T) Tube {
	t.Helper()
	return Tube{
		Radius: picked(t, "tube-radius"),
		Thick:  picked(t, "tube-thick"),
		Height: picked(t, "tube-height"),
	}
}

// A tube is a shape; inside its wall the distance is less than zero, on the
// wall it is zero, and away from it, in or out, it grows.
func TestTubeDistance(t *testing.T) {
	var _ Shape = Tube{}
	sample := tubeSample(t)
	tolerance := picked(t, "tube-distance-tolerance")
	for _, row := range readSheet(t, tubeDistancesFile) {
		point := maths.V(cell(t, row, "x"), cell(t, row, "y"),
			cell(t, row, "z"))
		want := cell(t, row, "want")
		if got := sample.Distance(point); math.Abs(got-want) > tolerance {
			t.Errorf("%s, at %v: %v, want %v", row["name"], point, got, want)
		}
	}
}

// The marched normal on the inside of the wall points toward the axis,
// and on the outside away from it.
func TestTubeNormalsFaceTheirSides(t *testing.T) {
	sample := tubeSample(t)
	inner := maths.V(nought, nought, picked(t, "tube-inside-z"))
	outer := maths.V(nought, nought, picked(t, "tube-outside-z"))
	in, out := Normal(sample, inner), Normal(sample, outer)
	tolerance := picked(t, "tube-normal-tolerance")
	if !in.Near(maths.V(nought, nought, -unity), tolerance) ||
		!out.Near(maths.V(nought, nought, unity), tolerance) {
		t.Fatalf("inside %v, outside %v", in, out)
	}
}

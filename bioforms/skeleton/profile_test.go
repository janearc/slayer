package skeleton

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

func TestAProfileEasesBetweenItsStations(test *testing.T) {
	profile := NewProfile(maths.V(0, 0, 0), maths.V(1, 0, 0),
		maths.V(0, 1, 0), []float64{.2, .4, .2}, []float64{.1, .1})
	wide, deep := profile.Size(.25)
	if math.Abs(wide-.3) > 1e-12 || math.Abs(deep-.1) > 1e-12 {
		test.Fatalf("a quarter along: %v, %v", wide, deep)
	}
	if wide, _ := profile.Size(1); math.Abs(wide-.2) > 1e-12 {
		test.Fatalf("at the end: %v", wide)
	}
}

func TestAProfileIsDrawnClosed(test *testing.T) {
	profile := NewProfile(maths.V(0, 0, 0), maths.V(1, 0, 0),
		maths.V(0, 1, 0), []float64{.2, .3, .1}, []float64{.1, .15, .05})
	points, faces := profile.Build(nil, nil,
		func(p maths.Vec) maths.Vec { return p })
	made, err := mesh.New(points, faces, nil, false)
	if err != nil {
		test.Fatal(err)
	}
	if !made.Closed() {
		test.Fatal("the profile is not closed")
	}
}

// Anti-tests: what a profile must not do.

func TestAProfileOutsideItsLengthKeepsItsEnds(test *testing.T) {
	profile := NewProfile(maths.V(0, 0, 0), maths.V(1, 0, 0),
		maths.V(0, 1, 0), []float64{.2, .4}, []float64{.1, .3})
	if wide, deep := profile.Size(-1); wide != .2 || deep != .1 {
		test.Fatalf("before the start: %v, %v", wide, deep)
	}
	if wide, deep := profile.Size(2); wide != .4 || deep != .3 {
		test.Fatalf("past the end: %v, %v", wide, deep)
	}
}

func TestAProfileWithNoStationsHasNoSize(test *testing.T) {
	if got := stationAt(nil, .5); got != 0 {
		test.Fatalf("no stations gave %v", got)
	}
}

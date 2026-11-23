package honopu

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/terrain"
)

func TestTriangleSlopeFollowsGroundFaces(t *testing.T) {
	heights := [2][2]float64{{0, 4}, {8, 28}}
	window, err := terrain.BuildWindow(0, 0, 2, 2, 4,
		func(column, row int) (float64, error) { return heights[row][column], nil })
	if err != nil {
		t.Fatal(err)
	}
	ground, _, err := window.Ground()
	if err != nil {
		t.Fatal(err)
	}
	for _, sample := range []struct {
		face        int
		east, south float64
	}{{0, 3, 1}, {1, 1, 3}} {
		a, b, c := ground.Corners(sample.face)
		normal := b.Sub(a).Cross(c.Sub(a))
		want := math.Hypot(normal.X, normal.Z) / math.Abs(normal.Y)
		if got := triangleSlope(window, sample.east, sample.south); math.Abs(got-want) > 1e-12 {
			t.Errorf("face %d: slope %g, want %g", sample.face, got, want)
		}
	}
}

func TestProceduralReliefIsOptInAndFiltersFineDetail(t *testing.T) {
	window, err := terrain.BuildWindow(0, 0, 2, 2, 4,
		func(column, row int) (float64, error) { return float64(column*8 + row*4), nil })
	if err != nil {
		t.Fatal(err)
	}
	base := relief{window: window}
	detailed := relief{window: window, detail: true}
	point := maths.V(2.3, 21, 1.4)
	if base.At(point) == detailed.At(point) {
		t.Fatal("authored detail did not change the material")
	}
	if detailed.At(point) == detailed.Within(point, 20) {
		t.Fatal("fine surface detail did not fade with the pixel footprint")
	}
	if base.At(point) != base.Within(point, 20) {
		t.Fatal("the original material changed with the pixel footprint")
	}
	if _, err := NewWithDetail(true, "unknown"); err == nil {
		t.Fatal("accepted an unknown detail mode")
	}
}

func TestVerticalWeatheringFollowsGravityAndFiltersFineDetail(t *testing.T) {
	var horizontalChange, verticalChange float64
	for i := 0; i < 50; i++ {
		point := maths.V(float64(i)*3.17, float64(i)*4.13, float64(i)*2.11)
		at := verticalWeathering(point, 0)
		eastward := point.Add(maths.V(2, 0, 0))
		upward := point.Add(maths.V(0, 2, 0))
		horizontal := verticalWeathering(eastward, 0)
		vertical := verticalWeathering(upward, 0)
		horizontalChange += math.Abs(horizontal - at)
		verticalChange += math.Abs(vertical - at)
	}
	if horizontalChange <= 3*verticalChange {
		t.Fatalf("weathering is not elongated vertically: horizontal %g, vertical %g",
			horizontalChange, verticalChange)
	}
	point := maths.V(13.4, 91.2, 28.7)
	if verticalWeathering(point, 0) == verticalWeathering(point, 10) {
		t.Fatal("fine weathering did not filter with pixel footprint")
	}
}

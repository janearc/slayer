package session

import (
	"math"
	"testing"

	"github.com/janearc/slayer/terrain"
)

// the near window the tests lay in a coarse one: eleven points a side, two
// metres apart, so twenty metres across. picked: small enough to count.
const (
	nearSide = 11
	nearStep = 2.0
)

// nearWindow is a flat window of nearSide points a side, laid at an east
// and a south.
func nearWindow(t *testing.T, east, south float64) nearGround {
	t.Helper()
	window, err := terrain.BuildWindow(0, 0, nearSide, nearSide, nearStep,
		func(column, row int) (float64, error) { return 7, nil })
	if err != nil {
		t.Fatal(err)
	}
	return nearGround{window: window, east: east, south: south}
}

// A near window holds the places inside it, edges and all, and no place
// outside it.
func TestNearGroundHoldsItsOwnPlaces(t *testing.T) {
	near := nearWindow(t, 100, 200)
	for _, inside := range [][2]float64{{100, 200}, {110, 210}, {120, 220}} {
		if !near.holds(inside[0], inside[1]) {
			t.Errorf("%v is inside and not held", inside)
		}
	}
	for _, outside := range [][2]float64{{99, 210}, {121, 210}, {110, 199},
		{110, 221}} {
		if near.holds(outside[0], outside[1]) {
			t.Errorf("%v is outside and held", outside)
		}
	}
}

// The hole a near window leaves in the coarse ground is its own rectangle.
func TestNearGroundHoleIsItsRectangle(t *testing.T) {
	hole := nearWindow(t, 100, 200).hole()
	if hole.East != 100 || hole.South != 200 || hole.Across != 20 ||
		hole.Down != 20 {
		t.Errorf("hole %+v", hole)
	}
}

// A near window that stands past the coarse window's edge is refused, and
// one wholly inside it is not.
func TestNearGroundMustLieInsideTheCoarseWindow(t *testing.T) {
	coarse, err := terrain.BuildWindow(0, 0, nearSide*10, nearSide*10,
		nearStep*5, func(column, row int) (float64, error) { return 0, nil })
	if err != nil {
		t.Fatal(err)
	}
	if err := nearWindow(t, 100, 200).lies(coarse); err != nil {
		t.Errorf("inside: %v", err)
	}
	if err := nearWindow(t, 1085, 200).lies(coarse); err == nil {
		t.Error("a near window past the east edge was not refused")
	}
	if err := nearWindow(t, -1, 200).lies(coarse); err == nil {
		t.Error("a near window past the west edge was not refused")
	}
}

// A near window whose middle is the coarse window's middle starts half the
// difference of their widths in from its first point, and one a thousand
// metres north of it starts a thousand metres nearer the north edge.
func TestOffsetFromPutsTheNearWindowByItsMiddle(t *testing.T) {
	far := groundPlace{latitude: 37.8, longitude: -122.4, across: 8000,
		down: 6000}
	near := groundPlace{latitude: 37.8, longitude: -122.4, across: 1000,
		down: 800}
	east, south := offsetFrom(far, near, terrain.EarthRadius)
	if east != 3500 || south != 2600 {
		t.Errorf("middles together: %v east %v south", east, south)
	}
	const thousand = 1000.0 // metres the near window is moved north
	near.latitude += thousand / terrain.EarthRadius * 180 / math.Pi
	east, south = offsetFrom(far, near, terrain.EarthRadius)
	if math.Abs(east-3500) > 1e-6 || math.Abs(south-1600) > 1e-6 {
		t.Errorf("a thousand metres north: %v east %v south", east, south)
	}
}

// Without a near window the water's top is at the coarse window's corner,
// where it always was.
func TestWaterTopIsTheCornerWithoutANearWindow(t *testing.T) {
	stage := &staging{}
	if x, z := stage.waterTop(); x != 0 || z != 0 {
		t.Errorf("water top %v %v", x, z)
	}
}

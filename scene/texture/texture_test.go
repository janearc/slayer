package texture

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

var a, b = maths.Grey(0.8), maths.Grey(0.25)

// Neighbouring cells alternate, across the floor in both directions.
func TestCheckerAlternates(t *testing.T) {
	c := Checker(1, a, b)
	if c.At(maths.V(0, 0, 0)) != a || c.At(maths.V(1, 0, 0)) != b ||
		c.At(maths.V(0, 0, 1)) != b || c.At(maths.V(1, 0, 1)) != a {
		t.Error("a checker's neighbours do not alternate")
	}
}

// A floor at height zero is one colour per square whichever side of zero
// the arithmetic lands a point, so it never speckles.
func TestTheFloorIsNotOnABoundary(t *testing.T) {
	c := Checker(1, a, b)
	for _, y := range []float64{-1e-12, 0, 1e-12} {
		if c.At(maths.V(0.3, y, 0.3)) != a {
			t.Errorf("a point at height %v changed colour", y)
		}
	}
}

// It is three dimensional: going up a whole cell changes the colour, so a
// ball in a checker is checkered all the way round.
func TestCheckerIsSolid(t *testing.T) {
	c := Checker(1, a, b)
	if c.At(maths.V(0, 0, 0)) == c.At(maths.V(0, 1, 0)) {
		t.Error("a cell up is the same colour")
	}
}

// Size scales the cells.
func TestCheckerSize(t *testing.T) {
	c := Checker(2, a, b)
	if c.At(maths.V(0, 0, 0)) != c.At(maths.V(0.9, 0, 0)) ||
		c.At(maths.V(0, 0, 0)) == c.At(maths.V(2, 0, 0)) {
		t.Error("cells of size 2 are not 2 across")
	}
}

// A wire is bright on a tiling's edges and black in the middle of a
// tile.
func TestWireIsBrightOnEdgesAndBlackWithin(t *testing.T) {
	tl := hexes.Sphere{N: 4}
	w := Wire{Tiling: tl, Light: maths.Grey(1)}
	tile := tl.Nearest(maths.V(0.3, 0.9, 0.1).Unit())
	if l := w.At(tl.Centre(tile)); l.Luminance() > 0.01 {
		t.Errorf("the middle of a tile is lit: %v", l)
	}
	// a point half way between two neighbouring centres is on the edge
	n := tl.Neighbours(tile)[0]
	mid := tl.Centre(tile).Add(tl.Centre(n)).Unit()
	if l := w.At(mid); l.Luminance() < 0.9 {
		t.Errorf("the edge is dim: %v", l)
	}
}

// A disc with an orbital rate winds with time: the streak at a point after a
// quarter turn's worth of time is where the orbit carries it, and at zero it is
// where it was.
//
// Prograde is from +X toward -Z, counterclockwise seen from up, the way
// thing.Orbit, celestial's hole's Circular and Beam.Boost all go, so the
// streaks run with the gas.
func TestAThinDiscWindsWithTime(t *testing.T) {
	d := Thin{Inner: 1, Outer: 10, Hot: 6000, Bright: 1, Streaks: 1, Seed: 3, Omega: 1}
	r := 4.0
	// at radius four the rate is one over eight: a quarter turn in
	// four pi seconds
	quarter := math.Pi / 2 * math.Pow(r, 1.5)
	later := d.Tick(time.Duration(quarter * float64(time.Second))).(Thin)
	now := d.At(maths.V(r, 0, 0))
	arrived := later.At(maths.V(0, 0, -r))
	if moved := later.At(maths.V(r, 0, 0)); now == moved {
		t.Errorf("the disc did not move: %v", now)
	}
	if math.Abs(arrived.Luminance()-now.Luminance()) > 1e-6*math.Max(1, now.Luminance()) {
		t.Errorf("the streak at +X is %v; a quarter turn on, at -Z, it is %v", now, arrived)
	}
	if still := d.Tick(0).(Thin).At(maths.V(r, 0, 0)); still != now {
		t.Errorf("at nought the disc moved: %v vs %v", still, now)
	}
}

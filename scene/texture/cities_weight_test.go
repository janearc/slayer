package texture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// A place wants a city by the bell of its latitude, 0.15 + 0.85 *
// exp(-(lat - 35)^2 / 392), whatever the side of the equator and however
// long the vector.
//
// the values are bc's: 0.18734639 at the equator, 0.62878745 at 20 and
// at 50 (the bell is the same 15 either side of 35), 1 at 35,
// 0.32257788 at 60, 0.15037848 at the pole.
func TestCitiesWeightIsTheBellOfTheLatitude(t *testing.T) {
	c := Cities{}
	for _, w := range []struct{ lat, want float64 }{
		{0, 0.1873463935798963}, {20, 0.6287874479264054}, {35, 1},
		{50, 0.6287874479264054}, {60, 0.3225778768134672}, {90, 0.15037847684441713},
	} {
		for _, side := range []float64{1, -1} {
			rad := w.lat * maths.Degree
			p := maths.V(math.Cos(rad), side*math.Sin(rad), 0).Scale(3)
			if got := c.weight(p); !within(got, w.want, 1e-9) {
				t.Errorf("the weight at %v degrees (side %v) is %v, want %v", w.lat, side, got, w.want)
			}
		}
	}
}

// Near water a place wants a city six tenths again, and not otherwise.
func TestCitiesWeightIsSixTenthsMoreOnACoast(t *testing.T) {
	p := maths.V(math.Cos(20*maths.Degree), math.Sin(20*maths.Degree), 0)
	for _, c := range []struct {
		coast func(maths.Vec) bool
		want  float64
	}{
		{nil, 0.6287874479264054},
		{func(maths.Vec) bool { return false }, 0.6287874479264054},
		{func(maths.Vec) bool { return true }, 1.0060599166822487},
	} {
		if got := (Cities{Coast: c.coast}).weight(p); !within(got, c.want, 1e-9) {
			t.Errorf("the weight at 20 degrees is %v, want %v", got, c.want)
		}
	}
}

// A direction 1e-160 long has a Unit whose Y is past one (1.0000056,
// checked first), and the weight holds it to the pole, 90 degrees, and
// not past, where the arcsine would be no number.
func TestCitiesWeightHoldsALatitudePastThePole(t *testing.T) {
	if y := maths.V(0, 1e-160, 0).Unit().Y; !(y > 1) {
		t.Fatalf("the direction was meant to measure past 1, and is %v", y)
	}
	for _, y := range []float64{1e-160, -1e-160} {
		if got := (Cities{}).weight(maths.V(0, y, 0)); !within(got, 0.15037847684441713, 1e-9) {
			t.Errorf("the weight at a hair off the pole, %v, is %v", y, got)
		}
	}
}

// The rungs count down from the world's tiling, and a count below zero is zero:
// no rung is no rung.
func TestCitiesRungIsThatManyFinerAndNoFewerThanNone(t *testing.T) {
	c := Cities{Tiling: rungs{specs: []rungSpec{junk, junk, junk, junk, junk, junk}}}
	for _, c2 := range []struct{ n, level int }{
		{-3, 0}, {-1, 0}, {0, 0}, {1, 1}, {2, 2}, {4, 4},
	} {
		got, ok := c.rung(c2.n).(rungs)
		if !ok || got.at != c2.level {
			t.Errorf("rung %d is level %d, want %d", c2.n, got.at, c2.level)
		}
	}
}

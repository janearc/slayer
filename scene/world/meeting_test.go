package world

import "testing"

// what two plates do to the ground between them, worked by hand from
// the model in the package comment. closing, two continents make a range
// at 1.1 of the rate, continent on ocean a coast at 0.8, ocean on
// continent a trench at minus 0.6, and two oceans an arc at 0.5.
//
// pulling apart, two continents sag into a rift at 0.35 of the rate, and
// any other pair is a small ridge at minus 0.12 of it. the rates are on
// both sides of one, and the table is every pair at each.
func TestMeetingIsWhatThePlatesAreMadeOf(t *testing.T) {
	for _, c := range []struct {
		closing float64
		// the effect for continent and continent, continent and ocean,
		// ocean and continent, ocean and ocean
		want [4]float64
	}{
		{2, [4]float64{2.2, 1.6, -1.2, 1.0}},
		{0.5, [4]float64{0.55, 0.4, -0.3, 0.25}},
		{-0.5, [4]float64{-0.175, 0.06, 0.06, 0.06}},
		{-2, [4]float64{-0.7, 0.24, 0.24, 0.24}},
	} {
		for k, pair := range [4][2]bool{{true, true}, {true, false},
			{false, true}, {false, false}} {
			got := meeting(c.closing, pair[0], pair[1])
			if !close12(got, c.want[k]) {
				t.Errorf("plates closing at %v, continent %v and %v: %v, want %v",
					c.closing, pair[0], pair[1], got, c.want[k])
			}
		}
	}
}

// plates that neither close nor part do nothing to the ground, for any pair: a
// rate of zero times any of the factors is zero.
func TestMeetingAtRestIsNothing(t *testing.T) {
	for _, pair := range [4][2]bool{{true, true}, {true, false},
		{false, true}, {false, false}} {
		if got := meeting(0, pair[0], pair[1]); got != 0 {
			t.Errorf("plates at rest, continent %v and %v: %v", pair[0], pair[1], got)
		}
	}
}

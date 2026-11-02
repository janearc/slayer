package texture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

var stormPaint = maths.RGB(0.8, 0.4, 0.2)

// swirled is what swirl says, named.
type swirled struct {
	lat, lon float64
	tint     maths.Light
	tintBy   float64
}

func swirlOf(s Spot, lat, lon float64, tint maths.Light, tintBy float64) swirled {
	a, b, c, d := s.swirl(lat, lon, tint, tintBy)
	return swirled{a, b, c, d}
}

// check says what is wrong with a swirl, if anything.
func (got swirled) check(t *testing.T, why string, want swirled) {
	t.Helper()
	if !within(got.lat, want.lat, 1e-9) || !within(got.lon, want.lon, 1e-9) ||
		!near(got.tint, want.tint) || !within(got.tintBy, want.tintBy, 1e-9) {
		t.Errorf("%s: %+v, want %+v", why, got, want)
	}
}

// A spot at latitude 0.6 (whose cosine is 0.8: asin 0.6 is a 3, 4, 5
// triangle's angle) and longitude 1, of size 0.5 and stretch 2, turning
// the gas by 1.5 radians at its heart.
//
// a point at latitude 0.9 and longitude 1.5 is 0.5 of longitude from the
// spot, which squashed is 0.5 * 0.8 / 2 = 0.2 across, and 0.3 up: 0.36
// from the heart, which is r = 0.7211 of its size. the values are bc's.
func TestSpotSwirlsWhatIsInsideItByTheEase(t *testing.T) {
	spot := Spot{Lat: 0.6, Lon: 1, Size: 0.5, Stretch: 2, Turn: 1.5, Paint: stormPaint}
	const (
		in    = 0.189954665296509772968798 // 1 - Smooth(0, 1, r)
		lat   = 0.944122669253025746389488 // the rotation by 1.5 * in
		lon   = 1.269021241101105348543443
		after = 0.5 + in*0.5 // a share already there of 0.5
	)
	swirlOf(spot, 0.9, 1.5, maths.Light{}, 0).check(t, "on a clean place",
		swirled{lat, lon, stormPaint, in})
	prior := maths.RGB(0.2, 0.4, 0.6)
	want := prior.Scale(0.5).Add(stormPaint.Scale(in)).Scale(1 / after)
	swirlOf(spot, 0.9, 1.5, prior, 0.5).check(t, "on a place already tinted",
		swirled{lat, lon, want, after})
}

// The longitude is brought to the same side as the spot's by a whole
// turn either way: a point 0.1 past the spot and a whole turn on is
// 0.1 away, and comes back at the spot's longitude and 0.1, not a turn
// on. in is bc's for r = hypot(0.1, 0.05) / 0.4.
func TestSpotSwirlsAcrossTheSeam(t *testing.T) {
	spot := Spot{Lat: 0, Lon: 0.5, Size: 0.4, Stretch: 1, Turn: 0, Paint: stormPaint}
	const in = 0.809298202685542767507991
	swirlOf(spot, 0.05, 0.6+2*math.Pi, maths.Light{}, 0).check(t, "a turn on",
		swirled{0.05, 0.6, stormPaint, in})
	swirlOf(spot, 0.05, 0.6, maths.Light{}, 0).check(t, "at once",
		swirled{0.05, 0.6, stormPaint, in})
	swirlOf(spot, 0.05, 0.6-2*math.Pi, maths.Light{}, 0).check(t, "a turn back",
		swirled{0.05, 0.6, stormPaint, in})
}

// Outside the spot, at its size and past it, nothing is touched: not
// the longitude a whole turn on, which a swirl would bring back, nor the
// tint. a point 0.5 up with a size of 0.5 is on the rim, at r of one.
func TestSpotLeavesWhatIsAtItsRimOrBeyondAlone(t *testing.T) {
	spot := Spot{Lat: 0.25, Lon: 0, Size: 0.5, Stretch: 1, Turn: 1, Paint: stormPaint}
	tint := maths.RGB(0.1, 0.2, 0.3)
	for _, lat := range []float64{0.75, 1.0, -0.25, -0.5} {
		a, b, c, d := spot.swirl(lat, 2*math.Pi, tint, 0.4)
		if a != lat || b != 2*math.Pi || c != tint || d != 0.4 {
			t.Errorf("at latitude %v: %v, %v, %v, %v, not left alone", lat, a, b, c, d)
		}
	}
}

// A spot of no size is nowhere, even at its own heart, where a division of zero
// by zero would be no number.
func TestSpotOfNoSizeIsNowhere(t *testing.T) {
	tint := maths.RGB(0.1, 0.2, 0.3)
	for _, size := range []float64{0, -1} {
		spot := Spot{Lat: 0.2, Lon: 0.3, Size: size, Stretch: 1, Turn: 1, Paint: stormPaint}
		a, b, c, d := spot.swirl(0.2, 0.3, tint, 0.4)
		if a != 0.2 || b != 0.3 || c != tint || d != 0.4 {
			t.Errorf("size %v: %v, %v, %v, %v, not left alone", size, a, b, c, d)
		}
	}
}

// A stretch of zero, or below it, is a thousandth: a point 0.0004 of longitude
// away is then 0.4 across, and 0.4 of the size, where the ease is 0.352 and the
// share 0.648. at a stretch of one it would be at the heart.
func TestSpotOfNoStretchIsAThousandth(t *testing.T) {
	for _, stretch := range []float64{0, -2} {
		spot := Spot{Lat: 0, Lon: 0, Size: 1, Stretch: stretch, Turn: 0, Paint: stormPaint}
		swirlOf(spot, 0, 0.0004, maths.Light{}, 0).check(t, "stretch of nought",
			swirled{0, 0.0004, stormPaint, 0.648})
	}
}

// A spot whose latitude is past a pole is held to it, where a longitude
// is no distance at all: the gas stays where it was, and only the share
// is the spot's.
//
// at latitude 2 with size 2 a point at 0.5 is 1.5 away, r of 3/4, where
// the ease is 27/32 and the share 5/32. at latitude -2 with size 3 it is
// 2.5 away, r of 5/6, ease 25/36*4/3 and share 2/27.
func TestSpotPastAPoleIsHeldToIt(t *testing.T) {
	north := Spot{Lat: 2, Lon: 0, Size: 2, Stretch: 1, Turn: 0, Paint: stormPaint}
	swirlOf(north, 0.5, 0.3, maths.Light{}, 0).check(t, "past the north pole",
		swirled{0.5, 0.3, stormPaint, 5.0 / 32})
	south := Spot{Lat: -2, Lon: 0, Size: 3, Stretch: 1, Turn: 0, Paint: stormPaint}
	swirlOf(south, 0.5, 0.3, maths.Light{}, 0).check(t, "past the south pole",
		swirled{0.5, 0.3, stormPaint, 2.0 / 27})
}

// A point so near the rim that the ease rounds to one has a share of zero and
// takes no tint: the tint stays nothing, not nothing over nothing. 1 - 1e-10 of
// the size is that near.
func TestSpotAHairInsideItsRimTakesNoTint(t *testing.T) {
	rim := 1 - 1e-10
	if in := 1 - maths.Smooth(0, 1, rim); in != 0 {
		t.Fatalf("the rim was meant to round the share to nought and it is %v", in)
	}
	spot := Spot{Lat: 0, Lon: 0, Size: 1, Stretch: 1, Turn: 1, Paint: stormPaint}
	got := swirlOf(spot, rim, 0, maths.Light{}, 0)
	if got.tint != (maths.Light{}) || got.tintBy != 0 {
		t.Errorf("a hair inside the rim: %+v, not nothing", got)
	}
}

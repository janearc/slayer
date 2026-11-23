package airlight

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The fields the zenith and the nadir are checked on: Earth's view and
// azimuth bins, one bin of the sun's zenith, an even sky but for the two
// polar bins, which turn with azimuth.
const (
	// poleCounts is the crossings behind each bin, so the noise of each
	// is one over its root, a twentieth.
	poleCounts = 400 // picked: about what a polar bin holds
	// pinwheelShare is how far a polar bin's light swings round from the
	// sun, as a share of the sky's: Earth's top row at 30 degrees swings
	// from 0.058 to 0.120 about a mean of 0.081, by FIX-REPORT.
	pinwheelShare = 0.37 // picked: see above
	// ringLooks is how many looks round a ring the sky is read at.
	ringLooks = 64 // picked: four to each azimuth bin
	// atThePole is a look's angle from the pole, in radians, small enough
	// that its cosine is one to the last bit and it still has an azimuth.
	atThePole = 1e-9 // picked: see above
	// ringDegrees is the ring's angle from the pole, in degrees.
	ringDegrees = 1 // defined: the brief's ring a degree out
	// insideDegrees is a look inside the top bin's centre, 14.36 degrees
	// from up, and well away from the pole, in degrees.
	insideDegrees = 7 // picked: see above
	// awayLooks is how many cosines from the nadir bin's centre to the
	// zenith bin's the reading is checked at, away from the poles.
	awayLooks = 97 // picked: about three to each view bin, and odd
)

// pinwheel is a sky whose light is handSky everywhere but in the polar
// view bins, where it is handSky times one and a share by azimuth bin.
func pinwheel(share func(round int) float64) func(int, int, int) float64 {
	return func(_, view, round int) float64 {
		if view == none || view == earthViews-one {
			return handSky * (one + share(round))
		}
		return handSky
	}
}

// turning is a pinwheel that turns as the cosine of the azimuth, as a
// sky that brightens toward the sun does.
func turning(round int) float64 {
	return pinwheelShare * math.Cos(centre(round, earthAzimuths)*math.Pi)
}

// noisy is a pinwheel of noise alone: each azimuth bin a standard
// deviation of its counts above or below, by turns, the worst a sky that
// does not turn could show.
func noisy(round int) float64 {
	return float64(round%two*two-one) / math.Sqrt(poleCounts)
}

// poleSky is the sky of hand fields on Earth's bins whose light is a
// pinwheel's, with poleCounts behind every bin.
func poleSky(share func(round int) float64) (*Fields, *Sky) {
	fields := handFields(one, earthViews, earthAzimuths, pinwheel(share))
	for bin := range fields.counts[none] {
		fields.counts[none][bin] = poleCounts
	}
	return fields, newSky(fields, wayAt(thirty*degree, none), one)
}

// ring is the luminance of a reading round a ring of looks at an angle
// from up, in radians, ringLooks of them evenly round: the least and the
// most.
func ring(read func(maths.Vec) maths.Light, from float64) (least,
	most float64) {
	least, most = math.Inf(one), math.Inf(-one)
	for look := range ringLooks {
		about := float64(look) / ringLooks * fullTurn
		luminance := read(wayAt(from, about)).Luminance()
		least, most = min(least, luminance), max(most, luminance)
	}
	return least, most
}

// at the zenith and at the nadir every azimuth reads the same light, to
// the bit: the polar bin's mean round the azimuths.
func TestAtThePoleEveryAzimuthIsOne(t *testing.T) {
	_, made := poleSky(turning)
	for name, from := range map[string]float64{"zenith": atThePole,
		"nadir": math.Pi - atThePole} {
		side, _ := made.polar(math.Cos(from))
		want := made.poles[side]
		for look := range ringLooks {
			about := float64(look) / ringLooks * fullTurn
			if got := made.Light(wayAt(from, about)); got != want {
				t.Errorf("the %s at %.3g round: %v, want %v", name, about,
					got, want)
			}
		}
	}
}

// the anti-test: the cubic alone, as the sky read before, holds the
// polar bin's pinwheel all the way to the pole, and the check sees it.
func TestTheCubicAloneIsAPinwheelAtThePole(t *testing.T) {
	_, made := poleSky(turning)
	least, most := ring(made.cubic, atThePole)
	if !(most > least) {
		t.Errorf("the cubic at the zenith is %v every way round", least)
	}
}

// ringBound is how far round a ring the luminance of a sky with no
// turn but its noise may swing: twice the standard error of the polar
// bin's mean, its light over the root of all the crossings behind it.
//
// The pole reads that mean, so it is known no better than that; a ring a
// degree from it in a sky that does not turn should differ from it by no
// more, and any swing past that round the ring is the reading's making.
func ringBound(fields *Fields, made *Sky) float64 {
	row := earthViews - one
	var counts float64
	for round := range earthAzimuths {
		counts += fields.counts[none][fields.index(none, row, round)]
	}
	return two * made.poles[one].Luminance() / math.Sqrt(counts)
}

// a ring a degree from the zenith of a sky whose polar bin is noise alone
// swings round by no more than the noise of the polar mean allows.
//
// It reads the mean and m = s + s^2 - s^3 of each point's departure from
// it, s being sin 1 degree over sin 14.36, so m is 0.075; the cubic round
// the azimuths keeps between its points, so the swing is at most 2 m
// sigma, 0.15 sigma, against a bound of 2 sigma over the root of 16, 0.5.
func TestARingADegreeOutSwingsByTheNoise(t *testing.T) {
	fields, made := poleSky(noisy)
	least, most := ring(made.Light, ringDegrees*degree)
	if bound := ringBound(fields, made); most-least > bound {
		t.Errorf("a ring a degree out swings by %.4g, past the noise's %.4g",
			most-least, bound)
	}
}

// the anti-test: the cubic alone swings round the same ring by the
// polar bin's whole noise, 2 sigma, four times the bound, and the check
// sees it.
func TestTheCubicAloneSwingsPastTheNoise(t *testing.T) {
	fields, made := poleSky(noisy)
	least, most := ring(made.cubic, ringDegrees*degree)
	if bound := ringBound(fields, made); !(most-least > bound) {
		t.Errorf("the cubic a degree out swings by %.4g, within %.4g",
			most-least, bound)
	}
}

// away from the polar bins' centres nothing changes: along cosines in
// even steps between the nadir bin's centre and the zenith bin's, each
// step's middle, so none is a rounding past either centre, the poles'
// weight is nought to the bit, and the sky is the cubic.
//
// The sky is held to the cubic within a few roundings, not to the bit:
// the way is made a unit inside Light and again here, and the compiler
// may fuse the sums of squares differently at the two places.
func TestAwayFromThePolesTheSkyIsTheCubic(t *testing.T) {
	_, made := poleSky(turning)
	edge := centre(earthViews-one, earthViews)*two - one
	for step := range awayLooks {
		cosine := edge * (two*(float64(step)+half)/awayLooks - one)
		if _, weight := made.polar(cosine); weight != none {
			t.Errorf("at cosine %v the poles weigh %v", cosine, weight)
		}
		for look := range ringLooks {
			about := float64(look) / ringLooks * fullTurn
			way := lookAt(cosine, about)
			got, want := made.Light(way), made.cubic(way.Unit())
			if !near(got, want) {
				t.Errorf("at cosine %v, %.3g round: %v, the cubic %v",
					way.Y, about, got, want)
			}
		}
	}
}

// the anti-test: inside a polar bin's centre the sky is not the cubic,
// so the check above is not passed by a reading that blends nowhere.
func TestInsideThePoleTheSkyIsNotTheCubic(t *testing.T) {
	_, made := poleSky(turning)
	way := wayAt(insideDegrees*degree, none)
	if got, cubic := made.Light(way), made.cubic(way); got == cubic {
		t.Errorf("%d degrees from up the sky is the cubic, %v",
			insideDegrees, got)
	}
}

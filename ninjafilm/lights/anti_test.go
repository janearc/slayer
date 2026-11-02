package lights_test

import (
	"testing"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/lights"
	"pgregory.net/rapid"
)

// sunHeat is the temperature of a body as hot as the sun's surface.
var sunHeat = data.Picked("lights-sun-heat")

// spectrumAt is the spectrum of a black body at a heat in kelvin.
func spectrumAt(t stopper, kelvin float64) bands.Light {
	t.Helper()
	return lights.Planck(heated(t, kelvin))
}

// anti-tests: a colder body does not give as much as a hotter one. The
// power in the spectrum rises with the heat and never falls.
func TestAColderBlackBodyGivesLess(t *testing.T) {
	rapid.Check(t, colderGivesLess)
}

func colderGivesLess(t *rapid.T) {
	cold := rapid.Float64Range(heatLeast, heatMost).Draw(t, "cold")
	hot := rapid.Float64Range(cold*pair, heatMost*pair).Draw(t, "hot")
	coldLight := everything(spectrumAt(t, cold))
	hotLight := everything(spectrumAt(t, hot))
	if !(coldLight < hotLight) {
		t.Fatalf("%v K gives %v W and %v K gives %v W", cold, coldLight, hot,
			hotLight)
	}
}

// anti-tests: a bulb is not the sun. Bodies as hot as the sun show more
// of their power to an eye than bulbs do.
func TestABodyAsHotAsTheSunShowsMoreToTheEyeThanABulb(t *testing.T) {
	bulbs, suns := spectrumAt(t, bulb), spectrumAt(t, sunHeat)
	seenBulb := within(bulbs, eyeShortest, eyeLongest) / everything(bulbs)
	seenSun := within(suns, eyeShortest, eyeLongest) / everything(suns)
	if !(seenBulb < seenSun) {
		t.Errorf("a bulb shows %v and a body at %v K shows %v", seenBulb,
			sunHeat, seenSun)
	}
}

// anti-tests: a point gives no more than the watts it is given, and a
// point of fewer watts gives fewer.
func TestAPointGivesNoMoreThanItIsGiven(t *testing.T) {
	rapid.Check(t, pointGivesNoMore)
}

func pointGivesNoMore(t *rapid.T) {
	spectrum := drawnSpectrum(t)
	watts := rapid.Float64Range(wattsError, wattsMost).Draw(t, "watts")
	least := watts / pair
	full, err := lights.Point(spectrum, powered(t, watts))
	if err != nil {
		t.Fatal(err)
	}
	half, err := lights.Point(spectrum, powered(t, least))
	if err != nil {
		t.Fatal(err)
	}
	given, fewer := everything(full.Light()), everything(half.Light())
	if given > watts*(upward+wattsError) || !(fewer < given) {
		t.Fatalf("%v W gave %v W and half of it gave %v W", watts, given,
			fewer)
	}
}

// anti-tests: a point gives no light in a band its spectrum has none in.
// The bands with no power stay dark.
func TestAPointGivesNoLightWhereItsSpectrumHasNone(t *testing.T) {
	rapid.Check(t, pointKeepsDarkBandsDark)
}

func pointKeepsDarkBandsDark(t *rapid.T) {
	spectrum := drawnSpectrum(t)
	lamp, err := lights.Point(spectrum, powered(t, sixty))
	if err != nil {
		t.Fatal(err)
	}
	first, last := spectrum.Span()
	for band := first; band <= last; band++ {
		if spectrum.Power(band) == none && lamp.Light().Power(band) != none {
			t.Fatalf("band %d had no power and the point gave %v", band,
				lamp.Light().Power(band))
		}
	}
}

// anti-tests: no band past the span of a black body holds power. The
// bands just before the first and just after the last are dark.
func TestNoBandPastTheSpanOfABlackBodyHoldsPower(t *testing.T) {
	rapid.Check(t, bandsPastTheSpanAreDark)
}

func bandsPastTheSpanAreDark(t *rapid.T) {
	kelvin := rapid.Float64Range(heatLeast, heatMost).Draw(t, "heat")
	spectrum := lights.Planck(heated(t, kelvin))
	first, last := spectrum.Span()
	for _, past := range []bands.Band{first - bands.Band(upward),
		last + bands.Band(upward)} {
		if spectrum.Power(past) != none {
			t.Fatalf("%v K: band %d past the span holds %v W", kelvin, past,
				spectrum.Power(past))
		}
	}
}

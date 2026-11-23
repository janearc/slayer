package observer_test

import (
	"math"
	"testing"

	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/observer"
)

// tintTolerance is how far a tint's colour may be off the colour asked for,
// in linear light: the lines are solved for exactly, so only rounding.
const tintTolerance = 1e-6

// corners are the corners of the srgb gamut but black: pure red, green and
// blue, the three pairs and white.
var corners = [][3]float64{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}, {1, 1, 0},
	{1, 0, 1}, {0, 1, 1}, {1, 1, 1}}

// seen is the colour of a tint as the observer sees it, in linear light.
func seen(t *testing.T, colour [3]float64) [3]float64 {
	t.Helper()
	light, err := observer.Tinted(colour[0], colour[1], colour[2])
	if err != nil {
		t.Fatalf("%v: %v", colour, err)
	}
	red, green, blue := srgb.Light(observer.Swatch(light))
	return [3]float64{red, green, blue}
}

func TestATintComesBackAsTheColourItWasMadeFor(t *testing.T) {
	colours := append([][3]float64{{1, 0.06, 0.5}, {0.04, 0.85, 1},
		{1, 0.5, 0.06}, {0.1, 0.06, 0.45}, {3, 0.2, 0.1}}, corners...)
	for _, colour := range colours {
		got := seen(t, colour)
		for index := range got {
			if math.Abs(got[index]-colour[index]) > tintTolerance {
				t.Errorf("%v came back as %v", colour, got)
				break
			}
		}
	}
}

func TestATintDoesNotComeBackAsAnotherColour(t *testing.T) {
	red, green := seen(t, [3]float64{1, 0, 0}), seen(t, [3]float64{0, 1, 0})
	if red == green || math.Abs(red[0]-green[0]) < 0.5 {
		t.Errorf("a red tint and a green one both come back as %v and %v",
			red, green)
	}
}

func TestPureBlueAndEveryCornerOfTheGamutIsMadeOfLinesOfNoNegativeAmount(t *testing.T) {
	for _, colour := range corners {
		light, err := observer.Tinted(colour[0], colour[1], colour[2])
		if err != nil {
			t.Errorf("%v: %v", colour, err)
			continue
		}
		first, last := light.Span()
		for band := first; band <= last; band++ {
			if light.Power(band) < 0 {
				t.Errorf("%v: band %d has power %v", colour, band,
					light.Power(band))
			}
		}
	}
}

func TestATintOutsideTheGamutOrOfNoLightIsRefused(t *testing.T) {
	for name, colour := range map[string][3]float64{
		"none":     {0, 0, 0},
		"negative": {1, -0.1, 0.5},
		"nan":      {math.NaN(), 0, 1},
		"endless":  {math.Inf(1), 0, 0},
	} {
		if _, err := observer.Tinted(colour[0], colour[1],
			colour[2]); err == nil {
			t.Errorf("%s: a tint of %v was made", name, colour)
		}
	}
}

func TestATintTwiceAsBrightIsTwiceTheLight(t *testing.T) {
	one, err := observer.Tinted(0.2, 0.4, 0.8)
	if err != nil {
		t.Fatal(err)
	}
	two, err := observer.Tinted(0.4, 0.8, 1.6)
	if err != nil {
		t.Fatal(err)
	}
	first, last := one.Span()
	for band := first; band <= last; band++ {
		if math.Abs(two.Power(band)-2*one.Power(band)) > tintTolerance {
			t.Errorf("band %d: %v and %v", band, one.Power(band),
				two.Power(band))
		}
	}
}

func TestATintOfAnotherColourIsNotProportional(t *testing.T) {
	one, err := observer.Tinted(0.2, 0.4, 0.8)
	if err != nil {
		t.Fatal(err)
	}
	other, err := observer.Tinted(0.4, 0.8, 1.0)
	if err != nil {
		t.Fatal(err)
	}
	first, last := one.Span()
	for band := first; band <= last; band++ {
		if math.Abs(other.Power(band)-2*one.Power(band)) > tintTolerance {
			return
		}
	}
	t.Error("a tint with a bluer share is twice another in every band")
}

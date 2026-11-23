package session

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
)

// pointLuminance is the luminance at a metre the stock tint tests ask for.
const pointLuminance = 40

// pointTolerance is how far a lamp's luminance may be off the one asked
// for: the spectra are exact, so only rounding.
const pointTolerance = 1e-6

// heatLamp is a light whose strength is given in the kit's relative light,
// of a black body at a heat, as a manufactured heat lamp is.
type heatLamp struct{ kelvin, luminance float64 }

func (heatLamp) Bands() bands.Light {
	return bands.Split(swatch.Blackbody(2700))
}
func (heatLamp) Strength() float64                  { return 1 }
func (heatLamp) Angles() (half, soft measure.Angle) { return }
func (heatLamp) Share(time.Duration) float64        { return 1 }

// Relative is the heat and luminance the lamp is given in.
func (lamp heatLamp) Relative() (kelvin, luminance float64, relative bool) {
	return lamp.kelvin, lamp.luminance, true
}

// tintLamp is a light of a colour of its own, as a manufactured sign's
// lamp is; it is also a heat lamp, as the manufactory's light is, which a
// tint must come before. Said is whether it says it is tinted.
type tintLamp struct {
	heatLamp
	colour maths.Light
	said   bool
}

// Tint is the colour and luminance the lamp is given in.
func (lamp tintLamp) Tint() (colour maths.Light, luminance float64,
	tinted bool) {
	return lamp.colour, lamp.luminance, lamp.said
}

// strengthOf is the strength at a metre of the lamp pointOf makes of a
// light, at no time in particular, or the test's end.
func strengthOf(t *testing.T, light emit.Described) maths.Light {
	t.Helper()
	point, err := pointOf(light, maths.Vec{}, maths.Up, 1)
	if err != nil {
		t.Fatal(err)
	}
	return point.When(0).Strength
}

func TestATintedLampAndAHeatLampOfOneLuminanceAreOfOneLuminance(t *testing.T) {
	heat := strengthOf(t, heatLamp{kelvin: 2700, luminance: pointLuminance})
	for _, colour := range []maths.Light{maths.RGB(1, 0.06, 0.5),
		maths.RGB(0.04, 0.85, 1), maths.RGB(0, 0, 1), maths.RGB(0, 1, 0)} {
		tint := strengthOf(t, tintLamp{heatLamp{luminance: pointLuminance},
			colour, true})
		if math.Abs(tint.Luminance()-heat.Luminance()) > pointTolerance ||
			math.Abs(tint.Luminance()-pointLuminance) > pointTolerance {
			t.Errorf("%v: luminance %v, a heat lamp's %v, asked %v", colour,
				tint.Luminance(), heat.Luminance(), pointLuminance)
		}
	}
}

func TestATintedLampOfAnotherLuminanceOrColourIsNotTheSameLamp(t *testing.T) {
	colour := maths.RGB(1, 0.06, 0.5)
	base := strengthOf(t, tintLamp{heatLamp{luminance: pointLuminance}, colour,
		true})
	if twice := strengthOf(t, tintLamp{heatLamp{luminance: 2 * pointLuminance},
		colour, true}); twice == base {
		t.Error("twice the luminance is the same lamp")
	}
	if other := strengthOf(t, tintLamp{heatLamp{luminance: pointLuminance},
		maths.RGB(0.04, 0.85, 1), true}); other == base {
		t.Error("another colour is the same lamp")
	}
}

func TestATintedLampHasTheColourOfItsTintAndNotABlackBodys(t *testing.T) {
	colour := maths.RGB(1, 0.06, 0.5)
	got := strengthOf(t, tintLamp{heatLamp{kelvin: 2700,
		luminance: pointLuminance}, colour, true})
	scale := pointLuminance / colour.Luminance()
	want := colour.Scale(scale)
	if math.Abs(got.R-want.R) > 1e-4 || math.Abs(got.G-want.G) > 1e-4 ||
		math.Abs(got.B-want.B) > 1e-4 {
		t.Errorf("strength %v, want %v", got, want)
	}
}

func TestAHeatLampIsMadeAsItWasBeforeTintsWere(t *testing.T) {
	lamp := heatLamp{kelvin: 2700, luminance: pointLuminance}
	spectrum, err := emit.Relative(lamp.kelvin, lamp.luminance)
	if err != nil {
		t.Fatal(err)
	}
	want := emit.Point{Light: spectrum}.When(0).Strength
	if got := strengthOf(t, lamp); got != want {
		t.Errorf("a heat lamp's strength is %v, want %v", got, want)
	}
}

func TestALampThatSaysItIsNotTintedIsAHeatLampWhateverItsTint(t *testing.T) {
	heat := heatLamp{kelvin: 2700, luminance: pointLuminance}
	untinted := tintLamp{heat, maths.RGB(0, 1, 0), false}
	if got, want := strengthOf(t, untinted), strengthOf(t, heat); got != want {
		t.Errorf("an untinted lamp with a tint is %v, a heat lamp %v", got,
			want)
	}
}

func TestATintTheKitCannotCarryIsRefusedAndNotMadeABlackBody(t *testing.T) {
	_, err := pointOf(tintLamp{heatLamp{luminance: pointLuminance},
		maths.Light{}, true}, maths.Vec{}, maths.Up, 1)
	if err == nil {
		t.Error("a lamp tinted with no colour was made")
	}
}

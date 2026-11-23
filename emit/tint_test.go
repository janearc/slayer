package emit

import (
	"math"
	"testing"

	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/observer"
)

// tintLuminance is the luminance the tint tests ask for at a metre.
const tintLuminance = 320

// luminanceOf is the colour of a relative tint as the observer sees it, in
// the kit's relative light.
func luminanceOf(t *testing.T, colour maths.Light,
	luminance float64) maths.Light {
	t.Helper()
	light, err := RelativeTint(colour, luminance)
	if err != nil {
		t.Fatal(err)
	}
	red, green, blue := srgb.Light(observer.Swatch(light))
	return maths.RGB(red, green, blue)
}

func TestARelativeTintHasTheLuminanceAskedForAndTheColourOfTheTint(t *testing.T) {
	for _, colour := range []maths.Light{maths.RGB(1, 0.06, 0.5),
		maths.RGB(0, 0, 1), maths.RGB(0, 1, 0)} {
		got := luminanceOf(t, colour, tintLuminance)
		if math.Abs(got.Luminance()-tintLuminance) > 1e-6 {
			t.Errorf("%v: luminance %v, want %v", colour, got.Luminance(),
				tintLuminance)
		}
		scale := tintLuminance / colour.Luminance()
		if math.Abs(got.R-colour.R*scale) > 1e-4 ||
			math.Abs(got.G-colour.G*scale) > 1e-4 ||
			math.Abs(got.B-colour.B*scale) > 1e-4 {
			t.Errorf("colour %v, want %v", got, colour.Scale(scale))
		}
	}
}

func TestARelativeTintOfAnotherLuminanceOrColourIsNotTheSameLight(t *testing.T) {
	colour := maths.RGB(1, 0.06, 0.5)
	base := luminanceOf(t, colour, tintLuminance)
	if brighter := luminanceOf(t, colour, 2*tintLuminance); brighter == base {
		t.Error("twice the luminance is the same light")
	}
	cyan := maths.RGB(0.04, 0.85, 1)
	if luminanceOf(t, cyan, tintLuminance) == base {
		t.Error("another colour at one luminance is the same light")
	}
}

func TestARelativeTintOfNoLightOrNoLuminanceIsRefused(t *testing.T) {
	for name, try := range map[string]struct {
		colour    maths.Light
		luminance float64
	}{
		"no colour":     {maths.Light{}, tintLuminance},
		"no luminance":  {maths.RGB(1, 0, 0), 0},
		"negative":      {maths.RGB(1, 0, 0), -1},
		"not a number":  {maths.RGB(1, 0, 0), math.NaN()},
		"negative part": {maths.RGB(1, -1, 0), tintLuminance},
		"endless light": {maths.RGB(1, 0, 0), math.Inf(1)},
	} {
		if _, err := RelativeTint(try.colour, try.luminance); err == nil {
			t.Errorf("%s: a light was made", name)
		}
	}
}

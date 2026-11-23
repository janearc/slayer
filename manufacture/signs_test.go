package manufacture

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/observer"
)

// signNames are the four kinds that are one sign.
var signNames = []string{"sign-roof", "sign-screen", "sign-blade",
	"sign-awning"}

// signedSeeds is how many seeds the tower sign tests look at.
const signedSeeds = 8

func TestEachSignKindIsMadeAtEveryDetailAndLightsItsSurroundings(t *testing.T) {
	for _, name := range signNames {
		row, err := kindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, detail := range Complexities() {
			made, err := MakeWithComplexity(row, sow(2), detail)
			if err != nil {
				t.Errorf("%s at %s: %v", name, detail, err)
				continue
			}
			if len(made.slots()) == 0 || len(made.lights) == 0 {
				materials := len(made.slots())
				t.Errorf("%s at %s has %d materials and %d lights", name,
					detail, materials, len(made.lights))
			}
		}
	}
}

func TestTheLightsOfASignAreTintedHaveNoHeatAndAreOfAnotherSpectrumThanWarm(t *testing.T) {
	warm, err := newRelativeLight(measure.Kelvin(2700), 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range signNames {
		made := mustMake(t, name, 5)
		if len(made.lights) == 0 {
			t.Errorf("%s has no light", name)
		}
		for _, one := range made.lights {
			colour, luminance, tinted := one.made.Tint()
			if !tinted || !(luminance > 0) || colour == (maths.Light{}) {
				t.Errorf("%s: a light of %v, %v, %v", name, colour,
					luminance, tinted)
			}
			if one.made.colour.heat.InKelvin() != 0 {
				t.Errorf("%s: a tinted light has a heat of %v K", name,
					one.made.colour.heat.InKelvin())
			}
			if sameSpectrum(one.made.bands(), warm.bands()) {
				t.Errorf("%s: a tinted light is as a warm one's", name)
			}
		}
	}
}

// sameSpectrum is whether two lights have the same power in every band
// either holds.
func sameSpectrum(one, other bands.Light) bool {
	oneFirst, oneLast := one.Span()
	otherFirst, otherLast := other.Span()
	first, last := oneFirst, oneLast
	if otherFirst < first {
		first = otherFirst
	}
	if otherLast > last {
		last = otherLast
	}
	for band := first; band <= last; band++ {
		if one.Power(band) != other.Power(band) {
			return false
		}
	}
	return true
}

func TestATintedLightIsNotABlackBodyAndAHeatLampStillIsOne(t *testing.T) {
	warm, err := newRelativeLight(measure.Kelvin(2700), 10)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, tinted := warm.Tint(); tinted {
		t.Error("a light of a heat is tinted")
	}
	if _, _, relative := warm.Relative(); !relative {
		t.Error("a light of a heat is not in the kit's relative light")
	}
	planck := bands.Split(swatch.Blackbody(2700))
	if !sameSpectrum(warm.bands(), planck) {
		t.Error("a light of a heat is not the black body's spectrum")
	}
	pink, err := newTintedLight(maths.RGB(1, 0.06, 0.5), 10)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, tinted := pink.Tint(); !tinted {
		t.Error("a tinted light is not tinted")
	}
	if _, _, relative := pink.Relative(); relative {
		t.Error("a tinted light claims a black body's heat")
	}
}

func TestATintedLightOfNoColourOrNoLuminanceIsRefused(t *testing.T) {
	if _, err := newTintedLight(maths.Light{}, 10); err == nil {
		t.Error("a tinted light of no colour was made")
	}
	if _, err := newTintedLight(maths.RGB(1, 0, 0), 0); !errors.Is(err,
		errTint) {
		t.Errorf("a tinted light of no luminance: %v", err)
	}
}

// luminanceAsWatts is the luminance of a light read as the road that reads
// watts a steradian reads it, at an exposure of one: its spectrum scaled
// until, added up over every band, it is the light's strength, and seen by
// the observer. emit.Of does this; the manufactory never imports emit
// (decided 2026-10-06, held by TestTheManufactoryImportsNoEmit), so the
// reading is made here from the description's own spectrum and strength.
func luminanceAsWatts(t *testing.T, made light) float64 {
	t.Helper()
	spectrum := made.Bands()
	first, last := spectrum.Span()
	sum := 0.0
	for band := first; band <= last; band++ {
		sum += spectrum.Power(band)
	}
	if !(sum > 0) {
		t.Fatal("a light with no power in any band")
	}
	scaled := spectrum.Scale(made.Strength() / sum)
	red, green, blue := srgb.Light(observer.Swatch(scaled))
	return maths.RGB(red, green, blue).Luminance()
}

func TestTwoTintsOfOneLuminanceAreOfOneLuminanceOnTheRoadsThatReadWatts(t *testing.T) {
	var got []float64
	for _, colour := range []maths.Light{maths.RGB(1, 0.06, 0.5),
		maths.RGB(0, 1, 0), maths.RGB(0, 0, 1)} {
		made, err := newTintedLight(colour, 10)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, luminanceAsWatts(t, made))
	}
	for _, each := range got[1:] {
		if math.Abs(each-got[0]) > 1e-9*got[0] {
			t.Errorf("tints of one luminance are of luminances %v", got)
		}
	}
}

func TestTwoTintsOfAnotherLuminanceAreOfAnotherLuminanceOnTheRoadsThatReadWatts(t *testing.T) {
	dim, err := newTintedLight(maths.RGB(0, 1, 0), 10)
	if err != nil {
		t.Fatal(err)
	}
	bright, err := newTintedLight(maths.RGB(0, 1, 0), 20)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(luminanceAsWatts(t, bright)-2*luminanceAsWatts(t, dim)) >
		1e-9*luminanceAsWatts(t, bright) {
		t.Error("twice the luminance is not twice the light")
	}
}

func TestASignNeedsASownSeed(t *testing.T) {
	for _, row := range signRows() {
		if _, err := row.made(seed{}); !errors.Is(err, errGood) {
			t.Errorf("an unsown seed made a sign: %v", err)
		}
	}
}

// towerLights is how many lights a tower of a seed has, with its recipe's
// signs pinned on, off, or left alone.
func towerLights(t *testing.T, number uint64, signs *bool) int {
	t.Helper()
	row, err := Tower(TowerSpec{Irregularity: 0.3, Signs: signs})
	if err != nil {
		t.Fatal(err)
	}
	made, err := makeOf(row, sow(number))
	if err != nil {
		t.Fatalf("tower from seed %d: %v", number, err)
	}
	return len(made.lights)
}

func TestATowerCarriesSignsOnlyWhenItsRecipePinsThemOn(t *testing.T) {
	yes, no := true, false
	var added int
	for number := range uint64(signedSeeds) {
		plain := towerLights(t, number, nil)
		if off := towerLights(t, number, &no); off != plain {
			t.Errorf("seed %d: signs pinned off change %d lights to %d",
				number, plain, off)
		}
		added += towerLights(t, number, &yes) - plain
	}
	if added <= 0 {
		t.Errorf("signs pinned on added %d lights in %d towers", added,
			signedSeeds)
	}
}

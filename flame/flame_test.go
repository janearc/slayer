package flame

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/observer"
)

// green is the wavelength the tests give an optical depth at
var green = measure.Metres(550e-9)

// soot is soot as asked, or the test stops.
func soot(t *testing.T, kelvin, depth, exponent float64) Soot {
	t.Helper()
	made, err := NewSoot(measure.Kelvin(kelvin), depth, green, exponent)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// near is whether a value is within a share of what it should be.
func near(got, want, share float64) bool {
	return math.Abs(got-want) <= share*math.Abs(want)
}

// thick grey soot is a black body: its light summed over every
// wavelength, and over the half of the sky a surface sees, is stefan's
// law with CODATA's constant.
func TestThickGreySootIsABlackBody(t *testing.T) {
	const kelvin = 1300
	thick := soot(t, kelvin, 50, 0)
	const steps, from, to = 200000, 1e-8, 1e-1
	sum, ratio := 0.0, math.Log(to/from)/steps
	for i := range steps {
		metres := from * math.Exp((float64(i)+0.5)*ratio)
		sum += thick.Radiance(measure.Metres(metres)).
			InWattsPerNanometre() * metres * 1e9 * ratio
	}
	const sigma = 5.670374419e-8
	if want := sigma * math.Pow(kelvin, 4); !near(math.Pi*sum, want,
		1e-6) {
		t.Errorf("thick soot gives %v W/m2, a black body %v", math.Pi*sum,
			want)
	}
}

// with an exponent of zero soot is grey: the same emissivity at every
// wavelength, one less e to the minus its depth.
func TestSootWithNoTiltIsGrey(t *testing.T) {
	grey := soot(t, 1300, 0.3, 0)
	want := 1 - math.Exp(-0.3)
	for _, metres := range []float64{360e-9, 550e-9, 830e-9, 5e-6} {
		got := grey.Emissivity(measure.Metres(metres))
		if !near(got, want, 1e-12) {
			t.Errorf("at %v m grey soot is %v, not %v", metres, got, want)
		}
	}
}

// thin soot is its depth tilted by the wavelength: between two
// wavelengths its emissivity goes as their ratio to the exponent, and at
// the wavelength its depth is given at, it is the depth.
func TestThinSootIsItsDepthTilted(t *testing.T) {
	thin := soot(t, 1300, 1e-7, Visible)
	blue, red := measure.Metres(400e-9), measure.Metres(700e-9)
	got := thin.Emissivity(blue) / thin.Emissivity(red)
	if want := math.Pow(700.0/400, Visible); !near(got, want, 1e-6) {
		t.Errorf("blue over red is %v, the tilt says %v", got, want)
	}
	if at := thin.Emissivity(green); !near(at, 1e-7, 1e-6) {
		t.Errorf("at 550 nm thin soot is %v, not its depth", at)
	}
}

// soot is bluer than a grey body as hot and as thick at 550 nm: its
// blue over its red is the larger.
func TestSootIsBluerThanGrey(t *testing.T) {
	tilted, grey := soot(t, 1300, 0.1, Visible), soot(t, 1300, 0.1, 0)
	red, _, blue := srgb.Light(observer.Swatch(tilted.Seen()))
	greyRed, _, greyBlue := srgb.Light(observer.Swatch(grey.Seen()))
	if !(blue/red > greyBlue/greyRed) || !(red > 0) || !(greyRed > 0) {
		t.Errorf("soot's blue over red %v, grey's %v", blue/red,
			greyBlue/greyRed)
	}
}

// a candle radiates hamins, bundy and dillon's 77 W times their 0.17
// (nist, 2005, both measured): 13.09 W.
func TestACandleRadiatesThirteenWatts(t *testing.T) {
	got, err := Radiated(measure.Watts(77), 0.17)
	if err != nil || !near(got.InWatts(), 13.09, 1e-12) {
		t.Errorf("a candle radiates %v W, %v", got.InWatts(), err)
	}
}

// a fire half a metre across puffs 1.5 over the square root of a half
// times a second, 2.1213 times a second, by cetegen and ahmed. a period
// is kept to the nanosecond, a few parts in a thousand million of it.
func TestAFlamePuffsForItsSize(t *testing.T) {
	period, err := Period(measure.Metres(0.5))
	if err != nil {
		t.Fatal(err)
	}
	if hertz := float64(time.Second) / float64(period); !near(hertz,
		1.5/math.Sqrt(0.5), 1e-8) {
		t.Errorf("half a metre puffs %v times a second", hertz)
	}
}

// a fire that cannot be is refused: no heat, a depth less than zero or past
// counting, no wavelength, an exponent outside what is measured, a fraction
// outside zero to one, a heat less than zero, and no width.
func TestAFireThatCannotBeIsRefused(t *testing.T) {
	nan := math.NaN()
	soots := [][4]float64{{0, 1, 550e-9, 1}, {1300, -1, 550e-9, 1},
		{1300, nan, 550e-9, 1}, {1300, math.Inf(1), 550e-9, 1},
		{1300, 1, 0, 1}, {1300, 1, 550e-9, -0.1}, {1300, 1, 550e-9, 2},
		{nan, 1, 550e-9, 1}}
	for _, asked := range soots {
		_, err := NewSoot(measure.Kelvin(asked[0]), asked[1],
			measure.Metres(asked[2]), asked[3])
		if err == nil {
			t.Errorf("soot %v was made", asked)
		}
	}
	for _, asked := range [][2]float64{{77, 1.1}, {77, -0.1}, {77, nan},
		{-1, 0.2}, {nan, 0.2}} {
		if _, err := Radiated(measure.Watts(asked[0]),
			asked[1]); err == nil {
			t.Errorf("a fire of %v W and %v radiated", asked[0], asked[1])
		}
	}
	for _, width := range []float64{0, -1, nan, math.Inf(1)} {
		if _, err := Period(measure.Metres(width)); err == nil {
			t.Errorf("a flame %v m across puffed", width)
		}
	}
}

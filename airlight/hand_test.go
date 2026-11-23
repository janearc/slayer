package airlight

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/moonlight"
	"github.com/janearc/slayer/skylight"
)

// The hand-made fields the reading is checked on.
const (
	handNanometres = 550           // picked: one wavelength, in the middle
	handZeniths    = 2             // picked: two bins of the sun's zenith
	handViews      = 4             // picked: two bins below the horizon, two above
	handAzimuths   = 2             // picked: two bins round from the sun
	handSky        = 0.1           // picked: radiance a steradian a unit of sun
	handUpward     = 0.3           // picked: the light coming up from below
	handExposure   = 1000          // picked: any exposure
	handShare      = 1e-12         // picked: a few roundings of the numbers read
	degree         = math.Pi / 180 // defined: a degree, in radians
	sixty          = 60            // picked: the sun's zenith, in degrees
	thirty         = 30            // picked: another
)

// handFields is fields of one wavelength, 550 nm, of a sun of one watt
// a square metre a nanometre and a band of one nanometre, on a grid of
// so many bins, whose radiance in each bin is what light gives it, and
// whose direct sun reaches the ground whole while the sun is up.
func handFields(zeniths, views, azimuths int,
	light func(zenith, view, round int) float64) *Fields {
	fields := &Fields{wavelengths: []int{handNanometres}, band: one,
		sun: []float64{one}, absorbed: []float64{none}, zeniths: zeniths,
		views: views, azimuths: azimuths,
		suns: skylight.Suns{From: none, To: math.Pi}}
	radiance := make([]float64, zeniths*views*azimuths)
	for zenith := range zeniths {
		for view := range views {
			for round := range azimuths {
				radiance[fields.index(zenith, view, round)] = light(zenith,
					view, round)
			}
		}
	}
	shares := make([]float64, transmittanceSteps+one)
	for step := range shares {
		if stepZenith(step) <= math.Pi/two {
			shares[step] = one
		}
	}
	fields.radiance = [][]float64{radiance}
	fields.counts = [][]float64{make([]float64, len(radiance))}
	fields.transmittance = [][]float64{shares}
	return fields
}

// even is a sky of one radiance every way.
func even(radiance float64) func(int, int, int) float64 {
	return func(int, int, int) float64 { return radiance }
}

// wayAt is the way at a zenith, in radians, and round from north toward
// east by so much, in the world's space: x east, y up, north minus z.
func wayAt(zenith, round float64) maths.Vec {
	return maths.V(math.Sin(zenith)*math.Sin(round), math.Cos(zenith),
		-math.Sin(zenith)*math.Cos(round))
}

// pictured is so much light at the hand wavelength, through the
// spectrum's route, at an exposure.
func pictured(power, exposure float64) maths.Light {
	return moonlight.Spectrum{handNanometres: power}.Picture(exposure)
}

// near is whether two lights are the same within a few roundings.
func near(got, want maths.Light) bool {
	scale := math.Max(want.Luminance(), math.SmallestNonzeroFloat64)
	return math.Abs(got.R-want.R) <= handShare*scale &&
		math.Abs(got.G-want.G) <= handShare*scale &&
		math.Abs(got.B-want.B) <= handShare*scale
}

// the sun is the sun's spectrum times the share of it that reaches the
// ground, by the spectrum's route.
func TestTheSunIsTheSpectrumTimesItsShare(t *testing.T) {
	fields := handFields(handZeniths, handViews, handAzimuths, even(handSky))
	for step := range fields.transmittance[none] {
		fields.transmittance[none][step] = half
	}
	got := Sun(fields, wayAt(sixty*degree, none), handExposure)
	if want := pictured(half, handExposure); !near(got, want) {
		t.Errorf("the sun %v, want %v", got, want)
	}
}

// the anti-test: a sun below the horizon, where the hand fields keep
// none of it, lights nothing.
func TestASunBelowTheHorizonLightsNothing(t *testing.T) {
	fields := handFields(handZeniths, handViews, handAzimuths, even(handSky))
	below := wayAt(math.Pi-thirty*degree, none)
	if got := Sun(fields, below, handExposure); got != (maths.Light{}) {
		t.Errorf("a sun below the horizon gives %v", got)
	}
}

// the meter of a hand-made field, worked: under an even sky of radiance
// L and a sun of S at 60 degrees whose light reaches the ground whole,
// level ground takes S (cos 60 + pi L), the sun's and the sky's.
func TestTheMeterOfAHandMadeField(t *testing.T) {
	fields := handFields(handZeniths, handViews, handAzimuths, even(handSky))
	got, err := Meter(fields, wayAt(sixty*degree, none))
	if err != nil {
		t.Fatal(err)
	}
	landing := half + math.Pi*handSky
	want := grey / pictured(landing*grey/math.Pi, one).Luminance()
	if math.Abs(got-want) > handShare*want {
		t.Errorf("metered %g, want %g", got, want)
	}
}

// the anti-test: the meter sees the sky, its exposure falling as the
// sky brightens by the worked share, and refuses ground nothing lights.
func TestTheMeterSeesTheSky(t *testing.T) {
	way := wayAt(sixty*degree, none)
	dimFields := handFields(handZeniths, handViews, handAzimuths,
		even(handSky))
	dim, err := Meter(dimFields, way)
	if err != nil {
		t.Fatal(err)
	}
	brightFields := handFields(handZeniths, handViews, handAzimuths,
		even(two*handSky))
	bright, err := Meter(brightFields, way)
	if err != nil {
		t.Fatal(err)
	}
	want := (half + math.Pi*handSky) / (half + two*math.Pi*handSky)
	if got := bright / dim; math.Abs(got-want) > handShare {
		t.Errorf("a sky twice as bright meters %g as much, want %g", got,
			want)
	}
	dark := handFields(handZeniths, handViews, handAzimuths, even(none))
	_, err = Meter(dark, wayAt(math.Pi-thirty*degree, none))
	if !errors.Is(err, errUnlit) {
		t.Errorf("unlit ground meters: %v", err)
	}
}

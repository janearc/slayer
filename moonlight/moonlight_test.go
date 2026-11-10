package moonlight

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
)

// near is whether a value is within a share of what it should be.
func near(got, want, share float64) bool {
	return math.Abs(got-want) <= share*math.Abs(want)
}

// the iau's nominal sun, worked out here in si from resolution b3's
// luminosity and radius and the codata constants, with no use of the
// package: its total light at an astronomical unit is the nominal
// solar irradiance, 1361 W a square metre (b3), and its light at
// 550 nm is planck's law there, times pi, times the disc's size.
func TestSunlightIsTheIAUsSun(t *testing.T) {
	const (
		luminosity, radius, au = 3.828e26, 6.957e8, 149597870700.0
		sigma, h, c, k         = 5.670374419e-8, 6.62607015e-34,
			299792458.0, 1.380649e-23
	)
	irradiance := luminosity / (4 * math.Pi * au * au)
	if !near(irradiance, 1361, 2e-4) {
		t.Errorf("the nominal sun gives %v W/m2 at an au", irradiance)
	}
	kelvin := math.Pow(luminosity/(4*math.Pi*radius*radius*sigma), 0.25)
	metres := 550e-9
	planck := 2 * h * c * c / math.Pow(metres, 5) /
		math.Expm1(h*c/(metres*k*kelvin)) * 1e-9
	want := planck * math.Pi * (radius / au) * (radius / au)
	if got := Sunlight()[550]; !near(got, want, 1e-9) {
		t.Errorf("sunlight at 550 nm is %v W/m2/nm, si says %v", got, want)
	}
}

// the full moon is the sun's light times the fact sheet's albedo,
// 0.12, times its radius, 1737.4 km, over its distance, 384,400 km,
// squared, at every wavelength.
func TestTheFullMoonIsTheSunsTimesItsAlbedoAndSize(t *testing.T) {
	share := 0.12 * (1737.4 / 384400) * (1737.4 / 384400)
	sun, moon := Sunlight(), Full()
	for _, nm := range []int{400, 550, 700} {
		if !near(moon[nm], sun[nm]*share, 1e-12) {
			t.Errorf("at %d nm the moon is %v of the sun", nm,
				moon[nm]/sun[nm])
		}
	}
	if whole := 1361 * share; !near(whole, 3.34e-3, 2e-3) {
		t.Errorf("the full moon over every wavelength is %v W/m2", whole)
	}
}

// the full moon's face, summed over its disc, sends the earth its
// light: radiance times the disc's solid angle, pi (1737.4 / 384,400)
// squared, is the full moon's light, at every wavelength.
func TestTheMoonsFaceIsItsLight(t *testing.T) {
	size := 1737.4 / 384400
	face, full := Disc(), Full()
	for _, nm := range []int{420, 600, 780} {
		if !near(face[nm]*math.Pi*size*size, full[nm], 1e-12) {
			t.Errorf("at %d nm the face gives %v, the moon %v", nm,
				face[nm]*math.Pi*size*size, full[nm])
		}
	}
}

// the air takes its share: a moon overhead from kilauea's 1222 m keeps
// most of its green, and one four degrees up keeps less, and of its
// blue less again, so it is redder.
func TestALowMoonIsDimmerAndRedder(t *testing.T) {
	height := measure.Metres(1222)
	high, err := Full().Through(height, measure.Radians(math.Pi/2))
	if err != nil {
		t.Fatal(err)
	}
	low, err := Full().Through(height, measure.Radians(4*math.Pi/180))
	if err != nil {
		t.Fatal(err)
	}
	above := Full()
	if !(high[550]/above[550] > 0.85) || !(low[550] < high[550]) {
		t.Errorf("green kept: %v overhead, %v low", high[550]/above[550],
			low[550]/above[550])
	}
	if !(low[700]/low[450] > high[700]/high[450]) {
		t.Error("the low moon is not redder than the high one")
	}
	if !(low.Total() < high.Total()) {
		t.Error("the low moon is not dimmer")
	}
}

// in the picture the moon's light is the sun's colour, a little warm of white:
// its red at least its blue, and neither zero.
func TestMoonlightIsSunlightsColour(t *testing.T) {
	seen := Full().Picture(1e5)
	if !(seen.R >= seen.B) || !(seen.B > 0) {
		t.Errorf("moonlight seen as %v", seen)
	}
}

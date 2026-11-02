package radiance_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/radiance"
)

// The numbers these tests use.
const (
	// glowing is a radiance picked by the test author, 2026-11-01, of no
	// meaning but its own.
	glowing = 3.25
	// halved is a share. Defined: a half.
	halved = 0.5
	// sunKelvin and greenNanometres are a black body as hot as the sun's
	// surface, 5772 K (IAU 2015 B3), at 500 nm, in the middle of what an
	// eye sees.
	sunKelvin       = 5772
	greenNanometres = 500
	// metresANanometre is the metres in a nanometre. Defined by the si.
	metresANanometre = 1e-9
	// relative is how near the typed law comes to celestial's plain one:
	// the wavelength goes to metres and back, which rounds in the last
	// bits. Measured on 2026-11-01 against slayer's own law over 360 to
	// 830 nm and 1000 to 10000 K: 1.4e-14 at worst.
	relative = 1e-12
)

// A radiance reads back as it was made, and scales by a share.
func TestARadianceReadsBackAndScales(t *testing.T) {
	glow := radiance.WattsPerNanometre(glowing)
	if got := glow.InWattsPerNanometre(); got != glowing {
		t.Errorf("a radiance of %v reads back %v", glowing, got)
	}
	if got := glow.Times(halved).InWattsPerNanometre(); got != glowing*halved {
		t.Errorf("half of %v is %v", glowing, got)
	}
}

// Planck is celestial's law, typed: a sun-hot body at 500 nm, the
// wavelength given in metres, is what radiation.Planck gives in
// nanometres. This holds the typing, not the law, which celestial's
// tests hold.
func TestPlanckIsCelestialsLawTyped(t *testing.T) {
	got := radiance.Planck(measure.Kelvin(sunKelvin),
		measure.Metres(greenNanometres*metresANanometre)).InWattsPerNanometre()
	want := radiation.Planck(greenNanometres, sunKelvin)
	if !(math.Abs(got-want) <= relative*want) {
		t.Errorf("planck at %v K and %v nm is %v, celestial's %v",
			float64(sunKelvin), float64(greenNanometres), got, want)
	}
}

// The anti-test: no heat, no wavelength, less of either, or no number,
// gives no light.
func TestPlanckOfNothingIsNone(t *testing.T) {
	green := measure.Metres(greenNanometres * metresANanometre)
	sun := measure.Kelvin(sunKelvin)
	for name, glow := range map[string]radiance.Radiance{
		"no heat":         radiance.Planck(measure.Kelvin(0), green),
		"less than none":  radiance.Planck(measure.Kelvin(-sunKelvin), green),
		"heat no number":  radiance.Planck(measure.Kelvin(math.NaN()), green),
		"no wavelength":   radiance.Planck(sun, measure.Metres(0)),
		"wavelength less": radiance.Planck(sun, measure.Metres(-1)),
	} {
		if got := glow.InWattsPerNanometre(); got != 0 {
			t.Errorf("%s gives %v", name, got)
		}
	}
}

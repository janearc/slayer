// Package moonlight computes full-moon illumination at Earth, including
// attenuation along its path through the atmosphere.
//
// The Sun is a black body at its measured temperature and angular size.
//
// At opposition, lunar geometric albedo gives reflected brightness relative to
// a white disc of the same apparent size. Irradiance above the atmosphere is
// solar irradiance times albedo times the squared lunar radius-to-distance
// ratio: about 3.3 mW/m^2 using 1361 W/m^2, 0.12, and 1737.4/384400.
//
// Wavelength samples follow the CIE 1931 observer's 1 nm table, so the result
// covers the wavelengths available to that observer and film models using it.
package moonlight

import (
	"math"

	"github.com/janearc/celestial/atmosphere"
	"github.com/janearc/celestial/bands"
	"github.com/janearc/celestial/length"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/observer"
	"github.com/janearc/slayer/radiance"
)

// A Spectrum is light arriving, in watts a square metre a nanometre, at
// each whole nanometre the observer has a row for.
type Spectrum map[int]float64

// co2 is the air's carbon dioxide in parts a million: picked round, at
// about mauna loa's measured yearly mean of 2023 and 2024 (noaa gml).
// rayleigh scattering barely moves with it.
const co2 = 420

// nanometre is a nanometre in metres
const nanometre = 1e-9

// Sunlight is the sun's light at the earth's distance, above the air: a
// black body at the sun's temperature, from its luminosity and radius,
// seen as a disc of its radius an astronomical unit off, pi times its
// radiance times its radius over the distance, squared.
func Sunlight() Spectrum {
	sun := measure.Sun()
	heat := measure.SurfaceTemperature(sun.Luminosity(),
		measure.Sphere(sun.Radius()))
	size := sun.Radius().Over(measure.Metres(length.AU))
	disc := math.Pi * size * size
	lo, hi := swatch.Observed()
	light := make(Spectrum, hi-lo+1)
	for nm := lo; nm <= hi; nm++ {
		wavelength := measure.Metres(float64(nm) * nanometre)
		glow := radiance.Planck(heat, wavelength)
		light[nm] = glow.InWattsPerNanometre() * disc
	}
	return light
}

// Full is the full moon's light at the earth, above the air: the sun's,
// times the moon's geometric albedo, times its radius over its
// distance, squared.
func Full() Spectrum {
	moon := Moon()
	size := moon.Radius().Over(moon.Distance())
	return Sunlight().scaled(moon.Albedo() * size * size)
}

// Disc is the full moon's own radiance, its face's light toward the
// earth in watts a square metre a steradian a nanometre.
//
// by geometric albedo's definition it is the albedo times the radiance
// of a white disc sending all the sun's light back alike every way: the
// sun's light over pi.
func Disc() Spectrum {
	return Sunlight().scaled(Moon().Albedo() / math.Pi)
}

// scaled is the light times a share.
func (light Spectrum) scaled(share float64) Spectrum {
	out := make(Spectrum, len(light))
	for nm, power := range light {
		out[nm] = power * share
	}
	return out
}

// Through is light after the air along a straight way down to a height,
// from an altitude above the horizon: each wavelength kept by rayleigh's
// optical depth along the way, so a low moon is dimmer and redder.
func (light Spectrum) Through(height measure.Length,
	altitude measure.Angle) (Spectrum, error) {
	air, err := atmosphere.DryAir(co2)
	if err != nil {
		return nil, err
	}
	column, err := atmosphere.Along(height, altitude)
	if err != nil {
		return nil, err
	}
	out := make(Spectrum, len(light))
	for nm, power := range light {
		kept, err := air.Through(*column,
			measure.Metres(float64(nm)*nanometre))
		if err != nil {
			return nil, err
		}
		out[nm] = power * *kept
	}
	return out, nil
}

// Total is the light's whole power, watts a square metre.
func (light Spectrum) Total() float64 {
	sum := 0.0
	for _, power := range light {
		sum += power
	}
	return sum
}

// Picture is the light as the tracer's three channels at an exposure, what a
// watt a square metre is in the picture: srgb's reading of it, zero where a
// channel would fall below.
func (light Spectrum) Picture(exposure float64) maths.Light {
	lines := make(map[float64]float64, len(light))
	for nm, power := range light {
		lines[float64(nm)] = power * exposure
	}
	red, green, blue := srgb.Light(observer.Swatch(bands.Lines(lines)))
	return maths.RGB(math.Max(0, red), math.Max(0, green),
		math.Max(0, blue))
}

package airlight

import (
	"errors"
	"math"

	"github.com/janearc/slayer/maths"
)

// ErrUnlit is fields that land no light on the ground to meter.
var ErrUnlit = errors.New("airlight: no light lands on the ground to " +
	"meter an exposure by")

// Sun is the direct sun's light at the fields' ground with the sun along
// a way, at an exposure, as a distant lamp takes it: what it lands on a
// surface square to it. It is the sun's spectrum at the top times the
// fields' share of it that reaches the ground.
//
// It is made as hillside makes the moon's lamp from moonlight's
// spectrum, by the spectrum's Picture, so the sun and the sky are one
// light in two parts.
func Sun(fields *Fields, way maths.Vec, exposure float64) maths.Light {
	zenith := zenithOf(way)
	return fields.seen(func(wave int) float64 {
		return fields.transmittanceAt(wave, zenith)
	}).Picture(exposure)
}

// Meter is the exposure a camera's incident meter gives under the fields
// with the sun along a way: the one at which a level grey card, of
// reflectance g = 0.18 and alike every way, is g in the picture's
// luminance.
//
// The card's radiance is g over pi times the light landing on it, E,
// which at a wavelength is S w (cos z T(z) + sum of L c dc da): the
// sun's, and the sky's over each bin above the horizon.
//
// S is the sun at the top, w the band, z the sun's zenith and T the
// share of it that reaches the ground; L is a bin's radiance, c its
// cosine from up, dc and da its widths in cosine and azimuth, da counted
// both sides of the sun.
//
// So exposure = g / Y(g E / pi), which is pi / Y(E): Y the luminance of
// a spectrum's Picture at exposure one. It refuses fields that land no
// light on the card, with ErrUnlit.
func Meter(fields *Fields, way maths.Vec) (float64, error) {
	zenith := zenithOf(way)
	card := fields.seen(func(wave int) float64 {
		return fields.landing(wave, zenith) * grey / math.Pi
	}).Picture(one).Luminance()
	if !(card > none) {
		return none, ErrUnlit
	}
	return grey / card, nil
}

// landing is the light landing on level ground at a wavelength, per unit
// sunlight at the top, with the sun at a zenith, in radians: the direct
// sun's and the sky's.
func (fields *Fields) landing(wave int, sunZenith float64) float64 {
	return fields.direct(wave, sunZenith) + fields.diffuse(wave, sunZenith)
}

// direct is the direct sun's light on level ground at a wavelength, per
// unit sunlight at the top: its share that reaches the ground, times
// the cosine of its zenith, and none from a sun below the horizon.
func (fields *Fields) direct(wave int, sunZenith float64) float64 {
	return max(none, math.Cos(sunZenith)) *
		fields.transmittanceAt(wave, sunZenith)
}

// diffuse is the sky's light on level ground at a wavelength, per unit
// sunlight at the top: each bin's radiance times its cosine from up,
// none for a bin below the horizon, times its solid angle.
func (fields *Fields) diffuse(wave int, sunZenith float64) float64 {
	slice := fields.slice(wave, sunZenith)
	width := two / float64(fields.views)
	turn := fullTurn / float64(fields.azimuths)
	var sum float64
	for view := range fields.views {
		cosine := max(none, centre(view, fields.views)*two-one)
		for round := range fields.azimuths {
			sum += slice[view*fields.azimuths+round] * cosine * width * turn
		}
	}
	return sum
}

// Package emit describes light sources independently of visible surfaces,
// including stellar light, polar beams, and jets. See doc/emitters.md for the
// design and the distinction between emitter kinds.
//
// Emitter state is data that can be uploaded to the GPU. Evaluated at a given
// time, it also supplies a thing.Lamp for the CPU tracer, which provides the
// reference image for GPU checks.
package emit

import (
	"math"
	"time"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/observer"
	"github.com/janearc/slayer/scene/thing"
)

// Point is light from one place, falling off with the square of the distance:
// alike every way, or kept to a cone.
//
// Light is its spectrum as libtheme carries it, the power in each band at one
// unit of distance. Beyond is how far from At the light starts: a star's
// surface, so the star's own sphere does not shade what it lights.
//
// Pulse, when set, is its strength at a time as a share of Light, for a pulse
// or a flare; nil is always one.
//
// Width, when it is greater than zero, keeps the light to a cone about Toward:
// how far off Toward it is at full strength, in radians, and Soft how much
// further it fades to nothing, as a Polar's beams do. A Width of zero is every
// way, and Toward is then not read.
type Point struct {
	At     maths.Vec
	Light  bands.Light
	Beyond float64
	Pulse  func(t time.Duration) float64
	Toward maths.Vec
	Width  float64
	Soft   float64
}

// Lit is a point emitter at one time: what slayer's tracer takes as a
// lamp, and what the gpu is given for a frame.
type Lit struct {
	At     maths.Vec
	Beyond float64
	// Strength is the light at one unit of distance, reduced to the
	// three channels maths.Light carries
	Strength maths.Light
	// Spectrum is the light at one unit of distance in every band,
	// for a picture drawn in one
	Spectrum bands.Light
	// Toward, Width and Soft are the Point's: a cone the light is
	// kept to, a unit long, or no Width for every way
	Toward maths.Vec
	Width  float64
	Soft   float64
}

// Banded is an emitter that lights things in a band outside the
// visible as well as in it: how strong it is there at one unit of
// distance, and how much of the band lands at a point, before anything
// in the way takes its share.
//
// doc/bands.md says why lamps light in a band here.
type Banded interface {
	thing.Lamp
	In(band maths.Band) float64
	Lands(at maths.Vec, band maths.Band) float64
}

// In is the emitter's power in the band at one unit of distance: the
// power of its spectrum in libtheme's band holding that wavelength.
func (lit Lit) In(band maths.Band) float64 {
	return lit.Spectrum.Power(bands.Of(band.Nanometres))
}

// When is the emitter at time t: the pulse worked out once, here on
// the cpu, so nothing that lights a sample has to ask the time again.
func (point Point) When(at time.Duration) Lit {
	spectrum := point.Light
	if point.Pulse != nil {
		spectrum = spectrum.Scale(point.Pulse(at))
	}
	red, green, blue := srgb.Light(observer.Swatch(spectrum))
	return Lit{At: point.At, Beyond: point.Beyond, Spectrum: spectrum,
		Strength: maths.RGB(red, green, blue),
		Toward:   point.Toward.Unit(), Width: point.Width,
		Soft: point.Soft}
}

// Along is how much of the emitter's light goes out a way, a unit
// long: all of it with no Width, and with one, all of it within Width
// of Toward, fading to none by Width and Soft further.
func (lit Lit) Along(out maths.Vec) float64 {
	if lit.Width <= 0 {
		return 1
	}
	off := math.Acos(math.Max(-1, math.Min(1, out.Dot(lit.Toward))))
	return 1 - maths.Smooth(lit.Width, lit.Width+lit.Soft, off)
}

// Light is what reaches a point from the emitter: the way toward it,
// how far the light travels past Beyond, and the light, by the inverse
// square. Inside Beyond nothing arrives.
func (lit Lit) Light(at, _ maths.Vec) (maths.Vec, float64, maths.Light) {
	toward := lit.At.Sub(at)
	distance := toward.Len()
	if distance <= lit.Beyond {
		return maths.Vec{}, 0, maths.Light{}
	}
	in := toward.Scale(1 / distance)
	return in, distance - lit.Beyond, lit.Strength.Scale(lit.falling(in,
		distance))
}

// Lands is how much of the band reaches a point from the emitter: its
// power in the band, by the inverse square and the cone as its light
// falls. Inside Beyond none of it arrives.
func (lit Lit) Lands(at maths.Vec, band maths.Band) float64 {
	toward := lit.At.Sub(at)
	distance := toward.Len()
	if distance <= lit.Beyond {
		return 0
	}
	in := toward.Scale(1 / distance)
	return lit.In(band) * lit.falling(in, distance)
}

// falling is the share of the emitter's light at one unit of distance
// that arrives a distance away along in, the way toward the emitter:
// the cone's share by the inverse square.
func (lit Lit) falling(in maths.Vec, distance float64) float64 {
	return lit.Along(in.Neg()) / (distance * distance)
}

// a Lit is a lamp slayer's tracer can use as it is, and lights in a band
var _ Banded = Lit{}

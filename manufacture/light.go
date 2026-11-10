package manufacture

import (
	"errors"
	"math"
	"time"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// a light here is a description: what its light is made of, how much of it
// there is, and how it is spread.
//
// a campfire, a spotlight and a maser are all this, with different numbers. the
// manufactory makes the description and stops; it makes no emitter and no lamp,
// and never calls emit. emit reads what is made here.

var errLight = errors.New("manufacture: a light needs a heat or a " +
	"wavelength above nought, a power above nought, each with an end, " +
	"a spread it can have, and for an opening, a line")

// spectrum is what a light's light is made of: the glow of a heat, or
// one line.
//
// made only by newThermal and newLine.
type spectrum struct {
	heat measure.Temperature
	wave measure.Length
}

// newThermal is the glow of a black body at a heat, as a fire's or a
// lamp's filament is, near enough.
func newThermal(heat measure.Temperature) (spectrum, error) {
	if !positive(heat.InKelvin()) {
		return spectrum{}, errLight
	}
	return spectrum{heat: heat}, nil
}

// newLine is light of one wavelength.
func newLine(wave measure.Length) (spectrum, error) {
	if !positive(inScene(wave)) {
		return spectrum{}, errLight
	}
	return spectrum{wave: wave}, nil
}

// made is whether the spectrum came from a door.
func (of spectrum) made() bool { return of.glows() || of.lined() }

// glows is whether it is a heat's glow.
func (of spectrum) glows() bool { return of.heat.InKelvin() > 0 }

// lined is whether it is one line.
func (of spectrum) lined() bool { return inScene(of.wave) > 0 }

// nanometres is how many of libtheme's unit of wavelength a metre is.
const nanometres = 1e9

// bands is the spectrum as Celestial carries light: a black body's by
// planck's law, or all of it in the line's own band.
func (of spectrum) bands() bands.Light {
	if of.lined() {
		return bands.Lines(map[float64]float64{
			inScene(of.wave) * nanometres: 1})
	}
	return bands.Split(swatch.Blackbody(of.heat.InKelvin()))
}

// spread is how a light is spread: the half angle of the cone it
// fills at full strength, and how much further it fades to nothing,
// for light of a spectrum.
type spread interface {
	angles(of spectrum) (half, soft measure.Angle, err error)
}

// everyway is a light alike every way: a bare lamp, a fire.
type everyway struct{}

// angles is a half angle of half a turn and no soft edge: the whole way round.
func (everyway) angles(spectrum) (measure.Angle, measure.Angle, error) {
	return measure.Radians(math.Pi), measure.Radians(0), nil
}

// cone is a beam of a half angle with a soft edge: a spotlight.
//
// made only by newCone.
type cone struct{ half, soft measure.Angle }

// newCone is a cone of a half angle greater than zero, fading over a soft edge
// of zero or more, the two together no more than half a turn.
func newCone(half, soft measure.Angle) (spread, error) {
	wide, edge := half.InRadians(), soft.InRadians()
	if !positive(wide) || !(edge >= 0) || wide+edge > math.Pi {
		return nil, errLight
	}
	return cone{half: half, soft: soft}, nil
}

// angles is the cone's own half angle and soft edge.
func (over cone) angles(spectrum) (measure.Angle, measure.Angle, error) {
	return over.half, over.soft, nil
}

// aperture is a beam as narrow as an opening allows: a dish.
//
// made only by newAperture.
type aperture struct{ across measure.Length }

// newAperture is an opening so far across.
func newAperture(across measure.Length) (spread, error) {
	if !positive(inScene(across)) {
		return nil, errLight
	}
	return aperture{across: across}, nil
}

// firstDark is how many wavelengths over its opening a beam's half
// angle is: where the light through a round opening first goes dark.
const firstDark = 1.22

// angles is the beam of a line through the opening: firstDark
// wavelengths over the opening, with a hard edge. a glow of every
// wavelength has no one such angle, and is refused.
func (over aperture) angles(of spectrum) (measure.Angle, measure.Angle,
	error) {
	if !of.lined() {
		return measure.Angle{}, measure.Angle{}, errLight
	}
	half := firstDark * of.wave.Over(over.across)
	if half > math.Pi {
		half = math.Pi
	}
	return measure.Radians(half), measure.Radians(0), nil
}

// light is a light: a spectrum, a power, a spread already worked out
// for that spectrum, and how it flickers.
//
// made only by newLight.
type light struct {
	colour     spectrum
	power      measure.Power
	half, soft measure.Angle
	flicker    seed
	depth      float64
}

// newLight is a light of a spectrum and a power, spread a way, or why
// there can be none.
func newLight(colour spectrum, power measure.Power,
	over spread) (light, error) {
	if !colour.made() || !positive(power.InWatts()) || over == nil {
		return light{}, errLight
	}
	half, soft, err := over.angles(colour)
	if err != nil {
		return light{}, err
	}
	return light{colour: colour, power: power, half: half,
		soft: soft}, nil
}

// flickering is the light flickering by a seed: down to as little as
// one less depth of itself, never above itself.
func (made light) flickering(from seed, depth float64) (light, error) {
	if !from.made() || !share(depth) {
		return light{}, errLight
	}
	made.flicker, made.depth = from, depth
	return made, nil
}

// bands is the light's spectrum, as Celestial carries light.
func (made light) bands() bands.Light { return made.colour.bands() }

// angles is the half angle of the light's beam at full strength and how much
// further it fades: half a turn and zero for every way.
func (made light) angles() (measure.Angle, measure.Angle) {
	return made.half, made.soft
}

// along is how much of the light's full strength goes a way so far
// off its beam, in radians: all of it inside the half angle, fading
// to none across the soft edge. it is emit.Polar's own fall.
func (made light) along(off float64) float64 {
	inner := made.half.InRadians()
	return 1 - maths.Smooth(inner, inner+made.soft.InRadians(), off)
}

// rings is how many rings a soft edge is cut into to add up what
// passes through it. tuning: the fall across the edge is smooth, so
// this many is good to far better than a part in a million.
const rings = 1024

// covered is the solid angle the beam covers, weighing each way by how much of
// the full strength goes that way: the cap inside the half angle, which is the
// whole ball for every way, and the soft edge counted for what passes through
// it, ring by ring.
//
// the cap is a full turn times one less the angle's cosine. it is worked out as
// twice the square of the half angle's sine, which is the same number, since
// one less a cosine near one is lost to rounding and a very narrow beam would
// come out covering nothing.
func (made light) covered() float64 {
	inner, edge := made.half.InRadians(), made.soft.InRadians()
	halfSine := math.Sin(inner / 2)
	total, width := fullTurn*2*halfSine*halfSine, edge/rings
	for ring := range rings {
		off := inner + (float64(ring)+midway)*width
		total += made.along(off) * fullTurn * math.Sin(off) * width
	}
	return total
}

// strength is the light's power along its beam, in watts per steradian: its
// power over what its beam covers. narrow the beam and the same watts are
// brighter in it.
func (made light) strength() float64 {
	return made.power.InWatts() / made.covered()
}

// flickers is how many times a second a flicker comes and goes, about. made up,
// for a fire. aside is where across the noise a flicker is read: off its
// lattice, where it is zero.
const (
	flickers = 7
	aside    = 0.5
)

// share is how much of itself the light is at a time: one, or for a
// flickering light, noise by its seed between one less its depth and
// one. it is the shape of emit's pulse.
func (made light) share(at time.Duration) float64 {
	if !made.flicker.made() {
		return 1
	}
	wave := noise.Gradient(made.flicker.number, at.Seconds()*flickers,
		aside, aside)
	return 1 - made.depth*asShare(wave)
}

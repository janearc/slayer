// Package film defines emitters and camera projections: the sources of
// radiation and the rays used to record it. The tracer carries light between
// them; render/colour develops the resulting frame.
//
// Most emitters describe visible light; radar also supplies a spectral band.
// The package depends only on maths and libtheme. A test enforces that boundary
// so the optics can remain independent of scene construction.
package film

import (
	"math"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
)

// Emitter is anything that gives off radiation: a lamp's visible light, or a
// band outside the visible, which a Radar gives off and no eye sees.
//
// Light does two jobs for a surface at p facing n. The first is where the
// emitter is from p: a unit vector toward it, and how far, +Inf for one at
// infinity. The second is how much visible light arrives on a surface facing it
// square.
//
// The renderer weighs the light by how squarely the surface faces it and checks
// nothing stands in the way. For a Beamer it asks the first job alone, where
// the beam comes from, and drops the light, which is zero.
//
// An emitter that lights from every direction at once has no one direction to
// give. It returns a zero toward, and light already weighed for the surface,
// and casts no shadow the renderer can test.
type Emitter interface {
	Light(p, n maths.Vec) (toward maths.Vec, dist float64,
		light maths.Light)
}

// Beamer is an emitter that beams a band outside the visible, which a radar
// does: how strong the beam is in the band, and zero in any other.
type Beamer interface {
	Emitter
	Beams(band maths.Band) float64
}

// Radar is a beam of one range from a place: a radar, which lights
// nothing in visible light and, in its own range, lights what it reaches
// so the picture in that range is what scatters back. Range is one of
// libtheme's radiation profiles, radiation.Radio for a radar.
type Radar struct {
	At       maths.Vec
	Range    radiation.Range
	Strength float64
}

// Light is where the radar is from p, at its distance, and no visible
// light at all: the first of Light's two jobs, which the radar's picture
// needs for the shadow it casts in its own band.
func (r Radar) Light(p, _ maths.Vec) (maths.Vec, float64, maths.Light) {
	to := r.At.Sub(p)
	return to.Unit(), to.Len(), maths.Light{}
}

// Beams is the radar's strength in a band of its range, and zero in any other.
func (r Radar) Beams(band maths.Band) float64 {
	if !r.Range.Holds(band.Nanometres) {
		return 0
	}
	return r.Strength
}

// Sun is light from a direction, infinitely far away: every ray of it
// parallel, so its shadows keep their shape at any distance.
type Sun struct {
	// Toward is the direction from anything lit toward the sun. It need
	// not be one unit long.
	Toward maths.Vec
	// Strength is the light arriving on a surface facing it square. A
	// strength of 1 on white paint that gives back all of it is a pixel
	// of 1.
	Strength maths.Light
	// Size is how wide it looks, in radians. The sun is about half a
	// degree. Zero gives sharp shadows; nothing uses it yet.
	Size float64
}

// Light is from the sun's direction, infinitely far, at the sun's full
// Strength on a surface facing it square. The direction is given, so the
// renderer tests for a shadow along it.
func (s Sun) Light(maths.Vec, maths.Vec) (maths.Vec, float64, maths.Light) {
	return s.Toward.Unit(), math.Inf(1), s.Strength
}

// Flood is light from a whole sky's worth of directions at once, the wide
// soft light a set is filled with so its shadows are not black.
//
// It is modelled as an even dome of light around Up: a surface facing up
// gets all of Strength, one on its side half, one facing down none. It
// casts no shadows. Real fill light is blocked by what stands near a
// surface, and showing that takes many rays a point, which is for later.
type Flood struct {
	// Up is the middle of the dome; zero means the world's own up.
	Up maths.Vec
	// Strength is the light on a surface facing straight up.
	Strength maths.Light
}

// Light is from no one direction: a zero toward, an infinite distance,
// and the flood's Strength already weighed for a surface facing n, all
// of it facing up and none facing down. A zero direction means the
// renderer casts no shadow for it.
func (f Flood) Light(_, n maths.Vec) (maths.Vec, float64, maths.Light) {
	up := f.Up
	if up == (maths.Vec{}) {
		up = maths.V(0, 1, 0)
	}
	facing := (1 + n.Unit().Dot(up.Unit())) / 2
	return maths.Vec{}, math.Inf(1), f.Strength.Scale(facing)
}

// Spot is a lamp at a place, aimed one way, giving off light in a cone: a
// pool on the floor with a soft edge, and hard shadows from a point.
//
// Its light falls off with the square of distance, as light from a point
// does: twice as far, a quarter as bright.
type Spot struct {
	// At is where the lamp is.
	At maths.Vec
	// Aim is the way it points.
	Aim maths.Vec
	// Strength is the light one unit away, straight down the middle.
	Strength maths.Light
	// Cone is the half-angle, in radians, out to where the light has
	// gone.
	Cone float64
	// Soft is how much of the cone, in radians, the light takes to fade
	// out at the edge. Zero is a hard edge.
	Soft float64
}

// Light is from the spot's place, at its distance, and the spot's Strength
// dimmed by the square of that distance and by how far off its aim p lies, zero
// past the cone. At the lamp itself there is no direction and no light.
//
// The direction is given, so the renderer tests for a shadow along it.
func (s Spot) Light(p, _ maths.Vec) (maths.Vec, float64, maths.Light) {
	to := s.At.Sub(p)
	dist := to.Len()
	if dist == 0 {
		return maths.Vec{}, 0, maths.Light{}
	}
	toward := to.Scale(1 / dist)
	off := math.Acos(math.Max(-1, math.Min(1,
		toward.Neg().Dot(s.Aim.Unit()))))
	return toward, dist, s.Strength.Scale(s.edge(off) / (dist * dist))
}

// edge is how much of the light reaches an angle off the aim: all of it
// inside the soft edge, none past the cone, and a smooth fall between.
func (s Spot) edge(off float64) float64 {
	inner := s.Cone - s.Soft
	switch {
	case off >= s.Cone:
		return 0
	case off <= inner || s.Soft <= 0:
		return 1
	}
	t := (s.Cone - off) / s.Soft
	return t * t * (3 - 2*t)
}

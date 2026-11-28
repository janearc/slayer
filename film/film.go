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
	// degree. Only a sun drawn as a disc uses it.
	Size float64
	// Disc draws the sun as what it is, a disc Size across: seen in the
	// sky and in mirrors, its light over its disc's solid angle, and
	// softening the shadows it casts by its width. Without it the sun is
	// a point, its shadows hard and no disc in the sky.
	Disc bool
}

// spread is how many ways across a sun's disc a shadow is sought along.
const spread = 8

// golden is the golden angle in radians, which the ways across the
// disc turn by, one to the next, so that eight of them cover it evenly.
const golden = 2.399963229728653

// Spread is the ways from a point toward the sun a shadow is sought
// along: across its disc, when it is drawn as one, eight on a spiral
// turning by the golden angle, each at the root of its place along the
// eight times the disc's radius out from its middle; and when it is not,
// the one way to its middle. The GPU's kernel draws the same spiral.
func (s Sun) Spread() []maths.Vec {
	return s.SpreadCount(spread)
}

// SpreadCount samples the same fixed golden-angle disc with the supplied
// count. Its deterministic directions do not vary between frames.
func (s Sun) SpreadCount(count int) []maths.Vec {
	middle := s.Toward.Unit()
	if !s.Disc || !(s.Size > 0) {
		return []maths.Vec{middle}
	}
	up := maths.V(0, 1, 0)
	if math.Abs(middle.Y) >= 0.9 {
		up = maths.V(1, 0, 0)
	}
	across := middle.Cross(up).Unit()
	other := middle.Cross(across)
	if count <= 0 {
		count = spread
	}
	ways := make([]maths.Vec, count)
	for i := range ways {
		out := s.Size / 2 * math.Sqrt((float64(i)+0.5)/float64(count))
		turn := golden * float64(i)
		aside := across.Scale(math.Cos(turn)).Add(other.Scale(math.Sin(turn)))
		ways[i] = middle.Scale(math.Cos(out)).Add(aside.Scale(math.Sin(out))).Unit()
	}
	return ways
}

// Shown is the sun's light seen along way, when it is drawn as a disc
// and way falls within it: its Strength over the disc's solid angle,
// 2 pi (1 - cos(Size/2)), so that its disc, summed, lights a surface
// facing it as Strength says. None outside the disc, and none when it
// is not drawn as one.
func (s Sun) Shown(way maths.Vec) maths.Light {
	if !s.Disc || !(s.Size > 0) {
		return maths.Light{}
	}
	edge := math.Cos(s.Size / 2)
	if way.Unit().Dot(s.Toward.Unit()) < edge {
		return maths.Light{}
	}
	return s.Strength.Scale(1 / (2 * math.Pi * (1 - edge)))
}

// Disced is the suns among lamps drawn as discs, in their order: all
// Discs need look at. A picture finds them once; looking through every
// lamp at every ray that reaches the sky cost moonlake's fire frames,
// with 219 lamps and no disc, five seconds a frame at 1024 across.
func Disced(lamps []Emitter) []Sun {
	var suns []Sun
	for _, lamp := range lamps {
		if sun, ok := lamp.(Sun); ok && sun.Disc && sun.Size > 0 {
			suns = append(suns, sun)
		}
	}
	return suns
}

// Discs is what suns drawn as discs, as Disced finds them, show along
// way: what an eye or a mirror sees of them in the sky.
func Discs(suns []Sun, way maths.Vec) maths.Light {
	var sum maths.Light
	for _, sun := range suns {
		sum = sum.Add(sun.Shown(way))
	}
	return sum
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

// Stands is where the spot is.
func (s Spot) Stands() maths.Vec { return s.At }

// AtMost is the most light the spot sends any point a distance from it,
// in its cone or out: its brightest part by the inverse square. At the
// spot itself it has no end.
func (s Spot) AtMost(distance float64) float64 {
	most := math.Max(s.Strength.R, math.Max(s.Strength.G, s.Strength.B))
	return most / (distance * distance)
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

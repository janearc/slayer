package film

import (
	"math"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
)

// three is the cubic's constant in the smooth step, 3t^2 - 2t^3. It is
// defined by that cubic.
const three = 3

// Emitter is anything that gives off radiation. It may be the visible
// light of a lamp. It may be a band outside the visible. Radars give
// that off and no eye sees it.
//
// Light does two jobs for a surface at a point facing a normal. The
// first is where the emitter is from the point. That is a unit vector
// toward it with the distance. One at infinity is +Inf away. The second
// is how much visible light arrives on a surface facing it square.
//
// The renderer weighs the light by how squarely the surface faces it. It
// checks that nothing stands in the way. For a Beamer it asks the first
// job alone. That is where the beam comes from. It drops the light,
// which is zero.
//
// Emitters that light from every direction at once have no one direction
// to give. They return a zero toward and light already weighed for the
// surface. They cast no shadow the renderer can test.
//
//ninja:vow waste slayer's tracer takes any lamp here; the mock stands in
type Emitter interface {
	Light(point, normal maths.Vec) (toward maths.Vec,
		dist float64, light maths.Light)
}

// Beamer is an emitter that beams a band outside the visible. Radars
// do. Beams says how strong the beam is in a band. It is zero in
// any other.
//
//ninja:vow waste slayer's tracer takes any beam here; the mock stands in
type Beamer interface {
	Emitter
	Beams(band maths.Band) float64
}

// Radar is a beam of one range from a place. It lights none in
// visible light. In its own range it lights what it reaches, so the
// picture in that range is what scatters back. The range is one of
// libtheme's radiation profiles, radiation.Radio for a radar.
//
//ninja:vow practice slayer's radar picture makes its radar here
type Radar struct {
	at       maths.Vec
	band     radiation.Range
	strength float64
}

// NewRadar is a radar at a place, beaming a range with a strength.
//
//ninja:vow practice slayer's radar picture makes its radar here
func NewRadar(at maths.Vec, band radiation.Range, strength float64) Radar {
	return Radar{at, band, strength}
}

// Light is where the radar is from the point, at its distance. It gives
// no visible light at all. That is the first of the two jobs of Light.
// The picture of the radar needs it for the shadow it casts in its own
// band.
func (radar Radar) Light(point, _ maths.Vec) (toward maths.Vec, dist float64,
	light maths.Light) {
	to := radar.at.Sub(point)
	return to.Unit(), to.Len(), maths.Light{}
}

// Beams is the radar's strength in a band of its range, and zero in any other.
// The range holds its low end and not its high one.
func (radar Radar) Beams(band maths.Band) float64 {
	if !radar.band.Holds(band.Nanometres) {
		return none
	}
	return radar.strength
}

// Sun is light from a direction, infinitely far away. Every ray of it is
// parallel. So its shadows keep their shape at any distance.
//
// Its toward is the direction from anything lit toward the sun. It need
// not be one unit long. Its strength is the light arriving on a surface
// facing it square. White paint that gives back all of a strength of 1
// is a pixel of 1.
//
//ninja:vow practice slayer's scenes make their sun here
type Sun struct {
	toward   maths.Vec
	strength maths.Light
}

// NewSun is a sun in a direction with a strength.
//
//ninja:vow practice slayer's scenes make their sun here
func NewSun(toward maths.Vec, strength maths.Light) Sun {
	return Sun{toward, strength}
}

// Light is from the direction of the sun, infinitely far. It has the full
// strength of the sun on a surface facing it square. The direction is
// given. So the renderer tests for a shadow along it.
func (sun Sun) Light(maths.Vec, maths.Vec) (toward maths.Vec,
	dist float64, light maths.Light) {
	return sun.toward.Unit(), math.Inf(whole), sun.strength
}

// Flood is light from a whole sky's worth of directions at once. It is
// the wide soft light a set is filled with so its shadows are not black.
//
// It is modelled as an even dome of light around its up. Surfaces facing
// up get all of the strength. Surfaces on their side get half. Surfaces
// facing down get none. It casts no shadows.
//
// What stands near a surface blocks real fill light. Showing that takes
// many rays a point. That is for later.
//
// Its up is the middle of the dome. Zero means the world's own up. Its
// strength is the light on a surface facing straight up.
//
//ninja:vow practice slayer's scenes make their flood here
type Flood struct {
	up       maths.Vec
	strength maths.Light
}

// NewFlood is a flood around an up with a strength.
//
//ninja:vow practice slayer's scenes make their flood here
func NewFlood(up maths.Vec, strength maths.Light) Flood {
	return Flood{up, strength}
}

// Light is from no one direction. It has a zero toward and an infinite
// distance. Its strength is already weighed for a surface facing the
// normal. Nothing faces down and everything faces up. Zero directions
// mean the renderer casts no shadow for it.
func (flood Flood) Light(_, normal maths.Vec) (toward maths.Vec,
	dist float64, light maths.Light) {
	up := flood.up
	if up == (maths.Vec{}) {
		up = maths.Up
	}
	facing := (whole + normal.Unit().Dot(up.Unit())) / pair
	return maths.Vec{}, math.Inf(whole), flood.strength.Scale(facing)
}

// Spot is a lamp at a place aimed one way. It gives off light in a cone.
// That is a pool on the floor with a soft edge. Its shadows are hard
// ones from a point.
//
// Its light falls off with the square of distance, as light from a point
// does. Twice as far is a quarter as bright.
//
// Its at is where the lamp is. Its aim is the way it points. Its
// strength is the light one unit away straight down the middle. Its cone
// is the half-angle in radians out to where the light has gone.
//
// Its soft is how much of the cone the light takes to fade out at the
// edge, in radians. Zero is a hard edge.
//
//ninja:vow practice slayer's scenes make their spot here
type Spot struct {
	at, aim    maths.Vec
	strength   maths.Light
	cone, soft float64
}

// NewSpot is a spot at a place and aimed. It has a strength, a cone and
// a soft edge.
//
//ninja:vow practice slayer's scenes make their spot here
func NewSpot(at, aim maths.Vec, strength maths.Light, cone,
	soft float64) Spot {
	return Spot{at, aim, strength, cone, soft}
}

// Light is from the spot's place, at its distance. The spot's strength is
// dimmed by the square of that distance and by how far off its aim p lies, zero
// past the cone. At the lamp itself there is no direction and no light.
//
// The direction is given, so the renderer tests for a shadow along it.
func (spot Spot) Light(point, _ maths.Vec) (toward maths.Vec, dist float64,
	light maths.Light) {
	to := spot.at.Sub(point)
	dist = to.Len()
	if dist == none {
		return maths.Vec{}, none, maths.Light{}
	}
	toward = to.Scale(whole / dist)
	cosine := toward.Neg().Dot(spot.aim.Unit())
	turned := math.Min(whole, cosine)
	off := math.Acos(math.Max(-whole, turned))
	return toward, dist, spot.strength.Scale(spot.edge(off) / (dist * dist))
}

// edge is how much of the light reaches an angle off the aim. All of it
// reaches inside the soft edge. None reaches past the cone. Between
// them lies a smooth fall.
func (spot Spot) edge(off float64) float64 {
	inner := spot.cone - spot.soft
	switch {
	case off >= spot.cone:
		return none
	case off <= inner || spot.soft <= none:
		return whole
	}
	along := (spot.cone - off) / spot.soft
	return along * along * (three - pair*along)
}

// The emitters are emitters, and a radar is a beamer.
var (
	_ Emitter = Sun{}
	_ Emitter = Flood{}
	_ Emitter = Spot{}
	_ Beamer  = Radar{}
)

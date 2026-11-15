package named

import (
	"io"
	"math"
	"time"

	"github.com/janearc/slayer/lantern"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

// Zappy alone: the beamed star in its swarm, package lantern's, with no
// orbitals round it. Its parameters default to zappy as defined, which
// is the far view's middle. A beam shows only where something scatters
// it, so zappy sits in the far view's nebula. One unit is a thousand
// kilometres.

// zappy's nebula: the far view's, with the channel the orbitals swept
// through it though they are not drawn, thicker and glowing less, so the
// beams show against a dark gas rather than a bright one; picked by the
// author of this scene, 2026-11-12, by looking
const (
	zappyNebulaThick = 3e-4
	zappyNebulaGlow  = 0.02
)

// zappy's eye and picture: distance out, elevation in degrees, the lens's
// width in degrees, and a tall picture, nine by sixteen, at two rays a
// side; picked by the author of this scene, 2026-11-12, by looking, so
// the swarm fills the width and the beams run up and down the height
const (
	zappyDistance           = 700
	zappyElevation          = 25
	zappyWideDegrees        = 40
	zappyWidth, zappyHeight = 1080, 1920
	zappyRays               = 2
	zappyHighest            = 90
	zappyFullWear           = 1
	zappyFewest             = 1
	zappyStartsAt           = orbitalStartsAt
	zappyExposure           = orbitalExposure
	zappyKnee, zappyTop     = orbitalKnee, orbitalTop
)

// zappyScene is zappy alone, in its haze, under a star sky.
func zappyScene() Scene {
	return Scene{name: "zappy",
		about: "zappy, the lantern grown up: a star in nested shells " +
			"of black-glass panels that tumble, its two beams spinning " +
			"and precessing, in the orbital's nebula without the orbitals",
		unit:       "a thousand kilometres",
		renderers:  []Renderer{CPU, GPU},
		parameters: zappyParameters(),
		defaults: Defaults{width: zappyWidth, height: zappyHeight,
			rays: zappyRays, wide: zappyWideDegrees * maths.Degree,
			at: zappyStartsAt, tone: colour.Tone{
				Exposure: zappyExposure, Knee: zappyKnee,
				Top: zappyTop}},
		build: buildZappy}
}

// zappyParameters are what zappy takes: its star, its swarm, its shells,
// its beams, its haze and its eye, each defaulting to zappy as defined.
func zappyParameters() []Parameter {
	defined := zappyDefined
	beams := defined.Beams
	return append([]Parameter{
		number("radius", defined.Radius, above(nothing),
			"the star's radius, in thousands of km"),
		number("kelvin", defined.Kelvin, above(nothing),
			"the star's surface, in kelvin"),
		whole("orbits", defined.Orbits, atLeast(zappyFewest),
			"how many rings of panels"),
		whole("panels", defined.Panels, atLeast(zappyFewest),
			"how many panels to a ring"),
		number("band-width", defined.Wide, atLeast(nothing),
			"how wide each ring is, in radii of the star"),
		number("wear", defined.Wear, between(nothing, zappyFullWear),
			"how battered the panels are, nought to one"),
		number("glass", defined.Glass, atLeast(nothing), "the index "+
			"of the panels' black glass; nought is warm metal"),
		number("spread", defined.Spread, atLeast(nothing),
			"radii of the star from one ring to the next out"),
		number("narrowest", defined.Narrowest, atLeast(nothing),
			"the outermost ring's width, narrowing from band-width"),
		span("fastest", defined.Fastest,
			"the outermost ring's turn, quickening from a minute"),
		span("tumble", defined.Tumble,
			"how long the innermost ring takes to tumble end over end"),
		number("glare", defined.Glare, atLeast(nothing), "how much "+
			"brighter along a beam than the star's own light"),
		toggle("still", false, "hold the swarm still, neither "+
			"turning nor tumbling; the beams still spin and precess"),
		number("beam-width", beams.Width, atLeast(nothing), "how far "+
			"off its pole a beam is at full strength, in degrees"),
		number("beam-soft", beams.Soft, atLeast(nothing), "how much "+
			"further a beam fades to nothing, in degrees"),
		number("beam-offset", beams.Offset, unbounded, "how far the "+
			"second pole is turned off the first's opposite, degrees"),
		number("beam-tilt", beams.Tilt, unbounded,
			"how far the poles sit off the spin axis, in degrees"),
		span("beam-spin", beams.Spin,
			"how long the poles take once round the spin axis"),
		number("beam-cone", beams.Cone, unbounded, "how far the spin "+
			"axis leans off the mean axis, in degrees"),
		span("beam-precess", beams.Precess, "how long the spin axis "+
			"takes once round the mean axis"),
	}, zappyView()...)
}

// zappyView are zappy's haze and eye parameters.
func zappyView() []Parameter {
	return []Parameter{
		number("haze", zappyNebulaThick, atLeast(nothing), "how thick "+
			"the nebula the beams show in is; nought is none, and the "+
			"beams light only what they strike"),
		number("glow", zappyNebulaGlow, atLeast(nothing), "how bright the "+
			"nebula's own glow is, as a luminance; nought leaves only "+
			"the light it scatters"),
		number("distance", zappyDistance, above(nothing),
			"how far out the eye is, in thousands of km"),
		number("elevation", zappyElevation,
			between(-zappyHighest, zappyHighest), "how far above the "+
				"swarm's middle plane the eye is, in degrees"),
	}
}

// buildZappy is zappy alone: the star sky, the haze, zappy at the middle
// with its beams' clock, and the eye at the distance and elevation asked,
// looking at the star, its up along the beams' mean axis so they run up
// and down a tall picture.
func buildZappy(held values, _ io.Writer) (*Built, error) {
	world := thing.NewWorld()
	world.Sky = sky.Stars{Dark: maths.RGB(orbitalSkyRed, orbitalSkyGreen,
		orbitalSkyBlue), Seed: orbitalSeed}
	gas := nebula{thick: held.number("haze"), glow: held.number("glow")}
	if err := addNebula(world, nested(), gas); err != nil {
		return nil, err
	}
	clock := new(time.Duration)
	beamed := zappyAsked(held)
	played, err := beamed.Play(world, clock)
	if err != nil {
		return nil, err
	}
	elevation := held.number("elevation") * maths.Degree
	away := held.number("distance")
	from := maths.V(0, away*math.Sin(elevation), away*math.Cos(elevation))
	return &Built{world: world, eye: maths.Subspace{Origin: from,
		Orientation: maths.LookAtUp(from, maths.V(0, 0, 0),
			beamed.Axis.Unit())}, clock: clock, played: played}, nil
}

// zappyAsked is zappy as defined with the values asked in place of its
// own. still holds the swarm, neither turning nor tumbling, and leaves
// the beams turning; the far view's still stops only the turn.
func zappyAsked(held values) lantern.Beamed {
	beamed := zappyDefined
	beamed.Radius = held.number("radius")
	beamed.Kelvin = held.number("kelvin")
	beamed.Orbits = held.whole("orbits")
	beamed.Panels = held.whole("panels")
	beamed.Wide = held.number("band-width")
	beamed.Wear = held.number("wear")
	beamed.Glass = held.number("glass")
	beamed.Spread = held.number("spread")
	beamed.Narrowest = held.number("narrowest")
	beamed.Fastest = held.span("fastest")
	beamed.Tumble = held.span("tumble")
	beamed.Glare = held.number("glare")
	beamed.Beams = lantern.Beams{Width: held.number("beam-width"),
		Soft: held.number("beam-soft"), Offset: held.number("beam-offset"),
		Tilt: held.number("beam-tilt"), Cone: held.number("beam-cone"),
		Spin: held.span("beam-spin"), Precess: held.span("beam-precess")}
	if held.toggle("still") {
		beamed.Turn, beamed.Tumble = nothing, nothing
	}
	return beamed
}

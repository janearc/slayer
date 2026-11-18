package named

import (
	"fmt"
	"io"
	"math"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

// The orbital: a pair of culture orbitals, two bands one inside the other,
// turning opposite ways, in a nebula. From inside, above one floor, a far
// sun lights them; from far outside, zappy at their middle does: the
// beamed star in its swarm, package lantern's.
//
// One unit is a thousand kilometres. Every number in the orbital's files
// was picked for the picture when cmd/orbital was written, by decision where
// a comment says so, and moved here unchanged.

// the orbital, in thousands of kilometres: the radius for one g at a 24
// hour turn, T = 2 pi root(R/g); a band 192 wide (six times the first,
// her calls of 2026-10-04, so the pair reads through the gas) with walls
// one high and a hull a tenth thick; the steps round a band's mesh, and
// the hexes round its floor, each about a unit and a half across
const (
	orbitalRadius = 1840.0
	bandWidth     = 192.0
	bandWall      = 1.0
	bandThick     = 0.1
	bandAround    = 6144
	hexAround     = 7700
)

// the orbital's sky: its darkness and the seed of its stars, which is
// also the seed of the gap walls' marble
const (
	orbitalSkyRed, orbitalSkyGreen, orbitalSkyBlue = 0.003, 0.003, 0.006
	orbitalSeed                                    = 1806
)

// the gap walls' black marble's colour
const gapRed, gapGreen, gapBlue = 0.03, 0.03, 0.035

// the near view's far sun and the dim flood that fills its shadows: the
// way to the sun, its light, and the flood's
const (
	nearSunX, nearSunY, nearSunZ          = 0.45, 0.3, -0.85
	nearSunRed, nearSunGreen, nearSunBlue = 1.2, 1.15, 1.05
	nearFloodRed, nearFloodGreen          = 0.03, 0.035
	nearFloodBlue                         = 0.05
)

// the near view's eye: 500 km above one floor near the gap, 3,000 km to
// one side, looking along the floors a fifth of a radian round and a
// little up, toward a point a million km ahead
const (
	nearAside   = -3.0
	nearAbove   = 0.5
	nearTurn    = 0.2
	nearRise    = 0.12
	nearTowards = 1000
)

// the far view's eye: 25 degrees above the pair's plane
const farElevation = 25

// the orbital as cmd/orbital drew it: the picture's size and rays a side,
// the lens's width in degrees, the first frame's time, and the tone's
// exposure, knee and top
const (
	orbitalWidth, orbitalHeight, orbitalRays = 1600, 900, 2
	orbitalWideDegrees                       = 90
	orbitalStartsAt                          = 500 * time.Millisecond
	orbitalExposure                          = 1
	orbitalKnee, orbitalTop                  = 0.7, 8
)

// the orbital's parameters' defaults: the far eye's distance, the
// nebula's thickness at one, and its glow, and a band's day
const (
	orbitalDistance = 3600
	nebulaThick     = 4.5e-5
	nebulaGlow      = 0.2
	orbitalDay      = 24 * time.Hour
)

// orbitalScene is the pair of orbitals.
func orbitalScene() Scene {
	return Scene{name: "orbital", fastLighting: true,
		about: "two culture orbitals, one inside the other, turning " +
			"opposite ways in a nebula; seen from inside under a far " +
			"sun, or from far outside round zappy, the beamed star in its " +
			"swarm",
		unit:       "a thousand kilometres",
		renderers:  []Renderer{CPU, GPU},
		parameters: orbitalParameters(),
		defaults: Defaults{width: orbitalWidth, height: orbitalHeight,
			rays: orbitalRays, wide: orbitalWideDegrees * maths.Degree,
			at: orbitalStartsAt, tone: colour.Tone{
				Exposure: orbitalExposure, Knee: orbitalKnee,
				Top: orbitalTop}},
		build: buildOrbital}
}

// orbitalParameters are what the orbital takes, as cmd/orbital's flags
// were.
func orbitalParameters() []Parameter {
	return []Parameter{
		toggle("zappy-flares", false, "occasional false-colour illumination from Zappy's X-ray bursts"),
		number("zappy-flare-offset", 0, between(0, 86400), "offset into the burst sequence in scene seconds"),
		oneOf("surface-style", "mixed", []string{"mixed", "terraced-arcology", "twisted-spires", "inhabited-bridge"}, "inhabited macrostructure family; mixed alternates construction rules"),
		whole("surface-structures", 0, between(0, 2048), "seeded inhabited macrostructures per ring; zero preserves the original floor"),
		whole("surface-landforms", 0, between(0, 2048), "seeded mountain and caldera features per ring"),
		toggle("surface-close", false, "look across the first inhabited surface site, travelling with its ring"),
		toggle("surface-tour", false, "fly into and past the inhabited surface site"),
		span("surface-tour-duration", 12*time.Second, "duration of the inhabited surface approach"),
		span("surface-tour-offset", 0, "resume the inhabited surface approach from this local time"),
		toggle("surface-night", false, "place the inhabited surface on the far sun's dark side"),
		whole("surface-district", 0, between(0, 64), "additional distinct inhabited structures around the selected city"),
		toggle("surface-relief", false, "add varied terrain around the inhabited surface site"),
		toggle("surface-city", false, "reserve a dry site for the dense inhabited city"),
		toggle("surface-urban-density", false, "add dense urban buildings, terraces, service surfaces and vegetation"),
		toggle("surface-park-flythrough", false, "enable the experimental human-scale park flythrough"),
		whole("floor-tiles", hexAround, between(16, hexAround), "floor texture tiles around each orbital; reduce for distant proofs"),
		whole("floor-hex-metres", 0, between(0, 1000000), "physical floor cell width across flats in metres; zero retains legacy floor-tiles"),
		toggle("far", false, "from far outside, 25 degrees above their "+
			"plane, the hoops standing upright, lit by zappy"),
		number("distance", orbitalDistance, unbounded,
			"with far, how far out the eye is, in thousands of km"),
		toggle("still", false, "hold zappy's rings still"),
		number("gas", single, unbounded,
			"how many times thicker the nebula"),
		number("neon", nothing, unbounded, "the share of the nebula "+
			"that is neon, nought to one"),
		number("argon", nothing, unbounded, "the share of the nebula "+
			"that is argon, ionized; what neon and argon leave is "+
			"hydrogen"),
		number("glow", nebulaGlow, unbounded,
			"how bright the nebula's glow is, as a luminance"),
		span("day", orbitalDay, "how long one band takes to turn; "+
			"the pair turn opposite ways; nought holds them still"),
	}
}

// buildOrbital is the pair in their nebula, and the eye: inside, under a
// far sun, or far outside, round zappy.
func buildOrbital(held values, said io.Writer) (*Built, error) {
	world := thing.NewWorld()
	world.Sky = sky.Stars{Dark: maths.RGB(orbitalSkyRed, orbitalSkyGreen,
		orbitalSkyBlue), Seed: orbitalSeed}
	// the two walls of each band: a black marble, dark, its cities
	// gathered in clusters and lit
	gapWall := &thing.Material{Name: "gap wall",
		Colour: maths.RGB(gapRed, gapGreen, gapBlue),
		Shine:  marble{seed: orbitalSeed}}
	pair := nested()
	if held.toggle("surface-park-flythrough") && (!held.toggle("surface-city") || !held.toggle("surface-tour")) {
		return nil, fmt.Errorf("orbital surface-park-flythrough needs surface-city and surface-tour")
	}
	if held.toggle("surface-urban-density") && !held.toggle("surface-city") {
		return nil, fmt.Errorf("orbital surface-urban-density needs surface-city")
	}
	if held.toggle("surface-tour") && (held.span("surface-tour-duration") <= 0 || held.span("surface-tour-offset") < 0) {
		return nil, fmt.Errorf("orbital surface tour needs a positive duration and nonnegative offset")
	}
	var focus *orbitalSurfaceSite
	for k, one := range pair {
		district := 0
		if k == 0 {
			district = held.whole("surface-district")
		}
		relief := k == 0 && held.toggle("surface-relief")
		city := k == 0 && held.toggle("surface-city")
		park := k == 0 && held.toggle("surface-park-flythrough")
		dense := k == 0 && (park || held.toggle("surface-urban-density"))
		site, err := one.placeSurface(world, k, gapWall, held.span("day"),
			held.whole("floor-tiles"), held.whole("floor-hex-metres"),
			held.whole("surface-structures"), held.whole("surface-landforms"), district, relief, city, park, dense, held.choice("surface-style"), said)
		if err != nil {
			return nil, err
		}
		if focus == nil {
			focus = site
		}
	}
	gas := nebula{thick: nebulaThick * held.number("gas"),
		neon: held.number("neon"), argon: held.number("argon"),
		glow: held.number("glow")}
	if err := addNebula(world, pair, gas); err != nil {
		return nil, err
	}
	if held.toggle("far") {
		view, err := farView(world, held.number("distance"), held.toggle("still"))
		if err != nil {
			return nil, err
		}
		attachOrbitalCity(view, focus)
		if held.toggle("zappy-flares") {
			addOrbitalFlares(view, time.Duration(held.number("zappy-flare-offset")*float64(time.Second)))
		}
		return view, nil
	}
	view := nearView(world)
	attachOrbitalCity(view, focus)
	if held.toggle("surface-night") {
		reverseOrbitalSun(world)
	}
	if held.toggle("surface-close") || held.toggle("surface-tour") {
		if focus == nil {
			return nil, fmt.Errorf("orbital surface-close needs surface-structures or surface-landforms")
		}
		view.advance = focus.camera
		if held.toggle("surface-tour") {
			view.advance = focus.tour(held.span("surface-tour-duration"), held.span("surface-tour-offset"))
		}
		view.eye, _ = view.advance(0)
	}
	if held.toggle("zappy-flares") {
		addOrbitalFlares(view, time.Duration(held.number("zappy-flare-offset")*float64(time.Second)))
	}
	return view, nil
}

// attachOrbitalCity preserves any prior scene tick and updates every factory
// fixture before the flare composes its own changing lamp into the scene.
func attachOrbitalCity(view *Built, site *orbitalSurfaceSite) {
	if site == nil || site.tick == nil {
		return
	}
	previous := view.tick
	view.tick = func(at time.Duration) {
		if previous != nil {
			previous(at)
		}
		site.tick(at)
	}
	view.tick(0)
}

// nearView lights the pair by a far sun, so the floor near the eye is in
// day and, further round, the arch turns to night and its cities light;
// and puts the eye inside, 500 km above one floor near the gap, its up
// toward the axis as a person's on the floor would be, looking along the
// floors and a little up, so they run away ahead and rise into the arch.
func nearView(world *thing.World) *Built {
	world.Lamp(thing.Sun(maths.V(nearSunX, nearSunY, nearSunZ),
		maths.RGB(nearSunRed, nearSunGreen, nearSunBlue)))
	world.Lamp(film.Flood{Strength: maths.RGB(nearFloodRed,
		nearFloodGreen, nearFloodBlue)})
	from := maths.V(0, nearAside, orbitalRadius-nearAbove)
	look := maths.V(math.Cos(nearTurn), nearRise, -math.Sin(nearTurn))
	to := from.Add(look.Scale(nearTowards))
	return &Built{world: world, eye: maths.Subspace{Origin: from,
		Orientation: maths.LookAtUp(from, to, maths.V(0, 0, -1))}}
}

// farView lights the pair from the middle only, by zappy: a star of
// 9,000 km in its dyson swarm, its light coming out between the panels
// in shafts, and its two beams; and puts the eye inside the nebula,
// distance out and 25 degrees above their plane, its up along the plane
// so the hoops stand.
//
// zappy's light at the rings, 1,840 out, is about what the four
// spotlights it replaced gave: only just. Its beams keep their own time,
// so the built scene carries their clock.
func farView(world *thing.World, away float64, still bool) (*Built,
	error) {
	clock := new(time.Duration)
	middle := zappyDefined
	if still {
		middle.Turn = 0
	}
	if _, err := middle.Place(world, clock); err != nil {
		return nil, err
	}
	el := farElevation * maths.Degree
	from := maths.V(0, away*math.Sin(el), away*math.Cos(el))
	return &Built{world: world, eye: maths.Subspace{Origin: from,
		Orientation: maths.LookAtUp(from, maths.V(0, 0, 0),
			maths.V(1, 0, 0))}, clock: clock}, nil
}

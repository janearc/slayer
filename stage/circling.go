package stage

import (
	"fmt"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	scenebodies "github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/generate"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

// A body that circles a pair's lava drop is the drop's planet: the drop is
// its sun, so its sun's mass and light are the drop's, worked from the
// drop and not said twice; it stands on a circle round the drop at its
// orbit's distance, a year of Kepler's for the two masses; and it is
// locked, turning once a circuit, so the same face looks at the drop, and
// its clouds are laid about that face (see bodies.Body.Locking).

// pairNamed is the pair of a world by its name, or the refusal that says
// the world has none by it.
func pairNamed(said set.World, name string) (set.Pair, error) {
	for _, each := range said.Binaries() {
		if each.Name() == name {
			return each, nil
		}
	}
	return set.Pair{}, fmt.Errorf("there is no pair of stars named %q", name)
}

// dressed is a body as the set says it beyond its numbers, and the scale
// that turns its kilometres into the world's unit: its axis tilted, its
// aurora kept or taken off, and its radius where the set says one.
func dressed(each set.Body, body scenebodies.Body,
	said set.World) (scenebodies.Body, float64) {
	if tilt, tilted := each.Tilt(); tilted {
		body = body.Tilting(measure.Degrees(tilt))
	}
	if shown, says := each.Aurora(); says && !shown {
		body = body.WithoutAurora()
	}
	scale := metresPerKilometre / said.Unit()
	if radius, sized := each.Radius(); sized {
		scale = radius / body.Radius()
	}
	return body, scale
}

// circled puts a body that goes round a pair's lava drop into the world,
// on its circle, locked to the drop, shifted as the world is. It refuses a
// pair that is not there or has no drop, and what the generator refuses.
func circled(world *thing.World, said set.World, each set.Body,
	shift maths.Vec) error {
	pair, err := pairNamed(said, each.Circles())
	if err != nil {
		return err
	}
	placed, middle, err := pair.Placed(said.Unit())
	if err != nil {
		return err
	}
	if placed.Drop == nil {
		return fmt.Errorf("the pair %q has no lava drop to go round",
			each.Circles())
	}
	planet, _ := each.Generated()
	planet.Sun = generate.Sun{Mass: placed.Drop.Shine.Mass,
		Luminosity: placed.Drop.Shine.Luminosity.InSolarLuminosities()}
	made, err := generate.Make(planet)
	if err != nil {
		return err
	}
	if planet, err = stoodAt(planet); err != nil {
		return err
	}
	body, scale := dressed(each, made, said)
	drop := moved(middle, shift).Add(placed.Drop.At.Scale(1 / said.Unit()))
	orbit := goingRound(body, planet, drop, each.Phase(), said.Unit())
	away := drop.Sub(orbit.At(0).Origin)
	body = body.Locking(orbit.At(0).Orientation.In(away))
	return body.PlaceOn(world, orbit, scale, each.Detail())
}

// stoodAt is the planet with its distance said in metres, worked from the
// light it is given where it is given that and no distance, from the
// drop's own light, so that the circle it goes round on is where its
// sunlight comes to the Earth's and not where a number was chosen.
func stoodAt(planet generate.Planet) (generate.Planet, error) {
	where, err := planet.Distance()
	planet.Orbit.Distance, planet.Orbit.Flux = where, 0
	return planet, err
}

// goingRound is the circle a locked body goes round a drop at: the
// planet's orbit's distance, in the world's unit, once a year, starting
// at a phase in degrees, its face kept to the drop, its axis as it is
// tilted.
func goingRound(body scenebodies.Body, planet generate.Planet,
	drop maths.Vec, phase, unit float64) thing.Orbit {
	metres := planet.Orbit.Distance.InMetres()
	heavy := planet.Sun.Mass.InKilograms() + planet.Mass.InKilograms()
	return thing.Orbit{Centre: drop, Radius: metres / unit,
		Period: thing.Period(heavy, metres), Phase: phase * maths.Degree,
		Facing: maths.Rotation(maths.XAxis, body.Tilt()), Locked: true}
}

package template

import (
	"fmt"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/world"
)

// roughness is a look whose paints are how rough each class is, for a
// radar: plains smooth, rock and ice rough.
func roughness(by [world.Biomes]float64) world.Look {
	var look world.Look
	for biome, rough := range by {
		look.Paints[biome] = maths.Grey(rough)
	}
	return look
}

// look is a world's look: a paint for each biome, by name, and how
// dark the lines between tiles are. A biome left out is painted black.
func look(lines float64, paints [world.Biomes]maths.Light) world.Look {
	return world.Look{Paints: paints, Lines: lines}
}

// palette is a world's paints from the paint box's palette for it, as
// the lights a Look takes.
func palette(paints swatches.Palette) [world.Biomes]maths.Light {
	light := swatches.Light
	return [world.Biomes]maths.Light{
		world.Deep: light(paints.Deep), world.Sea: light(paints.Sea),
		world.Ice:   light(paints.Ice),
		world.Beach: light(paints.Beach), world.Desert: light(paints.Desert),
		world.Grass: light(paints.Grass), world.Forest: light(paints.Forest),
		world.Jungle: light(paints.Jungle), world.Tundra: light(paints.Tundra),
		world.Rock: light(paints.Rock), world.Snow: light(paints.Snow),
		world.Lava: light(paints.Lava),
		world.Ash:  light(paints.Ash),
	}
}

// twoTone paints a world in a pair's two lights, alternating down the
// biomes: deep, ice, desert, forest, tundra, snow and ash in the pale;
// sea, beach, grass, jungle, rock and lava in the dark. It is for a
// ground with no biomes to speak of, where the two keep the tiles apart.
func twoTone(pair swatches.Pair) [world.Biomes]maths.Light {
	pale, dark := swatches.Light(pair.Pale), swatches.Light(pair.Dark)
	return [world.Biomes]maths.Light{
		world.Deep: pale, world.Sea: dark, world.Ice: pale,
		world.Beach: dark, world.Desert: pale, world.Grass: dark,
		world.Forest: pale, world.Jungle: dark, world.Tundra: pale,
		world.Rock: dark, world.Snow: pale, world.Lava: dark,
		world.Ash: pale,
	}
}

// hours is a duration from hours.
func hours(amount float64) time.Duration {
	return time.Duration(amount * float64(time.Hour))
}

// deg is an angle said in degrees.
func deg(amount float64) measure.Angle { return measure.Degrees(amount) }

// rad is an angle said in radians.
func rad(amount float64) measure.Angle { return measure.Radians(amount) }

// km is a length said in the unit bodies speak.
func km(amount float64) bodies.Span { return bodies.Kilometres(amount) }

// kg is a mass said in kilograms.
func kg(amount float64) measure.Mass { return measure.Kilograms(amount) }

// smallMoon is a bare grey moon of a radius and a mass, on an orbit of
// a radius, starting a phase around it.
func smallMoon(name string, radius bodies.Span, mass measure.Mass,
	orbit bodies.Span, phase measure.Angle) (bodies.Moon, error) {
	body, err := bodies.NewBody(name, bodies.Rock, radius, mass)
	if err != nil {
		return bodies.Moon{}, err
	}
	return bodies.NewMoon(body, orbit, phase)
}

// groundedMoon is a moon with a made and painted ground, on its orbit.
func groundedMoon(name string, radius bodies.Span, mass measure.Mass,
	terrain world.Terrain, worn world.Look, orbit bodies.Span,
	phase measure.Angle) (bodies.Moon, error) {
	ground, err := bodies.NewGround(terrain, worn)
	if err != nil {
		return bodies.Moon{}, fmt.Errorf("%s: %w", name, err)
	}
	body, err := bodies.NewBody(name, bodies.Rock, radius, mass, ground)
	if err != nil {
		return bodies.Moon{}, err
	}
	return bodies.NewMoon(body, orbit, phase)
}

// giantAurora is the oval the gas giants share, in ultraviolet blues,
// at a latitude and lean, grown from a seed.
func giantAurora(low, high bodies.Span, lat, tilt measure.Angle,
	seed uint64) (bodies.Aurora, error) {
	return bodies.NewAurora(bodies.NewShell(low, high),
		lat, deg(2), tilt, maths.Percent(2),
		maths.Percent(50), seed,
		bodies.NewAuroraColours(
			swatches.Light(swatches.AuroraViolet),
			swatches.Light(swatches.AuroraBlue),
			swatches.Light(swatches.AuroraMauve)))
}

// giantGas is the body the gas giants share: opaque eight per cent of
// the radius down, forty optical depths from the tops to there, in the
// giant's own stripes.
func giantGas(stripes texture.Banded) (bodies.Gas, error) {
	return bodies.NewGas(maths.Percent(8), 40, stripes)
}

// bands is a banded texture from a run of lights, with the detail the
// giants share.
func bands(roil, fine, mottle float64, jets int, ls ...maths.Light) texture.Banded {
	return texture.Banded{Paints: ls, Roil: roil, Shear: 7, Grain: 0.08,
		Fine: fine, Jets: jets, Mottle: mottle}
}

// jupiterRadiusKm is jupiter's radius in kilometres, the length the
// hot jupiter's tail is said in.
const jupiterRadiusKm = 69911

// tailOf is one tail of a comet: how long and how wide at its base, how
// far it spreads and curves, how much light it takes, its colours, how
// ragged it is and the seed of its wisps.
type tailOf struct {
	length, base  bodies.Span
	spread, curve float64
	loss          bodies.Loss
	colour, glow  maths.Light
	wisp          maths.Portion
	seed          uint64
}

// made is the tail the description names, or the first refusal.
func (each tailOf) made() (bodies.Tail, error) {
	tail, err := bodies.NewTail(each.length, each.base, each.spread,
		each.curve, each.loss, each.colour, each.glow)
	if err != nil {
		return bodies.Tail{}, err
	}
	return tail.Wisped(each.wisp, each.seed)
}

// comet is a rock of a radius and mass named name, with a coma, a dust
// tail and an ion tail, grown from a seed. Every refusal carries the
// name.
func comet(name string, radius bodies.Span, mass measure.Mass,
	coma bodies.Loss, comaSize bodies.Span, comaColours [2]maths.Light,
	dust, ion tailOf, seed uint64) (bodies.Body, error) {
	head, err := bodies.NewComa(comaSize, coma, comaColours[0],
		comaColours[1])
	if err != nil {
		return bodies.Body{}, fmt.Errorf("%s: %w", name, err)
	}
	dustTail, err := dust.made()
	if err != nil {
		return bodies.Body{}, fmt.Errorf("%s: %w", name, err)
	}
	ionTail, err := ion.made()
	if err != nil {
		return bodies.Body{}, fmt.Errorf("%s: %w", name, err)
	}
	nucleus, err := bodies.NewBody(name, bodies.Rock, radius, mass, head,
		dustTail, ionTail)
	if err != nil {
		return bodies.Body{}, err
	}
	return nucleus.Seeded(seed), nil
}

package template

import (
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/world"
)

// roughness is a look whose paints are how rough each class is, for a
// radar: plains smooth, rock and ice rough.
func roughness(by [world.Biomes]float64) *world.Look {
	var look world.Look
	for b, r := range by {
		look.Paints[b] = maths.Grey(r)
	}
	return &look
}

// look is a world's look: a paint for each biome, by name, and how
// dark the lines between tiles are. A biome left out is painted black.
func look(lines float64, paints [world.Biomes]maths.Light) world.Look {
	return world.Look{Paints: paints, Lines: lines}
}

// palette is a world's paints from the paint box's palette for it, as
// the lights a Look takes.
func palette(p swatches.Palette) [world.Biomes]maths.Light {
	l := swatches.Light
	return [world.Biomes]maths.Light{
		world.Deep: l(p.Deep), world.Sea: l(p.Sea), world.Ice: l(p.Ice),
		world.Beach: l(p.Beach), world.Desert: l(p.Desert),
		world.Grass: l(p.Grass), world.Forest: l(p.Forest),
		world.Jungle: l(p.Jungle), world.Tundra: l(p.Tundra),
		world.Rock: l(p.Rock), world.Snow: l(p.Snow),
		world.Lava: l(p.Lava),
		world.Ash:  l(p.Ash),
	}
}

// twoTone paints a world in a pair's two lights, alternating down the
// biomes: deep, ice, desert, forest, tundra, snow and ash in the pale;
// sea, beach, grass, jungle, rock and lava in the dark. It is for a
// ground with no biomes to speak of, where the two keep the tiles apart.
func twoTone(p swatches.Pair) [world.Biomes]maths.Light {
	a, b := swatches.Light(p.Pale), swatches.Light(p.Dark)
	return [world.Biomes]maths.Light{
		world.Deep: a, world.Sea: b, world.Ice: a, world.Beach: b,
		world.Desert: a, world.Grass: b, world.Forest: a, world.Jungle: b,
		world.Tundra: a, world.Rock: b, world.Snow: a, world.Lava: b,
		world.Ash: a,
	}
}

// hours is a duration from hours.
func hours(h float64) time.Duration { return time.Duration(h * float64(time.Hour)) }

// deg is radians from degrees.
func deg(d float64) float64 { return d * maths.Degree }

// bands is a banded texture from a run of lights, with the detail the
// giants share.
func bands(roil, fine, mottle float64, jets int, ls ...maths.Light) texture.Banded {
	return texture.Banded{Paints: ls, Roil: roil, Shear: 7, Grain: 0.08,
		Fine: fine, Jets: jets, Mottle: mottle}
}

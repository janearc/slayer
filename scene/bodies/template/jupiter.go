package template

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/world"
)

// Jupiter is the giant: the bands in detail, the great red spot, and
// the four big moons.
func Jupiter() bodies.Body {
	stripes := bands(0.06, 0.05, 0.5, 10,
		swatches.Light(swatches.Cream), swatches.Light(swatches.Rust),
		swatches.Light(swatches.Cream),
		swatches.Light(swatches.Ochre), swatches.Light(swatches.Cream),
		swatches.Light(swatches.Rust),
		swatches.Light(swatches.Cream), swatches.Light(swatches.Ochre),
		swatches.Light(swatches.Cream),
		swatches.Light(swatches.Rust), swatches.Light(swatches.Cream))
	stripes.Spots = []texture.Spot{{Lat: -0.38, Lon: -1.3, Size: 0.16, Stretch: 1.7,
		Turn: 2.8, Paint: swatches.Light(swatches.Storm)}}
	stripes.Streets = []texture.Street{{Lat: -0.55, Count: 7, Size: 0.045, Turn: 1.4,
		Paint: swatches.Light(swatches.JupiterOval), Seed: 14}}
	stripes.Seed = 14
	sulfur := look(0.2, palette(swatches.Io))
	ice := look(0.1, twoTone(swatches.Europa))
	dark := look(0.2, twoTone(swatches.Ganymede))
	return bodies.Body{Name: "jupiter", Kind: bodies.Giant, Radius: 69911, Mass: 1.898e27,
		Day: hours(9.93), Tilt: deg(3.13),
		Gas: &bodies.Gas{Depth: 0.08, Opacity: 40, Bands: stripes},
		// the main oval sixteen degrees from the pole, ten degrees off
		// the spin, ultraviolet shown in blues
		Aurora: &bodies.Aurora{Low: 300, High: 1500, Lat: deg(74), Width: deg(2),
			Tilt: deg(10), Strength: 0.02, Curtains: 0.5, Seed: 14,
			Fringe: swatches.Light(swatches.AuroraViolet),
			Bottom: swatches.Light(swatches.AuroraBlue),
			Top:    swatches.Light(swatches.AuroraMauve)},
		Moons: []bodies.Moon{
			{Body: bodies.Body{Name: "io", Kind: bodies.Rock, Radius: 1821.6, Mass: 8.93e22,
				Ground: &bodies.Ground{Terrain: world.Terrain{Seed: 7, Ocean: 0, Plates: 9},
					Look: sulfur, Features: ioVolcanoes},
				// pele's plume: three hundred kilometres
				// high, eleven hundred across, at nineteen
				// south and two hundred and fifty-five west
				Plumes: []bodies.Plume{{Toward: maths.V(-0.24, -0.32, -0.92),
					Height: 300, Width: 1100, Dense: 0.006,
					Colour: swatches.Light(swatches.IoPlume)}}},
				Orbit: 421700},
			{Body: bodies.Body{Name: "europa", Kind: bodies.Rock, Radius: 1560.8, Mass: 4.8e22,
				Ground: &bodies.Ground{Terrain: world.Terrain{Seed: 8, Ocean: 0, Plates: 4},
					Look: ice}}, Orbit: 671034, Phase: 1.2},
			{Body: bodies.Body{Name: "ganymede", Kind: bodies.Rock, Radius: 2634.1, Mass: 1.48e23,
				Ground: &bodies.Ground{Terrain: world.Terrain{Seed: 9, Ocean: 0.4, Plates: 8},
					Look: dark}}, Orbit: 1070412, Phase: 2.5},
			{Body: bodies.Body{Name: "callisto", Kind: bodies.Rock, Radius: 2410.3, Mass: 1.08e23,
				Ground: &bodies.Ground{Terrain: world.Terrain{Seed: 10, Ocean: 0, Plates: 6},
					Look: dark}}, Orbit: 1882709, Phase: 4},
		},
	}
}

// ioVolcanoes puts a dozen live volcanoes on io, where its ground has
// room, by a fixed pattern: one with no land near it is left out, by
// design, so this never refuses.
func ioVolcanoes(w *world.World) ([]world.Feature, error) {
	var out []world.Feature
	step := w.Tiling.Spacing()
	for i := range 12 {
		k := float64(i)
		toward := maths.V(math.Cos(k*2.4), math.Sin(k*1.7)*0.8, math.Sin(k*2.4))
		tile, ok := w.Find(toward, world.Land)
		if !ok {
			continue
		}
		out = append(out, world.Volcano{At: w.Tiling.Centre(tile),
			Reach: 2.5 * step, Rise: 0.3})
	}
	return out, nil
}

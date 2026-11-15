package template

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/world"
)

// Jupiter is the giant: the bands in detail, the great red spot, and
// the four big moons.
func Jupiter() (bodies.Body, error) {
	gas, err := giantGas(jupiterStripes())
	if err != nil {
		return bodies.Body{}, fmt.Errorf("jupiter: %w", err)
	}
	// the main oval sixteen degrees from the pole, ten degrees off
	// the spin, ultraviolet shown in blues
	aurora, err := giantAurora(km(300), km(1500), deg(74), deg(10), 14)
	if err != nil {
		return bodies.Body{}, fmt.Errorf("jupiter: %w", err)
	}
	io, err := ioMoon()
	if err != nil {
		return bodies.Body{}, fmt.Errorf("jupiter: %w", err)
	}
	europa, err := groundedMoon("europa", km(1560.8), kg(4.8e22),
		world.Terrain{Seed: 8, Ocean: 0, Plates: 4},
		look(0.1, twoTone(swatches.Europa)), km(671034), rad(1.2))
	if err != nil {
		return bodies.Body{}, fmt.Errorf("jupiter: %w", err)
	}
	dark := look(0.2, twoTone(swatches.Ganymede))
	ganymede, err := groundedMoon("ganymede", km(2634.1), kg(1.48e23),
		world.Terrain{Seed: 9, Ocean: 0.4, Plates: 8}, dark,
		km(1070412), rad(2.5))
	if err != nil {
		return bodies.Body{}, fmt.Errorf("jupiter: %w", err)
	}
	callisto, err := groundedMoon("callisto", km(2410.3), kg(1.08e23),
		world.Terrain{Seed: 10, Ocean: 0, Plates: 6}, dark,
		km(1882709), rad(4))
	if err != nil {
		return bodies.Body{}, fmt.Errorf("jupiter: %w", err)
	}
	jupiter, err := bodies.NewBody("jupiter", bodies.Giant, km(jupiterRadiusKm),
		kg(1.898e27), gas, aurora, io, europa, ganymede, callisto)
	if err != nil {
		return bodies.Body{}, err
	}
	return jupiter.Turning(hours(9.93), deg(3.13)), nil
}

// jupiterStripes is jupiter's banding: the creams and rusts, the
// great red spot, and a street of white ovals.
func jupiterStripes() texture.Banded {
	stripes := bands(0.06, 0.05, 0.5, 10,
		swatches.Light(swatches.Cream), swatches.Light(swatches.Rust),
		swatches.Light(swatches.Cream),
		swatches.Light(swatches.Ochre), swatches.Light(swatches.Cream),
		swatches.Light(swatches.Rust),
		swatches.Light(swatches.Cream), swatches.Light(swatches.Ochre),
		swatches.Light(swatches.Cream),
		swatches.Light(swatches.Rust), swatches.Light(swatches.Cream))
	stripes.Spots = []texture.Spot{{Lat: -0.38, Lon: -1.3, Size: 0.16,
		Stretch: 1.7, Turn: 2.8,
		Paint: swatches.Light(swatches.Storm)}}
	stripes.Streets = []texture.Street{{Lat: -0.55, Count: 7,
		Size: 0.045, Turn: 1.4,
		Paint: swatches.Light(swatches.JupiterOval), Seed: 14}}
	stripes.Seed = 14
	return stripes
}

// ioMoon is io: sulfur ground, a dozen live volcanoes, and pele's
// plume, three hundred kilometres high and eleven hundred across, at
// nineteen south and two hundred and fifty-five west.
func ioMoon() (bodies.Moon, error) {
	ground, err := bodies.NewGround(
		world.Terrain{Seed: 7, Ocean: 0, Plates: 9},
		look(0.2, palette(swatches.Io)))
	if err != nil {
		return bodies.Moon{}, fmt.Errorf("io: %w", err)
	}
	ground = ground.Featured(ioVolcanoes)
	pele, err := bodies.NewPlume(maths.V(-0.24, -0.32, -0.92),
		km(300), km(1100), bodies.PerKilometre(0.006),
		swatches.Light(swatches.IoPlume))
	if err != nil {
		return bodies.Moon{}, fmt.Errorf("io: %w", err)
	}
	body, err := bodies.NewBody("io", bodies.Rock, km(1821.6),
		kg(8.93e22), ground, pele)
	if err != nil {
		return bodies.Moon{}, err
	}
	return bodies.NewMoon(body, km(421700), rad(0))
}

// ioVolcanoes puts a dozen live volcanoes on io, where its ground has
// room, by a fixed pattern: one with no land near it is left out, by
// design, so this never refuses.
func ioVolcanoes(w *world.World) ([]world.Feature, error) {
	var out []world.Feature
	step := w.Tiling.Spacing()
	for count := range 12 {
		number := float64(count)
		toward := maths.V(math.Cos(number*2.4),
			math.Sin(number*1.7)*0.8, math.Sin(number*2.4))
		tile, ok := w.Find(toward, world.Land)
		if !ok {
			continue
		}
		out = append(out, world.Volcano{At: w.Tiling.Centre(tile),
			Reach: 2.5 * step, Rise: 0.3})
	}
	return out, nil
}

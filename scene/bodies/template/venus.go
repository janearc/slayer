package template

import (
	"fmt"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/scene/world"
)

// Venus is a rock under an air so thick nothing of the ground shows: a
// pale yellow ball with a soft limb. The clouds pass radio, and the
// ground returns it by how rough it is, which is how magellan mapped it.
//
// The plains are smooth and dark, the highlands and the volcanoes rough
// and bright, maxwell the highest and maat the great volcano.
func Venus() (bodies.Body, error) {
	ground, err := bodies.NewGround(
		world.Terrain{Seed: 2, Ocean: 0, Plates: 8},
		look(0, twoTone(swatches.Venus)))
	if err != nil {
		return bodies.Body{}, fmt.Errorf("venus: %w", err)
	}
	ground = ground.Featured(venusFeatures).
		Radiating(thing.Spectrum{Returns: map[radiation.Range]float64{
			radiation.Radio(): 0.9}}).
		RadarRough(roughness([world.Biomes]float64{
			world.Deep: 0.1, world.Sea: 0.15, world.Ice: 0.5,
			world.Beach: 0.18, world.Desert: 0.15,
			world.Grass: 0.22, world.Forest: 0.3,
			world.Jungle: 0.35, world.Tundra: 0.4,
			world.Rock: 0.95, world.Snow: 0.8, world.Lava: 0.7,
			world.Ash: 0.6}))
	air, err := bodies.NewAir(km(15.9), bodies.PerKilometre(0.6),
		swatches.Light(swatches.VenusAir))
	if err != nil {
		return bodies.Body{}, fmt.Errorf("venus: %w", err)
	}
	air = air.Radiating(thing.Spectrum{
		Passes: map[radiation.Range]float64{radiation.Radio(): 1}})
	venus, err := bodies.NewBody("venus", bodies.Rock, km(6051.8),
		kg(4.867e24), ground, air)
	if err != nil {
		return bodies.Body{}, err
	}
	// venus turns backwards: the tilt past ninety says so, as pluto's
	// does, and the day is positive; a negative day about an axis
	// turned over would turn it forwards again
	return venus.Turning(hours(5832.5), deg(177.4)), nil
}

// venusFeatures raises maxwell, the highest plateau, and maat, the
// great dead volcano, where the made world has land for them.
func venusFeatures(w *world.World) ([]world.Feature, error) {
	step := w.Tiling.Spacing()
	maxwell, ok := w.Find(maths.V(0.2, 0.9, 0.4), world.Land)
	if !ok {
		return nil, fmt.Errorf("no land for maxwell")
	}
	maat, ok := w.Find(maths.V(0.9, 0.05, 0.4), world.Land)
	if !ok {
		return nil, fmt.Errorf("no land for maat")
	}
	return []world.Feature{
		world.Plateau{At: w.Tiling.Centre(maxwell), Reach: 3 * step,
			Top: 0.9},
		world.Volcano{At: w.Tiling.Centre(maat), Reach: 3 * step,
			Rise: 0.5, Dead: true},
	}, nil
}

package template

import (
	"fmt"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/world"
)

// Mars is rust and dust with white caps, a thin pink air, two small
// moons, and the biggest volcano and canyon there are.
func Mars() (bodies.Body, error) {
	ground, err := bodies.NewGround(
		world.Terrain{Seed: 4, Ocean: 0, Plates: 7, Cold: 0.25},
		look(0.3, palette(swatches.Mars)))
	if err != nil {
		return bodies.Body{}, fmt.Errorf("mars: %w", err)
	}
	ground = ground.Featured(marsFeatures)
	air, err := bodies.NewAir(km(11.1), bodies.PerKilometre(0.003),
		swatches.Light(swatches.MarsAir))
	if err != nil {
		return bodies.Body{}, fmt.Errorf("mars: %w", err)
	}
	phobos, err := smallMoon("phobos", km(11.1), kg(1.07e16), km(9376), rad(0))
	if err != nil {
		return bodies.Body{}, err
	}
	deimos, err := smallMoon("deimos", km(6.2), kg(1.5e15), km(23463), rad(2))
	if err != nil {
		return bodies.Body{}, err
	}
	mars, err := bodies.NewBody("mars", bodies.Rock, km(3389.5),
		kg(6.417e23), ground, air, phobos, deimos)
	if err != nil {
		return bodies.Body{}, err
	}
	return mars.Turning(hours(24.62), deg(25.19)), nil
}

// marsFeatures raises the great dead volcano on tharsis and cuts the
// canyon from its side, where the made world has land for them.
func marsFeatures(w *world.World) ([]world.Feature, error) {
	step := w.Tiling.Spacing()
	tharsis, ok := w.Find(maths.V(0.9, 0.3, 0.3), world.Land)
	if !ok {
		return nil, fmt.Errorf("no land for the volcano")
	}
	at := w.Tiling.Centre(tharsis)
	from := at.Add(maths.V(0.3, -0.1, -0.5)).Unit()
	return []world.Feature{
		world.Volcano{At: at, Reach: 4 * step, Rise: 0.6, Dead: true},
		world.Canyon{From: from,
			To:    from.Add(maths.V(0.5, 0, 0.2)).Unit(),
			Width: 1.2 * step, Depth: 0.5},
	}, nil
}

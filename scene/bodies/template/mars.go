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
func Mars() bodies.Body {
	return bodies.Body{Name: "mars", Kind: bodies.Rock, Radius: 3389.5, Mass: 6.417e23,
		Day: hours(24.62), Tilt: deg(25.19),
		Ground: &bodies.Ground{
			Terrain: world.Terrain{Seed: 4, Ocean: 0, Plates: 7, Cold: 0.25},
			Look:    look(0.3, palette(swatches.Mars)),
			Features: func(w *world.World) ([]world.Feature, error) {
				step := w.Tiling.Spacing()
				tharsis, ok := w.Find(maths.V(0.9, 0.3, 0.3), world.Land)
				if !ok {
					return nil, fmt.Errorf("no land for the volcano")
				}
				at := w.Tiling.Centre(tharsis)
				from := at.Add(maths.V(0.3, -0.1, -0.5)).Unit()
				return []world.Feature{
					world.Volcano{At: at, Reach: 4 * step, Rise: 0.6, Dead: true},
					world.Canyon{From: from, To: from.Add(maths.V(0.5, 0, 0.2)).Unit(),
						Width: 1.2 * step, Depth: 0.5},
				}, nil
			},
		},
		Air: &bodies.Air{Height: 11.1, Dense: 0.003,
			Colour: swatches.Light(swatches.MarsAir)},
		Moons: []bodies.Moon{
			{Body: bodies.Body{Name: "phobos", Kind: bodies.Rock, Radius: 11.1, Mass: 1.07e16}, Orbit: 9376},
			{Body: bodies.Body{Name: "deimos", Kind: bodies.Rock, Radius: 6.2, Mass: 1.5e15}, Orbit: 23463, Phase: 2},
		},
	}
}

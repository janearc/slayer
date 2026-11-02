package template

import (
	"fmt"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/world"
)

// Pluto is a dwarf of nitrogen ice and red tholin, with the bright plain
// of sputnik on it; charon is its partner, and the two go round a point
// between them, which Binary places.
func Pluto() bodies.Body {
	return bodies.Body{Name: "pluto", Kind: bodies.Rock, Radius: 1188.3, Mass: 1.303e22,
		Day: hours(153.3), Tilt: deg(122.5),
		Ground: &bodies.Ground{
			Terrain: world.Terrain{Seed: 18, Ocean: 0, Plates: 6, Cold: 0.4},
			Look:    look(0.2, palette(swatches.Pluto)),
			Features: func(w *world.World) ([]world.Feature, error) {
				heart, ok := w.Find(maths.V(0.2, 0.1, 1), world.Land)
				if !ok {
					return nil, fmt.Errorf("no land for the heart")
				}
				return []world.Feature{world.Patch{At: w.Tiling.Centre(heart),
					Reach: 5 * w.Tiling.Spacing(), Of: world.Ice}}, nil
			},
		},
	}
}

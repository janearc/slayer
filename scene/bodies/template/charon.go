package template

import (
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/world"
)

// Charon is pluto's partner, grey with a dark red cap.
func Charon() bodies.Body {
	return bodies.Body{Name: "charon", Kind: bodies.Rock, Radius: 606, Mass: 1.586e21,
		Day: hours(153.3),
		Ground: &bodies.Ground{
			Terrain: world.Terrain{Seed: 19, Ocean: 0, Plates: 5, Cold: 0.3},
			Look:    look(0.2, palette(swatches.Charon)),
		},
	}
}

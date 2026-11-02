package template

import (
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/world"
)

// Mercury is a bare grey rock, no air, no sea, a long slow day.
func Mercury() bodies.Body {
	return bodies.Body{Name: "mercury", Kind: bodies.Rock, Radius: 2439.7, Mass: 3.301e23,
		Day: hours(1407.6), Tilt: deg(0.03),
		Ground: &bodies.Ground{
			Terrain: world.Terrain{Seed: 1, Ocean: 0, Plates: 6, Cold: 0.3},
			Look:    look(0.3, palette(swatches.Mercury)),
		}}
}

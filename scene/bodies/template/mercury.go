package template

import (
	"fmt"

	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/world"
)

// Mercury is a bare grey rock, no air, no sea, a long slow day.
func Mercury() (bodies.Body, error) {
	ground, err := bodies.NewGround(
		world.Terrain{Seed: 1, Ocean: 0, Plates: 6, Cold: 0.3},
		look(0.3, palette(swatches.Mercury)))
	if err != nil {
		return bodies.Body{}, fmt.Errorf("mercury: %w", err)
	}
	mercury, err := bodies.NewBody("mercury", bodies.Rock, km(2439.7),
		kg(3.301e23), ground)
	if err != nil {
		return bodies.Body{}, err
	}
	return mercury.Turning(hours(1407.6), deg(0.03)), nil
}

package template

import (
	"fmt"

	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/world"
)

// Charon is pluto's partner, grey with a dark red cap.
func Charon() (bodies.Body, error) {
	ground, err := bodies.NewGround(
		world.Terrain{Seed: 19, Ocean: 0, Plates: 5, Cold: 0.3},
		look(0.2, palette(swatches.Charon)))
	if err != nil {
		return bodies.Body{}, fmt.Errorf("charon: %w", err)
	}
	charon, err := bodies.NewBody("charon", bodies.Rock, km(606),
		kg(1.586e21), ground)
	if err != nil {
		return bodies.Body{}, err
	}
	return charon.Turning(hours(153.3), deg(0)), nil
}

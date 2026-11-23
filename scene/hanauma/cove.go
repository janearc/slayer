package hanauma

import (
	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// New builds the Hānauma location with its measured relief and water recipe.
func New(wet bool) (*arid.Cove, error) {
	bed, err := groundForWater(wet)
	if err != nil {
		return nil, err
	}
	shore := []thing.Thing{bed}
	if !wet {
		return arid.New(shore, nil)
	}
	// Rays beyond the surveyed crop still travel through deep water.
	shore = append(shore, thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{
		Name: "deep-water continuation bed", Colour: maths.RGB(.02, .036, .052),
	}, Motion: thing.Still(maths.NewSubspace(maths.V(0, -215, 0), maths.Identity, 1))})
	return arid.New(shore, &arid.Water{
		Name: "Hanauma water", Extinction: maths.RGB(.24, .034, .019),
		Sea:    oceantex.Sea{Wind: 4.6, Fetch: 3400, Depth: 15, Toward: -1.20},
		Extent: 1450, Collar: 18, Samples: 481, Seed: 112,
	})
}

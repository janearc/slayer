package bay

import (
	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/scene/thing"
)

// NewAquaticParkCove builds the cove's water and optional mapped geography.
// The Golden Gate view adds the western shore and bridge.
func NewAquaticParkCove(grid Grid, landmarks, goldenGate bool) (*arid.Cove, error) {
	var shore []thing.Thing
	if landmarks || goldenGate {
		island, err := embeddedAlcatraz(grid)
		if err != nil {
			return nil, err
		}
		piers, err := Piers(grid)
		if err != nil {
			return nil, err
		}
		shore = append(island, piers...)
	}
	if goldenGate {
		land, err := goldenGateShore(grid)
		if err != nil {
			return nil, err
		}
		bridge, err := goldenGateBridge(grid)
		if err != nil {
			return nil, err
		}
		shore = append(shore, land)
		shore = append(shore, bridge...)
	}
	return arid.New(shore, &arid.Water{
		Name: "bay water", Extinction: maths.RGB(.91, .48, .43),
		Sea:    oceantex.Sea{Wind: 3.2, Fetch: 1100, Depth: 8, Toward: .65},
		Extent: 32, Collar: 8, Samples: 641, Seed: 53,
	})
}

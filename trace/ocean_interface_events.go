package trace

import (
	"fmt"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/scene/thing"
)

// validateOceanInterfaceEvents checks the event count, and that the ocean
// boundary is a visible clear exact infinite mean plane in the world.
func validateOceanInterfaceEvents(world *thing.World) error {
	ocean := world.OceanSurface
	if ocean == nil || ocean.InterfaceEvents == 0 {
		return nil
	}
	if ocean.InterfaceEvents < 0 || ocean.InterfaceEvents > thing.MaxOceanInterfaceEvents {
		return fmt.Errorf("trace: ocean interface-events must be from 0 to %d",
			thing.MaxOceanInterfaceEvents)
	}
	boundary := ocean.Boundary
	if boundary == nil || boundary.Hidden() || boundary.Material == nil ||
		boundary.Material.Through <= 0 || boundary.Material.Metal != nil {
		return fmt.Errorf("trace: ocean interface-events requires a visible clear ocean boundary")
	}
	if _, exact := boundary.Shape.(*arid.InfiniteMeanContinuation); !exact {
		return fmt.Errorf("trace: ocean interface-events requires the exact infinite mean-plane continuation")
	}
	for _, each := range world.Things {
		if each == boundary {
			return nil
		}
	}
	return fmt.Errorf("trace: ocean interface-events boundary is absent from the world")
}

// oceanEventBudget returns the interface-event limit for target. Only the
// ocean boundary has a budget; any other thing gets zero.
func (scene *staged) oceanEventBudget(target *thing.Thing) int {
	if target == scene.oceanEventBoundary {
		return scene.oceanEventLimit
	}
	return 0
}

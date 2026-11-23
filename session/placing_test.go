package session

import (
	"testing"

	"github.com/janearc/slayer/scene/thing"
)

// A draw tells its world the filter the set's film said; with no set to
// say, the grid, each ray at the middle of its own square of the pixel.
func TestADrawTellsItsWorldWhereRaysStart(t *testing.T) {
	world := thing.NewWorld()
	asked{filter: thing.TentFilter}.placing(world)
	if world.Filter != thing.TentFilter {
		t.Errorf("a film with the tent left the world with filter %d",
			world.Filter)
	}
	asked{}.placing(world)
	if world.Filter != thing.GridFilter {
		t.Errorf("no film left the world with filter %d", world.Filter)
	}
}

// Anti: a draw with no world is not told anything, and is left for its
// tracer to refuse; it does not stop here.
func TestADrawWithNoWorldIsLeftToItsTracer(t *testing.T) {
	asked{filter: thing.TentFilter}.placing(nil)
}

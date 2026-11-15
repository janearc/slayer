//go:build draw

package lantern_test

import (
	"testing"
	"time"

	"github.com/janearc/slayer/lantern"
	"github.com/janearc/slayer/scene/thing"
)

// Zappy placed puts its star, its two sheets of glass to every ring, its
// light and its beams in a world, and says how many panels.
func TestZappyPlaced(t *testing.T) {
	world := thing.NewWorld()
	clock := new(time.Duration)
	count, err := lantern.Zappy().Place(world, clock)
	if err != nil {
		t.Fatal(err)
	}
	defined := lantern.Zappy()
	if count != defined.Orbits*defined.Panels ||
		len(world.Things) != once+2*defined.Orbits ||
		len(world.Lamps) != 2 {
		t.Errorf("zappy placed %d panels, %d things, %d lamps", count,
			len(world.Things), len(world.Lamps))
	}
}

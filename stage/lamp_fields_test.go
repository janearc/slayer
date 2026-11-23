package stage

import (
	"os"
	"testing"

	"github.com/janearc/slayer/set"
)

// a set's lamp-fields light the world: one lamp for each lamp the field
// scatters, beside the lamps the set writes one by one.
func TestLampFieldsLightTheWorld(t *testing.T) {
	raw, err := os.ReadFile("../sets/restir-crowd-1k.yaml")
	if err != nil {
		t.Fatal(err)
	}
	made, err := set.Read("restir-crowd-1k.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	said, _ := made.World()
	world, err := World(said)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(world.Lamps); got != 1000 {
		t.Fatalf("%d lamps, want 1000", got)
	}
	// anti: the set's own lamps are not counted twice
	if len(said.Lamps()) != 0 {
		written := len(said.Lamps())
		t.Errorf("the set writes %d lamps one by one", written)
	}
}

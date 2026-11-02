package gpu

import (
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// the hall under a distant lamp instead of its point one, low and from
// the side, so shadows lie long across the floor and the volumes are lit
// from one way
func sunlitHall(t *testing.T) (*thing.World, maths.Subspace) {
	world, eye := hall(t)
	sun, err := emit.NewDistant(maths.V(0.6, 0.5, 0.3),
		maths.RGB(2.5, 2.3, 2))
	if err != nil {
		t.Fatal(err)
	}
	world.Lamps = []thing.Lamp{sun}
	return world, eye
}

// the gpu lights the hall by a distant lamp as slayer's tracer does: one
// way and one light at every point, its shadows from that way, endless
func TestTheGPULightsByADistantLampAsSlayerDoes(t *testing.T) {
	world, eye := sunlitHall(t)
	against(t, world, eye)
}

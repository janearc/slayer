package trace

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// otherSun is a sun from the other side, to change a world's lamp to.
func otherSun() thing.Lamp {
	return thing.Sun(maths.V(-1, 2, 1), maths.Grey(1))
}

// lampAt is the light a lamp sends to a point above the floor, which
// tells one sun from another.
func lampAt(lamp thing.Lamp) maths.Light {
	_, _, light := lamp.Light(maths.V(0, 1, 0), maths.Up)
	return light
}

// A staged world keeps its own list of lamps. Changing the world's
// first lamp where it lies, and adding two more, leaves the staged
// world with the one lamp it was staged with.
func TestAStagedWorldKeepsItsOwnLamps(t *testing.T) {
	world := thing.NewWorld()
	world.Lamp(thing.Sun(sunwards, maths.Grey(0.5)))
	want := lampAt(world.Lamps[0])
	scene := stage(world, 0)
	world.Lamps[0] = otherSun()
	world.Lamp(otherSun())
	world.Lamp(otherSun())
	if len(scene.lamps) != 1 {
		t.Fatalf("the staged world has %d lamps, not the one it was "+
			"staged with", len(scene.lamps))
	}
	if got := lampAt(scene.lamps[0]); got != want {
		t.Errorf("the staged world's lamp gives %v, and gave %v when "+
			"it was staged", got, want)
	}
}

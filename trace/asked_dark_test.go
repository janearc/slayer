package trace

import (
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// spotHeight is how high over the floor the spot of these tests hangs
const spotHeight = 6

// spotWidth and spotSoft are the spot's cone and its soft rim: narrow,
// so that pointed up it sends nothing to the floor
const (
	spotWidth = 12 * maths.Degree
	spotSoft  = 4 * maths.Degree
)

// spotted is a floor under a flood, a spot pointed along aim, and a sun,
// the spot between the two so the sun's index follows it
func spotted(aim maths.Vec) *thing.World {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Colour: maths.Grey(0.5)}})
	world.Lamp(film.Flood{Strength: maths.Grey(0.2)})
	world.Lamp(emit.Lit{At: maths.V(0, spotHeight, 0),
		Strength: maths.Grey(60), Toward: aim.Unit(), Width: spotWidth,
		Soft: spotSoft})
	toward := maths.V(0.3, 1, 0.2).Unit()
	light := maths.Grey(1)
	world.Lamp(thing.Sun(toward, light))
	return world
}

// indicesAsked is the lamps a point on the floor under the spot asks,
// by index, in order
func indicesAsked(world *thing.World) []int {
	scene := stage(world, 0)
	var indices []int
	for _, one := range scene.asked(nil, maths.V(0.2, 0, 0.1),
		maths.V(0, 1, 0), byLight) {
		indices = append(indices, one.index)
	}
	return indices
}

// a spot pointed up sends nothing to the floor, so a point there asks
// the flood and the sun and not the spot, as the kernel asks them
func TestALampThatSendsNothingIsNotAsked(t *testing.T) {
	pointed := maths.V(0, 1, 0)
	got := indicesAsked(spotted(pointed))
	if len(got) != 2 || got[0] != 0 || got[1] != 2 {
		t.Errorf("under a spot pointed up, the lamps asked are %v; "+
			"want the flood and the sun, [0 2]", got)
	}
}

// the anti-test: the same spot pointed down lights the floor and is
// asked, between the flood and the sun
func TestALampThatSendsLightIsAsked(t *testing.T) {
	pointed := maths.V(0, -1, 0)
	got := indicesAsked(spotted(pointed))
	if len(got) != 3 || got[1] != 1 {
		t.Errorf("under a spot pointed down, the lamps asked are %v; "+
			"want all three, [0 1 2]", got)
	}
}

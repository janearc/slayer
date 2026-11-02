package trace

import (
	"strings"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// burning is a fire on a dark floor under a black sky: a flame's body
// half a metre across at the middle, its soot at quintiere and
// mccaffrey's 0.8 a metre made thicker to show, and nothing else lit.
func burning() *thing.World {
	body, err := flame.NewBody(flame.Fire{Radius: 0.5, Height: 1.2,
		Absorption: 3, Hottest: measure.Kelvin(1300),
		Edge: measure.Kelvin(900), Seed: 3}, 1e3)
	if err != nil {
		panic(err)
	}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Name: "floor", Colour: maths.Grey(0.3)}})
	world.AddVolume(thing.Volume{Medium: body,
		Material: &thing.Material{Name: "flame", Shine: body.Shine()}})
	return world
}

// a fire shows: the pixel through its middle is brighter than one
// beside it, and a replay of it tells each step through the flame with
// how hot the flame was there.
func TestAFireShowsAndTellsItsHeat(t *testing.T) {
	world := burning()
	eye := lookingAt(maths.V(0, 1, 4), maths.V(0, 0.4, 0))
	const width, height = 16, 12
	whole := sampled(world, eye, film.Pinhole{}, 0, width, height, 1)
	middle, beside := whole.At(width/2, height/2), whole.At(1, height/2)
	if !(middle.R > 10*beside.R) {
		t.Errorf("the fire's middle %v is not brighter than beside it %v",
			middle, beside)
	}
	_, steps, err := Replay(world, eye, film.Pinhole{}, 0, width, height, 1,
		width/2, height/2)
	if err != nil {
		t.Fatal(err)
	}
	strides := 0
	for _, step := range steps {
		if step.What == "a step through the flame" {
			strides++
			if !strings.Contains(step.Said, "kelvin=") {
				t.Fatalf("a step through the flame without its heat: %s",
					step.Said)
			}
		}
	}
	if strides == 0 {
		t.Error("the replay told no step through the flame")
	}
}

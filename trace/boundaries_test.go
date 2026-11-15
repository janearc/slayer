package trace

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestSceneExtentAndInvalidFieldsAreBounded(t *testing.T) {
	world := thing.NewWorld()
	if Spread(world, 0) != 0 {
		t.Fatal("empty scene has extent")
	}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 2}})
	if math.Abs(Spread(world, 0)-math.Sqrt(48)) > 1e-6 {
		t.Fatal("finite bounds lost")
	}
	world.AddField(thing.Field{Mass: 1, Reach: 10, Spin: 2})
	if len(placeFields(world, 0)) != 0 {
		t.Fatal("impossible field accepted")
	}
}
func TestEqualSurfaceDistancesKeepTheEarlierObject(t *testing.T) {
	world := thing.NewWorld()
	for range 2 {
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}})
	}
	seen := gathered(place(world, 0))
	best := nearest{index: 0, dist: 2}
	got := seen.closer(best, 1, maths.Ray{At: maths.V(0, 0, 3), Dir: maths.V(0, 0, -1)}, 0)
	if got.index != 0 || got.dist != 2 {
		t.Fatal("stable tie broken", got)
	}
}
func TestMeshReplayIncludesFaceAmbientLightAndHaze(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: aCube(t), Material: &thing.Material{Name: "cube", Colour: maths.Grey(.5)}})
	world.Lamp(fill)
	world.Haze = thing.Haze{Density: .2, Colour: maths.Grey(.2)}
	pixel, steps, err := Replay(world, lookingAt(maths.V(0, 0, 3), maths.Vec{}),
		film.Pinhole{}, 0, 1, 1, 1, 0, 0)
	if err != nil || pixel.R <= 0 {
		t.Fatal("mesh replay failed", pixel, err)
	}
	var said strings.Builder
	for _, step := range steps {
		said.WriteString(step.What + " " + step.Said + "\n")
	}
	for _, want := range []string{"cube", "face", "from everywhere", "the haze"} {
		if !strings.Contains(said.String(), want) {
			t.Fatal("replay lost physical step", want, said.String())
		}
	}
	ray := maths.Ray{At: maths.V(1, 2, 3), Dir: maths.Up}
	if !strings.Contains(exactly(ray), "from (1, 2, 3)") || exactly(nil) != "none" || exactly(&ray) != "maths.Ray" {
		t.Fatal("replay values lost")
	}
}

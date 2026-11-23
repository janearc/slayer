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

// balls is a crowd of balls at the world's middle, of the radii given,
// and how far along a ray down z from three away each is met: three
// less its radius.
func balls(radii ...float64) (crowd, maths.Ray) {
	world := thing.NewWorld()
	for _, radius := range radii {
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: radius}})
	}
	return gathered(place(world, 0)),
		maths.Ray{At: maths.V(0, 0, 3), Dir: maths.V(0, 0, -1)}
}

// Two balls of one size at one place are met at one distance, two along
// the ray. At the same distance the earlier thing is kept, whichever of
// the two was the best so far: the later one does not replace the
// earlier, and the earlier does replace the later, which it can only do
// if it is met at the best's own distance and not passed over.
func TestEqualSurfaceDistancesKeepTheEarlierObject(t *testing.T) {
	seen, ray := balls(1, 1)
	met, found := seen.first(ray, 0)
	if !found || !within(met.dist, 2, 1e-12) {
		t.Fatalf("the balls are met at %v, found %v, want two along",
			met.dist, found)
	}
	kept := seen.closer(nearest{index: 0, dist: met.dist}, 1, ray, 0)
	if kept.index != 0 || kept.dist != met.dist {
		t.Errorf("the later ball replaced the earlier at a tie: %+v", kept)
	}
	taken := seen.closer(nearest{index: 1, dist: met.dist}, 0, ray, 0)
	if taken.index != 0 || taken.dist != met.dist {
		t.Errorf("the earlier ball did not replace the later at a tie: "+
			"%+v", taken)
	}
}

// Anti: only a tie goes by which came first. A ball met nearer replaces
// the best whatever its place in the crowd, and one met further off
// does not: a ball of half the size at the same place is met half a
// unit further along.
func TestANearerObjectReplacesTheBestAndAFartherOneDoesNot(t *testing.T) {
	seen, ray := balls(0.5, 1)
	nearer := seen.closer(nearest{index: 0, dist: 2.5}, 1, ray, 0)
	if nearer.index != 1 || !within(nearer.dist, 2, 1e-12) {
		t.Errorf("the nearer, later ball did not replace the best: %+v",
			nearer)
	}
	farther := seen.closer(nearest{index: 1, dist: 2}, 0, ray, 0)
	if farther.index != 1 || farther.dist != 2 {
		t.Errorf("the farther, earlier ball replaced the best: %+v",
			farther)
	}
}
func TestMeshReplayIncludesFaceAmbientLightAndHaze(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: aCube(t), Material: &thing.Material{Name: "cube", Colour: maths.Grey(.5)}})
	world.Lamp(fill)
	world.Haze = thing.Haze{Density: .2, Colour: maths.Grey(.2)}
	from := maths.V(0, 0, 3)
	pixel, steps, err := Replay(world, lookingAt(from, maths.Vec{}),
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

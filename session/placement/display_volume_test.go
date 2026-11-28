package placement

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

func TestWorkingVolumeBoundsUsePlacedFrameAndIgnoreUnsupportedMedia(t *testing.T) {
	world := thing.NewWorld()
	fizz := medium.Fizz{Radius: .4, Height: 2, Thick: 4}
	clear, err := medium.NewClearCloud(fizz)
	if err != nil {
		t.Fatal(err)
	}
	motion := thing.Keyed{Length: 2 * time.Second, Ease: "linear", Frames: []thing.Keyframe{
		{Share: 0, At: maths.V(1, 0, -5)}, {Share: 1, At: maths.V(3, 0, -5)},
	}}
	world.AddVolume(thing.Volume{Medium: medium.Air{}})
	world.AddVolume(thing.Volume{Medium: fizz, Attrs: thing.Attrs{Hidden: true}})
	world.AddVolume(thing.Volume{Medium: clear, Motion: motion})
	world.AddVolume(thing.Volume{Medium: medium.Fizz{Radius: math.Inf(1), Height: 2}})
	bounds := WorkingVolumeBounds(world, time.Second)
	if len(bounds) != 1 {
		t.Fatalf("bounds = %+v, expected one supported volume", bounds)
	}
	want := DisplayVolumeBound{Kind: "Fizz", Box: fizz.Bounds(), At: motion.At(time.Second)}
	if bounds[0] != want || bounds[0].At.Origin.X != 2 {
		t.Fatalf("bounds = %+v, want %+v", bounds[0], want)
	}
}

func TestProjectVolumeBoundClipsOnlyExistingEdges(t *testing.T) {
	scene := &DisplayScene{Width: 100, Height: 100, Lens: film.Pinhole{Wide: math.Pi / 2, Pixel: 1}}
	box := DisplayVolumeBound{Kind: "Fizz",
		Box: maths.NewBox(maths.V(-1, -1, -1), maths.V(1, 1, 1)),
		At:  maths.NewSubspace(maths.V(0, 0, -5), maths.Identity, 1)}
	if got := len(scene.ProjectVolumeBound(box)); got != 12 {
		t.Fatalf("fully visible box has %d edges, want 12", got)
	}
	box.At.Origin.X = 4.5
	edges := scene.ProjectVolumeBound(box)
	if len(edges) != 11 || math.Abs(edges[4][1][0]-1) > 1e-12 {
		t.Fatalf("right edge clipping = %+v", edges)
	}
	box.At.Origin = maths.V(0, 0, 2)
	if got := len(scene.ProjectVolumeBound(box)); got != 0 {
		t.Fatalf("box behind camera has %d edges", got)
	}
}

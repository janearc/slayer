package manufacture

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// testPlume is a hot plume from the origin, carried down a light wind for
// a kilometre, under no lid.
func testPlume() Plume {
	return Plume{Name: "smoke", Turn: maths.Identity, Wind: 5,
		Reach: 1000, Mouth: 2, Exit: 10, Hot: 600, Air: 290, Rate: 100,
		Extinction: 0.01, Asymmetry: 0.5, Exponent: 1, Albedo: 0.3}
}

func TestAPlumesBoxRunsDownTheWindForItsWholeReach(t *testing.T) {
	box, err := testPlume().Extent()
	if err != nil {
		t.Fatal(err)
	}
	if box.Min.X > 0 || box.Max.X < 1000 {
		t.Errorf("a plume's box %+v does not run from 0 to its reach", box)
	}
	if !(box.Max.Z > 0 && box.Min.Z < 0 && box.Max.Y > 0) {
		t.Errorf("a plume's box %+v has no width or no height", box)
	}
}

func TestALidHoldsAPlumesBoxUnderIt(t *testing.T) {
	lidded := testPlume()
	lidded.Lid, lidded.Lidded = 50, true
	free, err := testPlume().Extent()
	if err != nil {
		t.Fatal(err)
	}
	held, err := lidded.Extent()
	if err != nil {
		t.Fatal(err)
	}
	if !(held.Max.Y <= 50 && held.Max.Y < free.Max.Y) {
		t.Errorf("a lid of 50 left the box %v high, free %v", held.Max.Y,
			free.Max.Y)
	}
}

func TestAPlumeWithNoWindHasNoBox(t *testing.T) {
	calm := testPlume()
	calm.Wind = 0
	if _, err := calm.Extent(); err == nil {
		t.Error("a plume in no wind was given a box")
	}
	if err := calm.Staged(thing.NewWorld()); err == nil {
		t.Error("a plume in no wind was staged")
	}
}

func TestAPlumeStagesAsItsStretchesInOrder(t *testing.T) {
	world := thing.NewWorld()
	if err := testPlume().Staged(world); err != nil {
		t.Fatal(err)
	}
	if len(world.Volumes) != plumeStretches {
		t.Fatalf("%d volumes, not %d", len(world.Volumes), plumeStretches)
	}
	first, last := world.Volumes[0].Medium.Bounds(),
		world.Volumes[len(world.Volumes)-1].Medium.Bounds()
	if first.Max.X >= last.Max.X || last.Max.X != 1000 {
		t.Errorf("stretches end at %v and %v, not out to 1000",
			first.Max.X, last.Max.X)
	}
}

func TestAPlumeAheadOfAnEyeIsSeenAndOneBehindIsNot(t *testing.T) {
	ahead := maths.Subspace{Origin: maths.V(-10, 0, -50),
		Orientation: maths.Identity, Scale: 1}
	got, err := testPlume().seen(sow(0), ahead, lookingDown())
	if err != nil || !got.Visible {
		t.Errorf("a plume ahead is %+v (%v)", got, err)
	}
	behind := maths.Subspace{Origin: maths.V(0, 0, 5000),
		Orientation: maths.Identity, Scale: 1}
	got, err = testPlume().seen(sow(0), behind, lookingDown())
	if err != nil || got.Visible {
		t.Errorf("a plume behind is %+v (%v)", got, err)
	}
}

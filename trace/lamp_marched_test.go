package trace

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// marchedPoints is how many points the tests of a step's picks look at.
const marchedPoints = 64

// pointAmongWindows is the nth point the tests look at: along the street
// of windows, a few metres from its wall.
func pointAmongWindows(nth int) maths.Vec {
	return maths.V(float64(nth)*0.7, 1, 5)
}

// TestAStepThroughAMediumPicksOneLampWhereASurfacePicksMany says that of
// more lamps than are lit one by one, with no tree, a step through a
// medium takes a single lamp and a surface takes several: a step's picks
// are summed along its ray, and several a step only buy shadow rays.
func TestAStepThroughAMediumPicksOneLampWhereASurfacePicksMany(
	t *testing.T) {
	lamps := streetOfWindows(t)[1:]
	if len(lamps) <= lampsEvery {
		t.Fatalf("%d lamps, want more than %d", len(lamps), lampsEvery)
	}
	scene := &staged{lamps: lamps}
	for nth := range marchedPoints {
		point := pointAmongWindows(nth)
		step := scene.marched(nil, point, maths.Vec{}, byLight)
		surface := scene.asked(nil, point, maths.Vec{}, byLight)
		if !takesOne(step) || len(surface) < 2 {
			t.Fatalf("point %v: a step takes %d lamps, a surface %d, want "+
				"one and several", point, len(step), len(surface))
		}
	}
}

// takesOne is whether the picks of a step are a single lamp.
func takesOne(picks []lit) bool { return len(picks) == 1 }

// TestAStepThatAskedAsASurfaceDoesIsToldFromOne is the anti-test of the
// above: the picks a step would make if it asked as a surface does, by
// lampsAsked, are several lamps, and the check that a step takes one says
// so at every point.
func TestAStepThatAskedAsASurfaceDoesIsToldFromOne(t *testing.T) {
	scene := &staged{lamps: streetOfWindows(t)[1:]}
	for nth := range marchedPoints {
		point := pointAmongWindows(nth)
		asked := scene.askedPicks(nil, point, maths.Vec{}, byLight, -1,
			lampsAsked, 0)
		if takesOne(asked) {
			t.Fatalf("point %v: %d picks of a surface's asks are taken "+
				"for a step's one", point, len(asked))
		}
	}
}

// TestAStageOfManyStandingLampsHasALampTree says that staging a world of
// more standing lamps than lampsEvery builds the tree its picks walk, and
// leaves the sun outside it.
func TestAStageOfManyStandingLampsHasALampTree(t *testing.T) {
	world := suns(0)
	for _, lamp := range streetOfWindows(t) {
		world.Lamp(lamp)
	}
	scene := stage(world, 0)
	if scene.tree == nil || len(scene.tree.lamps) != windows {
		t.Fatalf("a tree of %v for %d windows", scene.tree, windows)
	}
	if len(scene.others) != 1 {
		t.Errorf("%d lamps outside the tree, want the sun", len(scene.others))
	}
}

// TestAStageOfFewLampsHasNoLampTree is the anti-test of the above: lamps
// lampsEvery or fewer are lit one by one, with no tree.
func TestAStageOfFewLampsHasNoLampTree(t *testing.T) {
	world := suns(0)
	for _, lamp := range streetOfWindows(t)[:lampsEvery] {
		world.Lamp(lamp)
	}
	if scene := stage(world, 0); scene.tree != nil {
		t.Errorf("a tree for %d lamps", lampsEvery)
	}
}

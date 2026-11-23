package gpu

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// windowsAndSuns is crowdWindows lit windows, enough to be a lamp tree,
// and suns distant lamps, each from a way of its own.
func windowsAndSuns(t *testing.T, suns int) []thing.Lamp {
	t.Helper()
	var lamps []thing.Lamp
	for each := range crowdWindows {
		at := maths.V(float64(each%13)*4, float64(each/13)*3, 0)
		strength := maths.Grey(1 + float64(each)/crowdWindows)
		lamps = append(lamps, emit.Lit{
			At:       at,
			Beyond:   0.1,
			Strength: strength})
	}
	for each := range suns {
		way := maths.V(float64(each), 1, 0)
		sun, err := emit.NewDistant(way, maths.Grey(1))
		if err != nil {
			t.Fatal(err)
		}
		lamps = append(lamps, sun)
	}
	return lamps
}

// worldOfLamps is a world of one sphere under the lamps.
func worldOfLamps(lamps []thing.Lamp) *thing.World {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}})
	for _, lamp := range lamps {
		world.Lamp(lamp)
	}
	return world
}

// A step through a medium asks every distant lamp, and a lamp outside the
// tree, and the nodes of the cut, of the kernel's asks, which drop what
// does not fit. A world with a lamp tree and one distant lamp more than
// the asks have room for is refused, by the numbers, at the encoder and
// among Refusals, and not drawn with a lamp left out.
func TestADistantLampTooManyForTheAsksIsRefused(t *testing.T) {
	lamps := windowsAndSuns(t, lampsDistantMost+1)
	var refused errTooManyDistant
	err := lampCrowdOf(lamps).refuse(lamps)
	if !errors.As(err, &refused) || refused.distant != lampsDistantMost+1 {
		t.Fatalf("%d distant lamps were not refused by name: %v",
			lampsDistantMost+1, err)
	}
	for _, number := range []int{refused.distant, lampsDistantMost,
		lampsEvery, lampsMarched, lampSplitMost} {
		if !strings.Contains(err.Error(), fmt.Sprint(number)) {
			t.Errorf("the refusal does not say %d: %v", number, err)
		}
	}
	world := worldOfLamps(lamps)
	if _, err := encode(world, maths.Subspace{}, nil, 0, 8, 8); !errors.As(err,
		&refused) {
		t.Errorf("the encoder took the world: %v", err)
	}
	found := Refusals(world, 0)
	if len(found) != 1 || found[0].Part != partWorld ||
		found[0].Reason != err.Error() {
		t.Errorf("refusals %+v, want the world's, saying %q", found, err)
	}
}

// Anti-test: a world with as many distant lamps as the asks have room for
// beside the rest of a step's lamps is not refused, by the encoder or by
// Refusals, and neither is one with more distant lamps than that and no
// lamp tree, which a step asks lampsMarched lamps of and no more.
func TestDistantLampsThatFitTheAsksAreNotRefused(t *testing.T) {
	lamps := windowsAndSuns(t, lampsDistantMost)
	if err := lampCrowdOf(lamps).refuse(lamps); err != nil {
		t.Errorf("%d distant lamps were refused: %v", lampsDistantMost, err)
	}
	world := worldOfLamps(lamps)
	if _, err := encode(world, maths.Subspace{}, nil, 0, 8, 8); err != nil {
		t.Errorf("the encoder refused %d distant lamps: %v",
			lampsDistantMost, err)
	}
	if found := Refusals(world, 0); len(found) != 0 {
		t.Errorf("%d distant lamps have refusals: %+v", lampsDistantMost,
			found)
	}
	few := windowsAndSuns(t, lampsEvery)[crowdWindows-lampTreeLeast:]
	if crowd := lampCrowdOf(few); len(crowd.nodes) != 0 {
		t.Fatalf("%d standing lamps made a tree", lampTreeLeast)
	}
	if err := lampCrowdOf(few).refuse(few); err != nil {
		t.Errorf("%d distant lamps with no tree were refused: %v",
			lampsEvery, err)
	}
}

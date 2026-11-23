package trace

import (
	"testing"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// What a replay says of a look that follows a mirror, and of one that
// follows a bend: the legs that cost a look more work.
const (
	mirrorLeg = "a mirror, by fresnel's share"
	bendLeg   = "a clear surface, bent by snell"
)

// seabedWorld is a flat tagged sea, with events interface events
// allowed, over a seabed five metres under it, glossy or matte, under a
// sky of one.
func seabedWorld(t *testing.T, events int, glossy bool) *thing.World {
	t.Helper()
	field, err := shape.NewHeightField(make([]float64, 9), 3, 4)
	if err != nil {
		t.Fatal(err)
	}
	sea, err := arid.NewInfiniteMeanContinuation(field, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	boundary := &thing.Thing{Shape: sea, Material: &thing.Material{
		Through: 1, Gloss: 1, Index: 1.333}}
	bed := &thing.Material{}
	if glossy {
		bed.Gloss = 1
	}
	world := thing.NewWorld()
	world.Things = append(world.Things, boundary)
	bedAt := resting(maths.V(0, -5, 0))
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: bed,
		Motion: bedAt})
	world.Sky = even(maths.Grey(1))
	world.OceanSurface = &thing.OceanSurface{Boundary: boundary,
		InterfaceEvents: events}
	return world
}

// legsFollowed is how many mirror and bend legs one look from above the
// sea, nearly straight down, follows through the world.
func legsFollowed(world *thing.World) int {
	scene := stage(world, 0)
	scene.walked = &walk{}
	shade(scene, look{ray: maths.Ray{At: maths.V(0.3, 5, 0.2),
		Dir: maths.V(0, -1, 0)}})
	legs := 0
	for _, step := range scene.walked.steps {
		if step.What == mirrorLeg || step.What == bendLeg {
			legs++
		}
	}
	return legs
}

// seabedLegsBound is the most legs a look from the air can follow with
// events interface events and bounces of one, over a planar seabed. Each
// event on a path sends two legs on, a mirror and a bend, and a path takes
// at most events of them: two, four and so on to two to the events, a sum
// of two to the events plus one, less two. The seabed adds one mirror leg
// and no more: it is followed only on a path that has bent and not
// mirrored, its bounce then unspent, and from the air that is the one
// chain of bends down to the seabed; every other path has mirrored at the
// sea and its bounce is spent. It is met at one event, three of three,
// and an upper bound beyond: five of seven at two, five of 511 at eight.
// With several seabeds or a sea of waves it would not hold.
func seabedLegsBound(events int) int {
	return 1<<(events+1) - 2 + 1
}

// The cost of a glossy seabed under the tagged sea is a number of legs a
// look, and it is worked. Over a flat sea, from nearly straight down,
// with two events or more a look follows five: the sea's mirror and its
// bend from above, the seabed's one mirror, and the sea's mirror and
// bend from below, where the reflection reaches the seabed again with
// its bounce spent and goes no further. With one event the sea has none
// left from below, and it is three. More events than two add no legs on
// a flat sea, and every count is within the bound worked from the events:
// three of three at one event, five of seven at two. A matte seabed costs
// two legs, so a glossy one costs five for two at two events or more on a
// flat sea, and three for two at one.
func TestAGlossySeabedUnderTheTaggedSeaCostsAtMostFiveLegsALook(
	t *testing.T) {
	for events, want := range map[int]int{1: 3, 2: 5, 8: 5} {
		got := legsFollowed(seabedWorld(t, events, true))
		if got != want || got > seabedLegsBound(events) {
			t.Errorf("with %d events a look follows %d legs, want %d, "+
				"at most %d", events, got, want, seabedLegsBound(events))
		}
	}
}

// Anti: a matte seabed has no mirror to follow, and the look follows only
// the sea's own two legs from above: the seabed's mirror is what the
// other three are.
func TestAMatteSeabedUnderTheTaggedSeaCostsTwoLegsALook(t *testing.T) {
	for _, events := range []int{1, 2, 8} {
		if got := legsFollowed(seabedWorld(t, events, false)); got != 2 {
			t.Errorf("with %d events a look over a matte seabed follows "+
				"%d legs, want 2", events, got)
		}
	}
}

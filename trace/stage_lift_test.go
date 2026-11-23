package trace

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// A look a surface sends onward starts a millionth of the largest
// length to hand off the surface, along its normal: the scene's spread,
// the point's distance from the origin, or how far the look had come.
// At 3, 0, 4, five from the origin: come a hundred, it is lifted a
// ten-thousandth; in a scene two thousand across, two thousandths; and
// with neither, five millionths.
func TestALookSentOnwardIsLiftedAMillionthOfTheLargestLength(t *testing.T) {
	point := maths.V(3, 0, 4)
	for name, each := range map[string]struct {
		spread float64
		from   maths.Vec
		want   float64
	}{
		"come a hundred":      {0, maths.V(3, 100, 4), 1e-4},
		"a scene of 2000":     {2000, maths.V(3, 100, 4), 2e-3},
		"its own five alone":  {0, point, 5e-6},
		"a spread under five": {2, point, 5e-6},
	} {
		scene := &staged{spread: each.spread}
		got := scene.liftedAlong(point, maths.Up, each.from)
		if !within(got.Y, each.want, 1e-12*each.want) || got.X != 3 ||
			got.Z != 4 {
			t.Errorf("%s: lifted to %v, want %v up", name, got, each.want)
		}
	}
}

// Anti: the look's own travel only ever adds to the lift. A look that
// has come no way is lifted as lifted lifts a point, and never less.
func TestALookThatHasComeNoWayIsLiftedAsAPointIs(t *testing.T) {
	scene, point := &staged{spread: 7}, maths.V(3, 0, 4)
	if got, want := scene.liftedAlong(point, maths.Up, point),
		scene.lifted(point, maths.Up); got != want || !(got.Y > 0) {
		t.Errorf("a look come no way is lifted to %v, a point to %v", got,
			want)
	}
}

// balled is a world of one ball two across each way from its middle,
// whose box is four a side: a reach of the square root of 48.
func balled() *thing.World {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 2}})
	return world
}

// A world that says the length its ray origins are measured by is taken
// at its word, whatever its things reach: half a metre, under one, and
// fifty, over the ball's own reach.
func TestADeclaredRayOriginReferenceIsTheSpread(t *testing.T) {
	for _, metres := range []float64{0.5, 50} {
		world := balled()
		said, err := thing.NewRayOriginReference(measure.Metres(metres))
		if err != nil {
			t.Fatal(err)
		}
		world.RayOriginReference = said
		if got := Spread(world, 0); got != metres {
			t.Errorf("a reference of %v gives a spread of %v", metres, got)
		}
	}
}

// Anti: a world that says none is measured by its things' reach, the
// diagonal of the ball's box, to the hair a thing's box is widened by.
func TestWithNoReferenceTheSpreadIsTheThingsReach(t *testing.T) {
	if got := Spread(balled(), 0); !within(got, math.Sqrt(48), 1e-6) {
		t.Errorf("with no reference the spread is %v, want the square "+
			"root of 48", got)
	}
}

// A march goes on while any one channel still lets more than faint
// through: each channel alone is enough, and two with the third dark.
func TestAMarchGoesOnWhileAnyChannelShows(t *testing.T) {
	over := 2 * faint
	for _, keep := range []maths.Light{maths.RGB(over, 0, 0),
		maths.RGB(0, over, 0), maths.RGB(0, 0, over),
		maths.RGB(over, over, 0), maths.RGB(over, 0, over)} {
		if !shows(keep) {
			t.Errorf("a march that keeps %v was stopped", keep)
		}
	}
}

// Anti: a march stops once no channel keeps more than faint, and faint
// itself is not more than faint, in any channel.
func TestAMarchStopsWhenNoChannelShowsMoreThanFaint(t *testing.T) {
	for _, keep := range []maths.Light{{}, maths.Grey(faint),
		maths.RGB(faint, 0, 0), maths.RGB(0, faint, 0),
		maths.RGB(0, 0, faint), maths.Grey(faint / 2)} {
		if shows(keep) {
			t.Errorf("a march that keeps %v went on", keep)
		}
	}
}

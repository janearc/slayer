package trace

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// stagedBall is a staged world of one marched ball whose middle is away
// along x in its own space, placed back by as much, so it sits at the
// world's middle.
func stagedBall(away float64) *staged {
	world := thing.NewWorld()
	ballAt := resting(maths.V(-away, 0, 0))
	world.Add(thing.Thing{
		Shape:  farBall{middle: maths.V(away, 0, 0), radius: 1},
		Motion: ballAt})
	return stage(world, 0)
}

// The lift off a point of a marched surface is at least twice what the
// march takes as touching, worked in the shape's own space where the
// march works: the ball's own middle is a thousand off, so the march
// takes a point within a thousandth as met.
func TestTheLiftClearsWhatAMarchTakesAsTouching(t *testing.T) {
	scene := stagedBall(1000)
	point := maths.V(0, 0, 1)
	local := point.Sub(maths.V(-1000, 0, 0))
	ball := farBall{middle: maths.V(1000, 0, 0), radius: 1}
	reach := marchReach(ball.Bounds())
	least := leaveTouches * touched(reach, local.Len())
	if got := scene.lifting(point); got < least {
		t.Errorf("the lift is %g, less than %g", got, least)
	}
}

// Anti: where nothing is marched the lift is the lift's share of the
// larger of the spread and the point's distance, as it was.
func TestTheLiftIsItsOwnShareWhereNothingIsMarched(t *testing.T) {
	scene := stage(thing.NewWorld(), 0)
	point := maths.V(0, 0, 1000)
	if got, want := scene.lifting(point), lift*1000; got != want {
		t.Errorf("the lift is %g, want %g", got, want)
	}
	if len(scene.marching) > 0 {
		t.Errorf("a world with nothing in it marches something")
	}
}

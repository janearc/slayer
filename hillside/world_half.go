package hillside

import (
	"fmt"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
)

// halved adds the set's world half to the world: its boxes, spheres,
// lamps, spots, floods, suns, bodies, wisps and fizzes, built as the
// stage builds them, in the frame of the set's things.
//
// the world half says its places as a map does, east, north and up in
// metres, which the set reader turns to the world's space as x east, y up
// and z south: (east, up, minus north).
//
// a thing the set stands at an east and a north is at (east minus the
// window's west, the ground's height minus its lowest point, the window's
// north minus north). so a place of the world half lands in the same
// frame when moved by (minus the window's west, minus the lowest point,
// the window's north). a set on no terrain has no window, and its world
// half is not moved.
//
// a world half has its own lamps, which are added to the set's light and
// to the tally. its sky is not hung over the sky the set's light made.
func (making *staging) halved(read set.Set) error {
	said, has := read.World()
	if !has {
		return nil
	}
	before := len(making.world.Lamps)
	if err := stage.Add(making.world, said, making.rebasing(read)); err != nil {
		return fmt.Errorf("%s: %w", making.given.set, err)
	}
	making.made.lamps += len(making.world.Lamps) - before
	return nil
}

// rebasing is how far the world half's places move to stand in the
// frame of the things on the ground: the window's west and lowest point
// taken off, its north put on. a set with no ground has no window, and its
// places stay where they are.
func (making *staging) rebasing(read set.Set) maths.Vec {
	if _, grounded := read.Ground(); !grounded {
		return unshifted
	}
	return maths.V(-making.west, -making.lowest, making.north)
}

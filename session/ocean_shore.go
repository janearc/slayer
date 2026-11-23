package session

// A sea with no edge over a set's ground is told where the ground is, so
// its water lies only where the open sea reaches and its waves stand no
// taller than the depth carries. This file finds the sea and the ground
// in the staged world and puts them together.

import (
	"errors"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/set"
)

// shored puts the set's ground under a sea with no edge (arid.Sea.Surf,
// Ashore with what the bottom does to the waves), and says in the tally
// and the draw's notes what that changed. A sea with a square, a set with
// no ground and a world half with no sea are left as they are.
//
// The ground's first sample is at the world's origin, x east and z south,
// its heights less the scene's lowest; the sea's middle is the ocean's
// Mean. halved has refused a world unit other than metres, so the two
// frames differ by a move and no scale.
//
// A ground with a near window is refused: the sea meets one window, and
// the finer ground's shore is not the one the coarse window says.
func (making *staging) shored(read set.Set) error {
	ocean := making.world.OceanSurface
	if _, grounded := read.Ground(); !grounded || ocean == nil ||
		ocean.Boundary == nil {
		return nil
	}
	sea, open := ocean.Boundary.Shape.(*arid.Sea)
	if !open {
		return nil
	}
	if making.near != nil {
		return errors.New("a sea with no edge meets one window of ground, " +
			"and this terrain has a near window")
	}
	middle := ocean.Mean
	shored, surfline, err := sea.Surf(madeSea(read), making.ground,
		-middle.X, -middle.Z, middle.Y+making.lowest)
	if err != nil {
		return err
	}
	ocean.Boundary.Shape = shored
	making.made.sea = surfline.String()
	making.given.cached = append(making.given.cached,
		cacheNote{what: making.made.sea, told: true})
	return nil
}

// madeSea is the sea the set's ocean says, as oceantex makes its bands
// from, for the surf's wind, peak and height. A set with no world or no
// ocean says a calm sea, which has no surf.
func madeSea(read set.Set) oceantex.Sea {
	said, has := read.World()
	if !has {
		return oceantex.Sea{}
	}
	surface, has := said.OceanSurface()
	if !has {
		return oceantex.Sea{}
	}
	wind, fetch, depth, toward := surface.Sea()
	return oceantex.Sea{Wind: wind, Fetch: fetch, Depth: depth,
		Toward: toward}
}
